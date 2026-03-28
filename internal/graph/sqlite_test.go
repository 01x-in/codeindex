package graph_test

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *graph.SQLiteStore {
	t.Helper()
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	t.Cleanup(func() { store.Close() })
	return store
}

func setupFileTestStore(t *testing.T) *graph.SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := graph.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	t.Cleanup(func() { store.Close() })
	return store
}

func TestMigrate(t *testing.T) {
	store := setupTestStore(t)
	// Migrate is idempotent.
	assert.NoError(t, store.Migrate())
}

func TestUpsertAndGetNode(t *testing.T) {
	store := setupTestStore(t)

	node := graph.Node{
		Name:      "handleRequest",
		Kind:      "fn",
		FilePath:  "src/api/handler.ts",
		LineStart: 24,
		LineEnd:   28,
		ColStart:  0,
		ColEnd:    1,
		Exported:  true,
		Language:  "typescript",
	}

	id, err := store.UpsertNode(node)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	got, err := store.GetNode(id)
	require.NoError(t, err)
	assert.Equal(t, "handleRequest", got.Name)
	assert.Equal(t, "fn", got.Kind)
	assert.Equal(t, "src/api/handler.ts", got.FilePath)
	assert.True(t, got.Exported)
}

func TestUpsertNodeUpdatesExisting(t *testing.T) {
	store := setupTestStore(t)

	// Insert a node.
	node := graph.Node{
		Name:      "myFunc",
		Kind:      "fn",
		FilePath:  "a.ts",
		LineStart: 10,
		LineEnd:   15,
		ColStart:  0,
		ColEnd:    1,
		Exported:  false,
		Language:  "typescript",
		Signature: "(): void",
	}

	_, err := store.UpsertNode(node)
	require.NoError(t, err)

	// Upsert same node with updated fields.
	node.LineEnd = 20
	node.Exported = true
	node.Signature = "(x: number): string"

	_, err = store.UpsertNode(node)
	require.NoError(t, err)

	// Should reuse the same row (not create a duplicate).
	nodes, err := store.FindNodesByName("myFunc")
	require.NoError(t, err)
	assert.Len(t, nodes, 1, "upsert should not create duplicates")

	// Verify updated fields.
	got := nodes[0]
	assert.Equal(t, 20, got.LineEnd)
	assert.True(t, got.Exported)
	assert.Equal(t, "(x: number): string", got.Signature)
}

