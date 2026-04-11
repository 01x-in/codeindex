package indexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x-in/codeindex/internal/graph"
	"github.com/01x-in/codeindex/internal/indexer"
	"github.com/01x-in/codeindex/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonIndexFile_MockRunner(t *testing.T) {
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	dir := t.TempDir()

	testFile := filepath.Join(dir, "models.py")
	require.NoError(t, os.WriteFile(testFile, []byte(`class User:
    id: str
    name: str

def create_user(name: str) -> "User":
    return User(id="", name=name)
`), 0644))

	mockRunner := &indexer.MockRunner{
		Matches: []indexer.AstGrepMatch{
			{
				Text:   "class User:\n    id: str\n    name: str",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 2, Column: 14}},
				File:   testFile,
				Lines:  "class User:\n    id: str\n    name: str",
				RuleID: "python-class-def",
			},
			{
				Text:   "def create_user(name: str) -> \"User\":\n    return User(id=\"\", name=name)",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 4, Column: 0}, End: indexer.Position{Line: 5, Column: 31}},
				File:   testFile,
				Lines:  "def create_user(name: str) -> \"User\":\n    return User(id=\"\", name=name)",
				RuleID: "python-func-def",
			},
		},
	}

	idx := indexer.NewIndexer(store, mockRunner, dir, "python")
	result, err := idx.IndexFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, 2, result.NodeCount)

	userNodes, err := store.FindNodesByName("User")
	require.NoError(t, err)
	assert.Len(t, userNodes, 1)
	assert.Equal(t, "class", userNodes[0].Kind)
	assert.True(t, userNodes[0].Exported)
	assert.Equal(t, "python", userNodes[0].Language)

	createNodes, err := store.FindNodesByName("create_user")
	require.NoError(t, err)
	assert.Len(t, createNodes, 1)
	assert.Equal(t, "fn", createNodes[0].Kind)
	assert.True(t, createNodes[0].Exported)
}

func TestIndexPythonFixture(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "py-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "python")

	// Index models.py
	result, err := idx.IndexFile(filepath.Join(fixtureDir, "src", "models.py"))
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	t.Logf("models.py: %d nodes, %d edges", result.NodeCount, result.EdgeCount)

	nodes, err := store.FindNodesByFile("src/models.py")
	require.NoError(t, err)
	for _, n := range nodes {
		t.Logf("  %s %s (line %d, exported=%v)", n.Kind, n.Name, n.LineStart, n.Exported)
	}

	// User class must exist and be exported.
	userNodes, err := store.FindNodesByName("User")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(userNodes), 1, "User class should be indexed")
	assert.Equal(t, "class", userNodes[0].Kind)
	assert.True(t, userNodes[0].Exported)

	// Product class must exist and be exported.
	productNodes, err := store.FindNodesByName("Product")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(productNodes), 1, "Product class should be indexed")
	assert.Equal(t, "class", productNodes[0].Kind)

	// create_user function must exist and be exported.
	createNodes, err := store.FindNodesByName("create_user")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(createNodes), 1, "create_user function should be indexed")
	assert.Equal(t, "fn", createNodes[0].Kind)
	assert.True(t, createNodes[0].Exported)
}

func TestIndexPythonFixture_PrivateFuncNotExported(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "py-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "python")

	_, err = idx.IndexFile(filepath.Join(fixtureDir, "src", "models.py"))
	require.NoError(t, err)

	// _internal_helper must be indexed but not exported.
	helperNodes, err := store.FindNodesByName("_internal_helper")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(helperNodes), 1, "_internal_helper should be indexed")
	assert.False(t, helperNodes[0].Exported, "_internal_helper should not be exported")
}

func TestIndexPythonAll(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "py-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "python")

	results, err := idx.IndexAll()
	require.NoError(t, err)

	t.Logf("Indexed %d Python files", len(results))
	for _, r := range results {
		t.Logf("  %s: %d nodes, %d edges, status=%s", r.FilePath, r.NodeCount, r.EdgeCount, r.Status)
	}

	// Should index models.py, service.py, utils.py.
	assert.GreaterOrEqual(t, len(results), 3, "should index at least 3 Python files")

	allMeta, err := store.GetAllFileMetadata()
	require.NoError(t, err)
	totalNodes := 0
	for _, m := range allMeta {
		totalNodes += m.NodeCount
	}
	t.Logf("Total nodes: %d", totalNodes)
	assert.GreaterOrEqual(t, totalNodes, 5, "fixture should have at least 5 symbols total")
}
