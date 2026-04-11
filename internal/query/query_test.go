package query_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x-in/codeindex/internal/graph"
	"github.com/01x-in/codeindex/internal/hash"
	"github.com/01x-in/codeindex/internal/query"
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

// === GetCallers tests (M4-S1) ===

func TestGetCallers_BasicChain(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// formatDate is called by handleRequest.
	callers, meta, err := engine.GetCallers("formatDate", 3)
	require.NoError(t, err)
	assert.Len(t, callers, 1)
	assert.Equal(t, "handleRequest", callers[0].Name)
	assert.Equal(t, "fn", callers[0].Kind)
	assert.Equal(t, "src/handler.ts", callers[0].File)
	assert.Equal(t, 1, callers[0].Depth)
	assert.False(t, callers[0].Stale)
	assert.Empty(t, meta.StaleFiles)
}

func TestGetCallers_DeepChain(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)

	// Build a chain: d -> c -> b -> a
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	for _, name := range []string{"a.ts", "b.ts", "c.ts", "d.ts"} {
		content := []byte("function " + name)
		os.WriteFile(filepath.Join(dir, "src/"+name), content, 0644)
		store.SetFileMetadata(graph.FileMetadata{
			FilePath: "src/" + name, ContentHash: hash.Bytes(content),
			Language: "typescript", NodeCount: 1, IndexStatus: "ok",
		})
	}

	idA, _ := store.UpsertNode(graph.Node{Name: "funcA", Kind: "fn", FilePath: "src/a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idB, _ := store.UpsertNode(graph.Node{Name: "funcB", Kind: "fn", FilePath: "src/b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idC, _ := store.UpsertNode(graph.Node{Name: "funcC", Kind: "fn", FilePath: "src/c.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idD, _ := store.UpsertNode(graph.Node{Name: "funcD", Kind: "fn", FilePath: "src/d.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})

	store.UpsertEdge(graph.Edge{SourceID: idB, TargetID: idA, Kind: "calls", FilePath: "src/b.ts", Line: 3})
	store.UpsertEdge(graph.Edge{SourceID: idC, TargetID: idB, Kind: "calls", FilePath: "src/c.ts", Line: 3})
	store.UpsertEdge(graph.Edge{SourceID: idD, TargetID: idC, Kind: "calls", FilePath: "src/d.ts", Line: 3})

	// Depth 2: should find funcB (depth 1) and funcC (depth 2), but NOT funcD.
	callers, _, err := engine.GetCallers("funcA", 2)
	require.NoError(t, err)
	assert.Len(t, callers, 2)

	nameSet := map[string]int{}
	for _, c := range callers {
		nameSet[c.Name] = c.Depth
	}
	assert.Equal(t, 1, nameSet["funcB"])
	assert.Equal(t, 2, nameSet["funcC"])
	_, hasFuncD := nameSet["funcD"]
	assert.False(t, hasFuncD, "funcD should not appear at depth 2")

	// Depth 3: should find all three.
	callers3, _, err := engine.GetCallers("funcA", 3)
	require.NoError(t, err)
	assert.Len(t, callers3, 3)
}

func TestGetCallers_CycleDetection(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)

	// Build a cycle: a -> b -> c -> a
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	for _, name := range []string{"a.ts", "b.ts", "c.ts"} {
		content := []byte("function " + name)
		os.WriteFile(filepath.Join(dir, "src/"+name), content, 0644)
		store.SetFileMetadata(graph.FileMetadata{
			FilePath: "src/" + name, ContentHash: hash.Bytes(content),
			Language: "typescript", NodeCount: 1, IndexStatus: "ok",
		})
	}

	idA, _ := store.UpsertNode(graph.Node{Name: "cycleA", Kind: "fn", FilePath: "src/a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idB, _ := store.UpsertNode(graph.Node{Name: "cycleB", Kind: "fn", FilePath: "src/b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idC, _ := store.UpsertNode(graph.Node{Name: "cycleC", Kind: "fn", FilePath: "src/c.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})

	// a calls b, b calls c, c calls a (cycle)
	store.UpsertEdge(graph.Edge{SourceID: idA, TargetID: idB, Kind: "calls", FilePath: "src/a.ts", Line: 2})
	store.UpsertEdge(graph.Edge{SourceID: idB, TargetID: idC, Kind: "calls", FilePath: "src/b.ts", Line: 2})
	store.UpsertEdge(graph.Edge{SourceID: idC, TargetID: idA, Kind: "calls", FilePath: "src/c.ts", Line: 2})

	// GetCallers for cycleB — who calls cycleB? → cycleA calls cycleB. Then who calls cycleA? → cycleC calls cycleA.
	// Then who calls cycleC? → cycleB calls cycleC, but cycleB is the starting node (visited), so stop.
	callers, _, err := engine.GetCallers("cycleB", 10)
	require.NoError(t, err)

	// Should not infinite loop; should have at most 2 results (cycleA, cycleC).
	assert.LessOrEqual(t, len(callers), 2)
	nameSet := map[string]bool{}
	for _, c := range callers {
		nameSet[c.Name] = true
	}
	assert.True(t, nameSet["cycleA"], "cycleA should appear as a caller of cycleB")
}

func TestGetCallers_NotFound(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	callers, _, err := engine.GetCallers("nonexistent", 3)
	require.NoError(t, err)
	assert.Len(t, callers, 0)
}

func TestGetCallers_NoCaller(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// handleRequest has no callers in the test graph.
	callers, _, err := engine.GetCallers("handleRequest", 3)
	require.NoError(t, err)
	assert.Len(t, callers, 0)
}

func TestGetCallers_DefaultDepth(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Passing 0 should use default depth (3).
	callers, _, err := engine.GetCallers("formatDate", 0)
	require.NoError(t, err)
	assert.Len(t, callers, 1) // handleRequest at depth 1
}

func TestGetCallers_StaleCaller(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Make handler.ts stale.
	os.WriteFile(filepath.Join(dir, "src/handler.ts"), []byte("// modified"), 0644)

	callers, meta, err := engine.GetCallers("formatDate", 3)
	require.NoError(t, err)
	assert.Len(t, callers, 1)
	assert.True(t, callers[0].Stale, "stale caller should be flagged")
	assert.Contains(t, meta.StaleFiles, "src/handler.ts")
}

func TestGetCallers_MaxDepthClamped(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Depth > 10 should be clamped to 10 and not panic.
	callers, _, err := engine.GetCallers("formatDate", 100)
	require.NoError(t, err)
	assert.Len(t, callers, 1) // Still just handleRequest
}
