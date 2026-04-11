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

func TestGoIndexFile_MockRunner(t *testing.T) {
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	dir := t.TempDir()

	testFile := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(testFile, []byte(`package main

func main() {
	fmt.Println("hello")
}
`), 0644))

	mockRunner := &indexer.MockRunner{
		Matches: []indexer.AstGrepMatch{
			{
				Text:   "func main() {\n\tfmt.Println(\"hello\")\n}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 2, Column: 0}, End: indexer.Position{Line: 4, Column: 1}},
				File:   testFile,
				Lines:  "func main() {\n\tfmt.Println(\"hello\")\n}",
				RuleID: "go-function-def",
			},
			{
				Text:   "fmt.Println(\"hello\")",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 3, Column: 1}, End: indexer.Position{Line: 3, Column: 21}},
				File:   testFile,
				Lines:  "\tfmt.Println(\"hello\")",
				RuleID: "go-call-expr",
			},
		},
	}

	idx := indexer.NewIndexer(store, mockRunner, dir, "go")
	result, err := idx.IndexFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, 1, result.NodeCount)

	nodes, err := store.FindNodesByName("main")
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "fn", nodes[0].Kind)
	assert.False(t, nodes[0].Exported)
	assert.Equal(t, "go", nodes[0].Language)
}

func TestGoIndexFile_Integration(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "go-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "go")

	// Index the models file.
	result, err := idx.IndexFile(filepath.Join(fixtureDir, "pkg", "models", "user.go"))
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Status)

	nodes, err := store.FindNodesByFile("pkg/models/user.go")
	require.NoError(t, err)
	t.Logf("Found %d nodes in pkg/models/user.go", len(nodes))
	for _, n := range nodes {
		t.Logf("  %s %s (line %d, exported=%v, scope=%s)", n.Kind, n.Name, n.LineStart, n.Exported, n.Scope)
	}

	// Should find: User (struct), UserFilter (struct), Validatable (interface),
	// Validate (method), FormatName (fn).
	assert.GreaterOrEqual(t, len(nodes), 4, "should find at least User, UserFilter, Validatable, and FormatName/Validate")

	// Check specific symbols.
	userNodes, err := store.FindNodesByName("User")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(userNodes), 1)
	assert.Equal(t, "class", userNodes[0].Kind) // struct -> class

	validatableNodes, err := store.FindNodesByName("Validatable")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(validatableNodes), 1)
	assert.Equal(t, "interface", validatableNodes[0].Kind)
}

func TestGoIndexAll_Integration(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "go-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "go")

	results, err := idx.IndexAll()
	require.NoError(t, err)

	t.Logf("Indexed %d Go files", len(results))
	for _, r := range results {
		t.Logf("  %s: %d nodes, %d edges, status=%s", r.FilePath, r.NodeCount, r.EdgeCount, r.Status)
	}

	// Should index main.go, pkg/models/user.go, pkg/service/user_service.go.
	assert.GreaterOrEqual(t, len(results), 3, "should index at least 3 Go files")

	allMeta, err := store.GetAllFileMetadata()
	require.NoError(t, err)
	totalNodes := 0
	for _, m := range allMeta {
		totalNodes += m.NodeCount
	}
	t.Logf("Total nodes: %d", totalNodes)
	assert.GreaterOrEqual(t, totalNodes, 8, "fixture should have at least 8 symbols total")
}

func TestGoIndexFile_MethodScope(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "go-project")

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	defer store.Close()

	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, fixtureDir, "go")

	_, err = idx.IndexFile(filepath.Join(fixtureDir, "pkg", "models", "user.go"))
	require.NoError(t, err)

	// Validate is a method on *User — its scope should be "User".
	validateNodes, err := store.FindNodesByName("Validate")
	require.NoError(t, err)
	if assert.GreaterOrEqual(t, len(validateNodes), 1) {
		assert.Equal(t, "User", validateNodes[0].Scope, "method receiver type should be stored as scope")
	}
}
