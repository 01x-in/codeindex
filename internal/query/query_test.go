package query_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/hash"
	"github.com/01x/codeindex/internal/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupQueryEngine(t *testing.T) (*query.Engine, *graph.SQLiteStore, string) {
	t.Helper()

	dir := t.TempDir()
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	t.Cleanup(func() { store.Close() })

	engine := query.NewEngine(store, dir)
	return engine, store, dir
}

func populateTestGraph(t *testing.T, store *graph.SQLiteStore, dir string) {
	t.Helper()

	// Create fixture files on disk (for staleness checks).
	os.MkdirAll(filepath.Join(dir, "src"), 0755)

	content1 := []byte("export function formatDate(date: Date): string {}")
	os.WriteFile(filepath.Join(dir, "src/utils.ts"), content1, 0644)

	content2 := []byte("import { formatDate } from './utils';")
	os.WriteFile(filepath.Join(dir, "src/handler.ts"), content2, 0644)

	// Populate graph.
	id1, err := store.UpsertNode(graph.Node{
		Name: "formatDate", Kind: "fn", FilePath: "src/utils.ts",
		LineStart: 1, LineEnd: 3, Exported: true, Language: "typescript",
		Signature: "(date: Date): string",
	})
	require.NoError(t, err)

	id2, err := store.UpsertNode(graph.Node{
		Name: "Config", Kind: "type", FilePath: "src/utils.ts",
		LineStart: 5, LineEnd: 8, Exported: true, Language: "typescript",
	})
	require.NoError(t, err)

	id3, err := store.UpsertNode(graph.Node{
		Name: "handleRequest", Kind: "fn", FilePath: "src/handler.ts",
		LineStart: 3, LineEnd: 10, Exported: true, Language: "typescript",
	})
	require.NoError(t, err)

	// handleRequest calls formatDate.
	store.UpsertEdge(graph.Edge{SourceID: id3, TargetID: id1, Kind: "calls", FilePath: "src/handler.ts", Line: 5})
	// handleRequest imports formatDate.
	store.UpsertEdge(graph.Edge{SourceID: id3, TargetID: id1, Kind: "imports", FilePath: "src/handler.ts", Line: 1})

	// Set file metadata (fresh).
	store.SetFileMetadata(graph.FileMetadata{
		FilePath: "src/utils.ts", ContentHash: hash.Bytes(content1),
		Language: "typescript", NodeCount: 2, IndexStatus: "ok",
	})
	store.SetFileMetadata(graph.FileMetadata{
		FilePath: "src/handler.ts", ContentHash: hash.Bytes(content2),
		Language: "typescript", NodeCount: 1, IndexStatus: "ok",
	})

	_ = id2
}

func TestGetFileStructure(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	fs, meta, err := engine.GetFileStructure("src/utils.ts")
	require.NoError(t, err)

	assert.Equal(t, "src/utils.ts", fs.File)
	assert.False(t, fs.Stale, "freshly indexed file should not be stale")
	assert.Len(t, fs.Symbols, 2) // formatDate + Config
	assert.Empty(t, meta.StaleFiles)

	// Verify symbol details.
	var found bool
	for _, s := range fs.Symbols {
		if s.Name == "formatDate" {
			assert.Equal(t, "fn", s.Kind)
			assert.True(t, s.Exported)
			assert.Contains(t, s.Signature, "(date: Date)")
			found = true
		}
	}
	assert.True(t, found, "should find formatDate symbol")
}

func TestGetFileStructure_EmptyFile(t *testing.T) {
	engine, _, _ := setupQueryEngine(t)

	fs, _, err := engine.GetFileStructure("nonexistent.ts")
	require.NoError(t, err)
	assert.Empty(t, fs.Symbols)
}

func TestFindSymbol(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	results, _, err := engine.FindSymbol("formatDate", "")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "fn", results[0].Kind)
	assert.Equal(t, "src/utils.ts", results[0].File)
}

func TestFindSymbol_WithKindFilter(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Should find Config as type.
	results, _, err := engine.FindSymbol("Config", "type")
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Should NOT find Config as fn.
	results, _, err = engine.FindSymbol("Config", "fn")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestFindSymbol_NotFound(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	results, _, err := engine.FindSymbol("nonexistent", "")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestGetReferences(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	refs, _, err := engine.GetReferences("formatDate")
	require.NoError(t, err)
	assert.Len(t, refs, 2) // calls + imports from handleRequest

	kinds := map[string]bool{}
	for _, r := range refs {
		kinds[r.Kind] = true
		assert.Equal(t, "src/handler.ts", r.File)
	}
	assert.True(t, kinds["calls"])
	assert.True(t, kinds["imports"])
}

func TestGetReferences_NoRefs(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	refs, _, err := engine.GetReferences("Config")
	require.NoError(t, err)
	assert.Len(t, refs, 0) // Nothing references Config
}

func TestGetReferences_NotFound(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	refs, _, err := engine.GetReferences("nonexistent")
	require.NoError(t, err)
	assert.Len(t, refs, 0)
}

func TestStaleFlagInResults(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Modify handler.ts to make it stale.
	os.WriteFile(filepath.Join(dir, "src/handler.ts"), []byte("// modified content"), 0644)

	results, meta, err := engine.FindSymbol("handleRequest", "")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Stale, "modified file should be flagged as stale")
	assert.Contains(t, meta.StaleFiles, "src/handler.ts")
}