func TestFindNodesByName(t *testing.T) {
	store := setupTestStore(t)

	_, err := store.UpsertNode(graph.Node{Name: "foo", Kind: "fn", FilePath: "a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)
	_, err = store.UpsertNode(graph.Node{Name: "foo", Kind: "fn", FilePath: "b.ts", LineStart: 10, LineEnd: 15, Language: "typescript"})
	require.NoError(t, err)
	_, err = store.UpsertNode(graph.Node{Name: "bar", Kind: "fn", FilePath: "c.ts", LineStart: 1, LineEnd: 3, Language: "typescript"})
	require.NoError(t, err)

	results, err := store.FindNodesByName("foo")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestFindNodesByFile(t *testing.T) {
	store := setupTestStore(t)

	_, err := store.UpsertNode(graph.Node{Name: "a", Kind: "fn", FilePath: "src/a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)
	_, err = store.UpsertNode(graph.Node{Name: "b", Kind: "type", FilePath: "src/a.ts", LineStart: 10, LineEnd: 12, Language: "typescript"})
	require.NoError(t, err)
	_, err = store.UpsertNode(graph.Node{Name: "c", Kind: "fn", FilePath: "src/b.ts", LineStart: 1, LineEnd: 3, Language: "typescript"})
	require.NoError(t, err)

	results, err := store.FindNodesByFile("src/a.ts")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestUpsertEdgeAndQuery(t *testing.T) {
	store := setupTestStore(t)

	id1, err := store.UpsertNode(graph.Node{Name: "caller", Kind: "fn", FilePath: "a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)
	id2, err := store.UpsertNode(graph.Node{Name: "callee", Kind: "fn", FilePath: "b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)

	err = store.UpsertEdge(graph.Edge{SourceID: id1, TargetID: id2, Kind: "calls", FilePath: "a.ts", Line: 3})
	require.NoError(t, err)

	from, err := store.GetEdgesFrom(id1, "calls")
	require.NoError(t, err)
	assert.Len(t, from, 1)
	assert.Equal(t, id2, from[0].TargetID)

	to, err := store.GetEdgesTo(id2, "calls")
	require.NoError(t, err)
	assert.Len(t, to, 1)
	assert.Equal(t, id1, to[0].SourceID)
}

func TestDeleteFileData(t *testing.T) {
	store := setupTestStore(t)

	id1, err := store.UpsertNode(graph.Node{Name: "fn1", Kind: "fn", FilePath: "target.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)
	_, err = store.UpsertNode(graph.Node{Name: "fn2", Kind: "fn", FilePath: "other.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)

	err = store.SetFileMetadata(graph.FileMetadata{FilePath: "target.ts", ContentHash: "abc", Language: "typescript", IndexStatus: "ok"})
	require.NoError(t, err)

	err = store.DeleteFileData("target.ts")
	require.NoError(t, err)

	nodes, err := store.FindNodesByFile("target.ts")
	require.NoError(t, err)
	assert.Len(t, nodes, 0)

	_, err = store.GetNode(id1)
	assert.Error(t, err)

	// Other file's nodes remain.
	other, err := store.FindNodesByFile("other.ts")
	require.NoError(t, err)
	assert.Len(t, other, 1)
}

func TestCascadeDeleteEdgesOnNodeDelete(t *testing.T) {
	store := setupTestStore(t)

	id1, err := store.UpsertNode(graph.Node{Name: "a", Kind: "fn", FilePath: "a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)
	id2, err := store.UpsertNode(graph.Node{Name: "b", Kind: "fn", FilePath: "b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	require.NoError(t, err)

	err = store.UpsertEdge(graph.Edge{SourceID: id1, TargetID: id2, Kind: "calls", FilePath: "a.ts", Line: 3})
	require.NoError(t, err)

	// Delete file a.ts, which removes node id1.
	err = store.DeleteFileData("a.ts")
	require.NoError(t, err)

	// Edges referencing id1 should be gone.
	edges, err := store.GetEdgesTo(id2, "")
	require.NoError(t, err)
	assert.Len(t, edges, 0, "cascade delete should remove edges when source node is deleted")
}

func TestFileMetadata(t *testing.T) {
	store := setupTestStore(t)

	meta := graph.FileMetadata{
		FilePath:    "src/a.ts",
		ContentHash: "sha256abc",
		Language:    "typescript",
		NodeCount:   5,
		EdgeCount:   3,
		IndexStatus: "ok",
	}

	err := store.SetFileMetadata(meta)
	require.NoError(t, err)

	got, err := store.GetFileMetadata("src/a.ts")
	require.NoError(t, err)
	assert.Equal(t, "sha256abc", got.ContentHash)
	assert.Equal(t, 5, got.NodeCount)

	all, err := store.GetAllFileMetadata()
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestGetNeighborhood(t *testing.T) {
	store := setupTestStore(t)

	id1, _ := store.UpsertNode(graph.Node{Name: "a", Kind: "fn", FilePath: "a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	id2, _ := store.UpsertNode(graph.Node{Name: "b", Kind: "fn", FilePath: "b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	id3, _ := store.UpsertNode(graph.Node{Name: "c", Kind: "fn", FilePath: "c.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})

	store.UpsertEdge(graph.Edge{SourceID: id1, TargetID: id2, Kind: "calls", FilePath: "a.ts", Line: 3})
	store.UpsertEdge(graph.Edge{SourceID: id2, TargetID: id3, Kind: "calls", FilePath: "b.ts", Line: 2})

	nodes, edges, err := store.GetNeighborhood(id1, 1, nil)
	require.NoError(t, err)
	assert.Len(t, nodes, 2) // a and b (depth 1)
	assert.Len(t, edges, 1)

	nodes2, edges2, err := store.GetNeighborhood(id1, 2, nil)
	require.NoError(t, err)
	assert.Len(t, nodes2, 3) // a, b, and c (depth 2)
	assert.Len(t, edges2, 2)
}

func TestConcurrentReads(t *testing.T) {
	// Use file-based DB for concurrent access (in-memory has connection isolation issues).
	store := setupFileTestStore(t)

	// Populate some data.
	for i := 0; i < 20; i++ {
		_, err := store.UpsertNode(graph.Node{
			Name:      "func" + string(rune('A'+i)),
			Kind:      "fn",
			FilePath:  "concurrent.ts",
			LineStart: i * 10,
			LineEnd:   i*10 + 5,
			Language:  "typescript",
		})
		require.NoError(t, err)
	}

	// Run concurrent reads.
	var wg sync.WaitGroup
	errs := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nodes, err := store.FindNodesByFile("concurrent.ts")
			if err != nil {
				errs <- err
				return
			}
			if len(nodes) != 20 {
				errs <- assert.AnError
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent read error: %v", err)
	}
}
