package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/01x/codeindex/internal/mcp"
	"github.com/01x/codeindex/internal/query"
	"github.com/01x/codeindex/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEnd_FullWorkflow tests: init -> index -> query -> modify -> detect stale -> reindex -> query updated.
func TestEndToEnd_FullWorkflow(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "ts-project")

	// Step 1: Detect config (simulating init).
	cfg, _, err := config.LoadOrDetect(fixtureDir)
	require.NoError(t, err)
	assert.Contains(t, cfg.Languages, "typescript")

	// Step 2: Open graph store.
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	// Step 3: Index all files.
	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "typescript")

	results, err := idx.IndexAll()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 3, "should index at least 3 files")

	t.Logf("Indexed %d files", len(results))
	totalNodes := 0
	for _, r := range results {
		t.Logf("  %s: %d nodes, %d edges, status=%s", r.FilePath, r.NodeCount, r.EdgeCount, r.Status)
		totalNodes += r.NodeCount
	}
	assert.GreaterOrEqual(t, totalNodes, 8, "should have at least 8 symbols")

	// Step 4: Query via the engine.
	engine := query.NewEngine(store, fixtureDir)

	// 4a: GetFileStructure
	fs, _, err := engine.GetFileStructure("src/utils.ts")
	require.NoError(t, err)
	assert.Equal(t, "src/utils.ts", fs.File)
	assert.False(t, fs.Stale)
	assert.GreaterOrEqual(t, len(fs.Symbols), 4, "utils.ts should have at least 4 symbols")

	// 4b: FindSymbol
	symbols, _, err := engine.FindSymbol("handleRequest", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(symbols), 1)
	assert.Equal(t, "fn", symbols[0].Kind)

	// Step 5: Simulate file modification and detect stale.
	tmpDir := t.TempDir()
	copyFixture(t, fixtureDir, tmpDir)

	store2, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store2.Migrate())
	defer store2.Close()

	idx2 := indexer.NewIndexer(store2, runner, tmpDir, "typescript")
	_, err = idx2.IndexAll()
	require.NoError(t, err)

	// Modify a file.
	utilsPath := filepath.Join(tmpDir, "src", "utils.ts")
	originalContent, err := os.ReadFile(utilsPath)
	require.NoError(t, err)
	err = os.WriteFile(utilsPath, append(originalContent, []byte("\nexport function newHelper(): void {}")...), 0644)
	require.NoError(t, err)

	// Detect stale.
	stale, err := idx2.IsStale(filepath.Join(tmpDir, "src", "utils.ts"))
	require.NoError(t, err)
	assert.True(t, stale, "modified file should be detected as stale")

	// Step 6: Reindex stale files.
	staleResults, err := idx2.IndexStale()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(staleResults), 1, "should reindex at least 1 stale file")

	// Verify no longer stale.
	stale, err = idx2.IsStale(filepath.Join(tmpDir, "src", "utils.ts"))
	require.NoError(t, err)
	assert.False(t, stale, "reindexed file should not be stale")

	// Step 7: Query updated results.
	engine2 := query.NewEngine(store2, tmpDir)
	symbols2, _, err := engine2.FindSymbol("newHelper", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(symbols2), 1, "new function should be found after reindex")

	t.Log("End-to-end workflow test PASSED")
}

// TestEndToEnd_MCPProtocol validates MCP protocol compliance with tool calls.
func TestEndToEnd_MCPProtocol(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "ts-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	// Index fixture.
	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "typescript")
	_, err = idx.IndexAll()
	require.NoError(t, err)

	// Create MCP server.
	engine := query.NewEngine(store, fixtureDir)
	server := mcp.NewServer(engine, nil)

	// Test: get_file_structure via MCP.
	result, err := server.HandleToolCall(mcp.ToolCallParams{
		Name:      "get_file_structure",
		Arguments: map[string]interface{}{"file_path": "src/utils.ts"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	assert.Equal(t, "src/utils.ts", data["file"])

	meta, ok := data["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, meta, "stale_files")
	assert.Contains(t, meta, "query_duration_ms")

	// Test: find_symbol via MCP.
	result2, err := server.HandleToolCall(mcp.ToolCallParams{
		Name:      "find_symbol",
		Arguments: map[string]interface{}{"name": "formatDate"},
	})
	require.NoError(t, err)
	assert.False(t, result2.IsError)

	var data2 map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result2.Content[0].Text), &data2))
	matches, ok := data2["matches"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(matches), 1)

	// Test: get_references via MCP.
	result3, err := server.HandleToolCall(mcp.ToolCallParams{
		Name:      "get_references",
		Arguments: map[string]interface{}{"symbol": "formatDate"},
	})
	require.NoError(t, err)
	assert.False(t, result3.IsError)

	// Test: error for missing params.
	result4, err := server.HandleToolCall(mcp.ToolCallParams{
		Name:      "find_symbol",
		Arguments: map[string]interface{}{}, // missing "name"
	})
	require.NoError(t, err)
	assert.True(t, result4.IsError)

	var errData map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result4.Content[0].Text), &errData))
	errObj, ok := errData["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, errObj["type"], "codeindex.dev")

	t.Log("MCP protocol compliance test PASSED")
}

// copyFixture copies the fixture directory to a temp directory.
func copyFixture(t *testing.T, src string, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	require.NoError(t, err)
}
