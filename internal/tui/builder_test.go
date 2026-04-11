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

	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "handler.ts",
		ContentHash: "handler-hash",
		Language:    "typescript",
		NodeCount:   2,
		EdgeCount:   1,
		IndexStatus: "ok",
	}))
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "routes.ts",
		ContentHash: "routes-hash",
		Language:    "typescript",
		NodeCount:   1,
		EdgeCount:   2,
		IndexStatus: "ok",
	}))
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "validation.ts",
		ContentHash: "validation-hash",
		Language:    "typescript",
		NodeCount:   1,
		EdgeCount:   0,
		IndexStatus: "ok",
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

func TestBuildRepoTree(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()
	seedGraph(t, store)

	builder := NewSymbolTreeBuilder(store, tmpDir)
	root, err := builder.BuildRepoTree()
	require.NoError(t, err)

	assert.Equal(t, filepath.Base(tmpDir), root.Name)
	assert.True(t, root.Expanded)
	require.Len(t, root.Children, 3)
	assert.Equal(t, "handler.ts", root.Children[0].Name)
	assert.Equal(t, "routes.ts", root.Children[1].Name)
	assert.Equal(t, "validation.ts", root.Children[2].Name)
}

func TestBuildSymbolTreePrefersRepoFilesAndFiltersPackageReferences(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()

	repoRootID, err := store.UpsertNode(graph.Node{
		Name: "Connect", Kind: "fn", FilePath: "internal/service.go",
		LineStart: 10, LineEnd: 20, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "go",
	})
	require.NoError(t, err)

	repoCallerID, err := store.UpsertNode(graph.Node{
		Name: "Run", Kind: "fn", FilePath: "cmd/codeindex/main.go",
		LineStart: 5, LineEnd: 12, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "go",
	})
	require.NoError(t, err)

	packageRootID, err := store.UpsertNode(graph.Node{
		Name: "Connect", Kind: "fn", FilePath: ".venv/lib/python3.11/site-packages/httpx/_client.py",
		LineStart: 100, LineEnd: 120, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "python",
	})
	require.NoError(t, err)

	packageCallerID, err := store.UpsertNode(graph.Node{
		Name: "PoolConnect", Kind: "fn", FilePath: ".venv/lib/python3.11/site-packages/httpx/_pool.py",
		LineStart: 40, LineEnd: 60, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "python",
	})
	require.NoError(t, err)

	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: repoCallerID, TargetID: repoRootID,
		Kind: "calls", FilePath: "cmd/codeindex/main.go", Line: 7,
	}))
	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: packageCallerID, TargetID: repoRootID,
		Kind: "calls", FilePath: ".venv/lib/python3.11/site-packages/httpx/_pool.py", Line: 44,
	}))
	require.NoError(t, store.UpsertEdge(graph.Edge{
		SourceID: packageCallerID, TargetID: packageRootID,
		Kind: "calls", FilePath: ".venv/lib/python3.11/site-packages/httpx/_pool.py", Line: 48,
	}))

	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "internal/service.go",
		ContentHash: "repo-root",
		Language:    "go",
		NodeCount:   1,
		EdgeCount:   1,
		IndexStatus: "ok",
	}))
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "cmd/codeindex/main.go",
		ContentHash: "repo-caller",
		Language:    "go",
		NodeCount:   1,
		EdgeCount:   1,
		IndexStatus: "ok",
	}))
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    ".venv/lib/python3.11/site-packages/httpx/_client.py",
		ContentHash: "pkg-root",
		Language:    "python",
		NodeCount:   1,
		EdgeCount:   0,
		IndexStatus: "ok",
	}))
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    ".venv/lib/python3.11/site-packages/httpx/_pool.py",
		ContentHash: "pkg-caller",
		Language:    "python",
		NodeCount:   1,
		EdgeCount:   2,
		IndexStatus: "ok",
	}))

	builder := NewSymbolTreeBuilder(store, tmpDir)
	root, err := builder.BuildSymbolTree("Connect")
	require.NoError(t, err)

	assert.Equal(t, "internal/service.go", root.FilePath)

	var callersGroup *TreeNode
	for _, child := range root.Children {
		if child.label == "callers" {
			callersGroup = child
			break
		}
	}
	require.NotNil(t, callersGroup, "callers group should exist")
	require.Len(t, callersGroup.Children, 1)
	assert.Equal(t, "Run", callersGroup.Children[0].Name)
	assert.Equal(t, "cmd/codeindex/main.go", callersGroup.Children[0].FilePath)
}

func TestBuildRepoTreeFiltersPackageFiles(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()

	repoNodeID, err := store.UpsertNode(graph.Node{
		Name: "Run", Kind: "fn", FilePath: "cmd/codeindex/main.go",
		LineStart: 5, LineEnd: 12, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "go",
	})
	require.NoError(t, err)
	packageNodeID, err := store.UpsertNode(graph.Node{
		Name: "Connect", Kind: "fn", FilePath: ".venv/lib/python3.11/site-packages/httpx/_client.py",
		LineStart: 100, LineEnd: 120, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "python",
	})
	require.NoError(t, err)
	assert.NotZero(t, repoNodeID)
	assert.NotZero(t, packageNodeID)

	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "cmd/codeindex/main.go",
		ContentHash: "repo",
		Language:    "go",
		NodeCount:   1,
		EdgeCount:   0,
		IndexStatus: "ok",
	}))
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    ".venv/lib/python3.11/site-packages/httpx/_client.py",
		ContentHash: "package",
		Language:    "python",
		NodeCount:   1,
		EdgeCount:   0,
		IndexStatus: "ok",
	}))

	builder := NewSymbolTreeBuilder(store, tmpDir)
	root, err := builder.BuildRepoTree()
	require.NoError(t, err)

	require.Len(t, root.Children, 1)
	assert.Equal(t, "cmd/codeindex/main.go", root.Children[0].Name)
}
