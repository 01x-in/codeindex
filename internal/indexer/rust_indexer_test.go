package indexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/01x/codeindex/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexRustWithMockRunner(t *testing.T) {
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	dir := t.TempDir()

	testFile := filepath.Join(dir, "models.rs")
	require.NoError(t, os.WriteFile(testFile, []byte(`pub struct User {
    pub id: String,
    pub name: String,
}

pub fn create_user(name: String) -> User {
    User { id: String::new(), name }
}
`), 0644))

	mockRunner := &indexer.MockRunner{
		Matches: []indexer.AstGrepMatch{
			{
				Text:   "pub struct User {\n    pub id: String,\n    pub name: String,\n}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 3, Column: 1}},
				File:   testFile,
				Lines:  "pub struct User {",
				RuleID: "rust-struct-def",
			},
			{
				Text:   "pub fn create_user(name: String) -> User {\n    User { id: String::new(), name }\n}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 5, Column: 0}, End: indexer.Position{Line: 7, Column: 1}},
				File:   testFile,
				Lines:  "pub fn create_user(name: String) -> User {",
				RuleID: "rust-func-def",
			},
		},
	}

	idx := indexer.NewIndexer(store, mockRunner, dir, "rust")
	result, err := idx.IndexFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, 2, result.NodeCount)

	userNodes, err := store.FindNodesByName("User")
	require.NoError(t, err)
	assert.Len(t, userNodes, 1)
	assert.Equal(t, "class", userNodes[0].Kind)
	assert.True(t, userNodes[0].Exported)
	assert.Equal(t, "rust", userNodes[0].Language)

	createNodes, err := store.FindNodesByName("create_user")
	require.NoError(t, err)
	assert.Len(t, createNodes, 1)
	assert.Equal(t, "fn", createNodes[0].Kind)
	assert.True(t, createNodes[0].Exported)
}

func TestIndexRustSingleFile(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "rust-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "rust")

	result, err := idx.IndexFile(filepath.Join(fixtureDir, "src", "models.rs"))
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)
	t.Logf("models.rs: %d nodes, %d edges", result.NodeCount, result.EdgeCount)

	nodes, err := store.FindNodesByFile("src/models.rs")
	require.NoError(t, err)
	for _, n := range nodes {
		t.Logf("  %s %s (line %d, exported=%v)", n.Kind, n.Name, n.LineStart, n.Exported)
	}

	// User struct must exist and be exported.
	userNodes, err := store.FindNodesByName("User")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(userNodes), 1, "User struct should be indexed")
	assert.Equal(t, "class", userNodes[0].Kind)
	assert.True(t, userNodes[0].Exported)

	// Repository trait must be a "type" kind.
	repoNodes, err := store.FindNodesByName("Repository")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(repoNodes), 1, "Repository trait should be indexed")
	assert.Equal(t, "type", repoNodes[0].Kind)
}

func TestIndexRustAll(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "rust-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "rust")

	results, err := idx.IndexAll()
	require.NoError(t, err)

	t.Logf("Indexed %d Rust files", len(results))
	for _, r := range results {
		t.Logf("  %s: %d nodes, %d edges, status=%s", r.FilePath, r.NodeCount, r.EdgeCount, r.Status)
	}

	assert.GreaterOrEqual(t, len(results), 3, "should index at least 3 Rust files")

	allMeta, err := store.GetAllFileMetadata()
	require.NoError(t, err)
	totalNodes := 0
	for _, m := range allMeta {
		totalNodes += m.NodeCount
	}
	t.Logf("Total nodes: %d", totalNodes)
	assert.GreaterOrEqual(t, totalNodes, 5, "fixture should have at least 5 symbols total")
}
