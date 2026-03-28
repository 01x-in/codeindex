package indexer_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func skipIfNoAstGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Skip("ast-grep not found in PATH — skipping integration test")
	}
}

func TestIndexFile_MockRunner(t *testing.T) {
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	dir := t.TempDir()

	testFile := filepath.Join(dir, "test.ts")
	require.NoError(t, os.WriteFile(testFile, []byte(`export function hello(): void {}`), 0644))

	mockRunner := &indexer.MockRunner{
		Matches: []indexer.AstGrepMatch{
			{
				Text:   "function hello(): void {}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 7}, End: indexer.Position{Line: 0, Column: 31}},
				File:   testFile,
				Lines:  "export function hello(): void {}",
				RuleID: "ts-function-def",
			},
			{
				Text:   "export function hello(): void {}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 31}},
				File:   testFile,
				Lines:  "export function hello(): void {}",
				RuleID: "ts-export-stmt",
			},
		},
	}

	idx := indexer.NewIndexer(store, mockRunner, dir, "typescript")
	result, err := idx.IndexFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, "ok", result.Status)
	assert.Greater(t, result.NodeCount, 0)

	nodes, err := store.FindNodesByName("hello")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes), 1)
	assert.Equal(t, "fn", nodes[0].Kind)
}

func TestIndexFile_Integration(t *testing.T) {
	skipIfNoAstGrep(t)

	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "ts-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "typescript")

	result, err := idx.IndexFile(filepath.Join(fixtureDir, "src", "utils.ts"))
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)

	nodes, err := store.FindNodesByFile("src/utils.ts")
	require.NoError(t, err)
	t.Logf("Found %d nodes in utils.ts", len(nodes))
	for _, n := range nodes {
		t.Logf("  %s %s (line %d, exported=%v)", n.Kind, n.Name, n.LineStart, n.Exported)
	}
	assert.GreaterOrEqual(t, len(nodes), 4, "should find at least formatDate, parseId, Config, Logger")

	formatDate, err := store.FindNodesByName("formatDate")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(formatDate), 1)
	assert.Equal(t, "fn", formatDate[0].Kind)

	logger, err := store.FindNodesByName("Logger")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logger), 1)
}

func TestIndexAll_Integration(t *testing.T) {
	skipIfNoAstGrep(t)

	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "ts-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "typescript")

	results, err := idx.IndexAll()
	require.NoError(t, err)

	t.Logf("Indexed %d files", len(results))
	for _, r := range results {
		t.Logf("  %s: %d nodes, %d edges, status=%s", r.FilePath, r.NodeCount, r.EdgeCount, r.Status)
	}

	assert.GreaterOrEqual(t, len(results), 3, "should index at least 3 TypeScript files")

	allMeta, err := store.GetAllFileMetadata()
	require.NoError(t, err)
	totalNodes := 0
	for _, m := range allMeta {
		totalNodes += m.NodeCount
	}
	t.Logf("Total nodes: %d", totalNodes)
	assert.GreaterOrEqual(t, totalNodes, 8, "fixture should have at least 8 symbols total")
}

func TestIndexFile_AstGrepError(t *testing.T) {
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.ts")
	require.NoError(t, os.WriteFile(testFile, []byte(`const x = 1;`), 0644))

	mockRunner := &indexer.MockRunner{
		Err: assert.AnError,
	}

	idx := indexer.NewIndexer(store, mockRunner, dir, "typescript")
	result, err := idx.IndexFile(testFile)
	require.NoError(t, err) // Error should be captured, not returned.
	assert.Equal(t, "error", result.Status)
	assert.NotEmpty(t, result.Error)

	// Metadata should show error status.
	relPath, _ := filepath.Rel(dir, testFile)
	meta, err := store.GetFileMetadata(relPath)
	require.NoError(t, err)
	assert.Equal(t, "error", meta.IndexStatus)
}
