package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) (*graph.SQLiteStore, string) {
	t.Helper()

	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())

	// Create a temp dir with a test file for staleness checks.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.ts")
	require.NoError(t, os.WriteFile(testFile, []byte("export function handleRequest() {}"), 0644))

	return store, tmpDir
}

func seedGraph(t *testing.T, store *graph.SQLiteStore) {
	t.Helper()

	// Nodes
	handleReqID, err := store.UpsertNode(graph.Node{
		Name: "handleRequest", Kind: "fn", FilePath: "handler.ts",
		LineStart: 1, LineEnd: 1, ColStart: 0, ColEnd: 35,
		Exported: true, Language: "typescript",
	})
	require.NoError(t, err)

	routeReqID, err := store.UpsertNode(graph.Node{
		Name: "routeRequest", Kind: "fn", FilePath: "routes.ts",
		LineStart: 5, LineEnd: 10, ColStart: 0, ColEnd: 20,
		Exported: true, Language: "typescript",
	})
	require.NoError(t, err)

	validateID, err := store.UpsertNode(graph.Node{
		Name: "validateInput", Kind: "fn", FilePath: "validation.ts",
		LineStart: 3, LineEnd: 8, ColStart: 0, ColEnd: 30,
		Exported: true, Language: "typescript",
	})
	require.NoError(t, err)

	configTypeID, err := store.UpsertNode(graph.Node{
		Name: "RequestConfig", Kind: "type", FilePath: "handler.ts",
		LineStart: 15, LineEnd: 20, ColStart: 0, ColEnd: 25,
		Exported: true, Language: "typescript",
	})
	require.NoError(t, err)

	// Edges: routeRequest calls handleRequest
	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: routeReqID, TargetID: handleReqID,
		Kind: "calls", FilePath: "routes.ts", Line: 7,
	}))

	// Edges: handleRequest calls validateInput
	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: handleReqID, TargetID: validateID,
		Kind: "calls", FilePath: "handler.ts", Line: 2,
	}))

	// Edges: routeRequest imports handleRequest
	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: routeReqID, TargetID: handleReqID,
		Kind: "imports", FilePath: "routes.ts", Line: 1,
	}))

	// Edges: handleRequest references RequestConfig
	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: handleReqID, TargetID: configTypeID,
		Kind: "references", FilePath: "handler.ts", Line: 1,
	}))
}

func TestBuildSymbolTree(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()
	seedGraph(t, store)

	builder := NewSymbolTreeBuilder(store, tmpDir)

	root, err := builder.BuildSymbolTree("handleRequest")
	require.NoError(t, err)

	assert.Equal(t, "handleRequest", root.Name)
	assert.Equal(t, "fn", root.Kind)
	assert.True(t, root.Expanded)

	// Should have groups: callers, callees, importers
	// (type references group is for edges TO the root, but we have refs FROM root)
	require.True(t, len(root.Children) >= 2, "expected at least 2 child groups, got %d", len(root.Children))

	// Find callers group
	var callersGroup *TreeNode
	for _, child := range root.Children {
		if child.label == "callers" {
			callersGroup = child
			break
		}
	}
	require.NotNil(t, callersGroup, "callers group should exist")
	require.Len(t, callersGroup.Children, 1)
	assert.Equal(t, "routeRequest", callersGroup.Children[0].Name)

	// Find callees group
	var calleesGroup *TreeNode
	for _, child := range root.Children {
		if child.label == "callees" {
			calleesGroup = child
			break
		}
	}
	require.NotNil(t, calleesGroup, "callees group should exist")
	require.Len(t, calleesGroup.Children, 1)
	assert.Equal(t, "validateInput", calleesGroup.Children[0].Name)
}

func TestBuildSymbolTreeNotFound(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()

	builder := NewSymbolTreeBuilder(store, tmpDir)
	_, err := builder.BuildSymbolTree("nonExistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBuildFileTree(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()
	seedGraph(t, store)

	builder := NewSymbolTreeBuilder(store, tmpDir)

	root, err := builder.BuildFileTree("handler.ts")
	require.NoError(t, err)

	assert.Equal(t, "handler.ts", root.Name)
	assert.True(t, root.Expanded)

	// Should have groups for functions and types.
	require.True(t, len(root.Children) >= 1, "expected at least 1 kind group")

	// Find Functions group
	var fnGroup *TreeNode
	for _, child := range root.Children {
		if child.label == "Functions" {
			fnGroup = child
			break
		}
	}
	require.NotNil(t, fnGroup, "Functions group should exist")
	assert.True(t, len(fnGroup.Children) >= 1)

	// Find Types group
	var typeGroup *TreeNode
	for _, child := range root.Children {
		if child.label == "Types" {
			typeGroup = child
			break
		}
	}
	require.NotNil(t, typeGroup, "Types group should exist")
	assert.Len(t, typeGroup.Children, 1)
	assert.Equal(t, "RequestConfig", typeGroup.Children[0].Name)
}

func TestBuildFileTreeNotIndexed(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()

	builder := NewSymbolTreeBuilder(store, tmpDir)
	_, err := builder.BuildFileTree("nonexistent.ts")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not indexed")
}

func TestBuildSymbolTreeWithImporters(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()
	seedGraph(t, store)

	builder := NewSymbolTreeBuilder(store, tmpDir)
	root, err := builder.BuildSymbolTree("handleRequest")
	require.NoError(t, err)

	var importersGroup *TreeNode
	for _, child := range root.Children {
		if child.label == "importers" {
			importersGroup = child
			break
		}
	}
	require.NotNil(t, importersGroup, "importers group should exist")
	require.Len(t, importersGroup.Children, 1)
	assert.Equal(t, "routeRequest", importersGroup.Children[0].Name)
}
