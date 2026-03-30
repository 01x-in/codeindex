package query_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSubgraph_BasicNeighborhood(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// formatDate has: handleRequest -> formatDate (calls edge + imports edge)
	sg, meta, err := engine.GetSubgraph("formatDate", 1, nil)
	require.NoError(t, err)

	// Should include formatDate (root) + handleRequest (1 hop).
	assert.GreaterOrEqual(t, len(sg.Nodes), 2)
	assert.GreaterOrEqual(t, len(sg.Edges), 1)
	assert.Empty(t, meta.StaleFiles)

	// Verify root node is present.
	nameSet := map[string]bool{}
	for _, n := range sg.Nodes {
		nameSet[n.Name] = true
	}
	assert.True(t, nameSet["formatDate"])
	assert.True(t, nameSet["handleRequest"])
}

func TestGetSubgraph_DepthLimit(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)

	// Build a chain: a -> b -> c -> d
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	files := []string{"a.ts", "b.ts", "c.ts", "d.ts"}
	for _, name := range files {
		content := []byte("function " + name)
		os.WriteFile(filepath.Join(dir, "src/"+name), content, 0644)
		store.SetFileMetadata(graph.FileMetadata{
			FilePath: "src/" + name, ContentHash: hash.Bytes(content),
			Language: "typescript", NodeCount: 1, IndexStatus: "ok",
		})
	}

	idA, _ := store.UpsertNode(graph.Node{Name: "chainA", Kind: "fn", FilePath: "src/a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idB, _ := store.UpsertNode(graph.Node{Name: "chainB", Kind: "fn", FilePath: "src/b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idC, _ := store.UpsertNode(graph.Node{Name: "chainC", Kind: "fn", FilePath: "src/c.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idD, _ := store.UpsertNode(graph.Node{Name: "chainD", Kind: "fn", FilePath: "src/d.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})

	store.UpsertEdge(graph.Edge{SourceID: idA, TargetID: idB, Kind: "calls", FilePath: "src/a.ts", Line: 2})
	store.UpsertEdge(graph.Edge{SourceID: idB, TargetID: idC, Kind: "calls", FilePath: "src/b.ts", Line: 2})
	store.UpsertEdge(graph.Edge{SourceID: idC, TargetID: idD, Kind: "calls", FilePath: "src/c.ts", Line: 2})

	// Depth 1 from chainB: should get chainA (caller), chainB (root), chainC (callee).
	sg1, _, err := engine.GetSubgraph("chainB", 1, nil)
	require.NoError(t, err)
	assert.Len(t, sg1.Nodes, 3, "depth 1 from chainB should include A, B, C")

	// Depth 2 from chainB: should get all 4.
	sg2, _, err := engine.GetSubgraph("chainB", 2, nil)
	require.NoError(t, err)
	assert.Len(t, sg2.Nodes, 4, "depth 2 from chainB should include A, B, C, D")
}

func TestGetSubgraph_EdgeKindFilter(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Filter to only "calls" edges.
	sg, _, err := engine.GetSubgraph("formatDate", 1, []string{"calls"})
	require.NoError(t, err)

	// Should have formatDate + handleRequest (calls edge only).
	assert.GreaterOrEqual(t, len(sg.Nodes), 2)

	// All edges should be "calls".
	for _, e := range sg.Edges {
		assert.Equal(t, "calls", e.Kind)
	}
}

func TestGetSubgraph_NotFound(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	sg, _, err := engine.GetSubgraph("nonexistent", 2, nil)
	require.NoError(t, err)
	assert.Empty(t, sg.Nodes)
	assert.Empty(t, sg.Edges)
}

func TestGetSubgraph_IsolatedNode(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Config type has no edges.
	sg, _, err := engine.GetSubgraph("Config", 2, nil)
	require.NoError(t, err)
	assert.Len(t, sg.Nodes, 1) // Just the Config node itself
	assert.Empty(t, sg.Edges)
}

func TestGetSubgraph_CycleHandling(t *testing.T) {
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

	idA, _ := store.UpsertNode(graph.Node{Name: "sgCycA", Kind: "fn", FilePath: "src/a.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idB, _ := store.UpsertNode(graph.Node{Name: "sgCycB", Kind: "fn", FilePath: "src/b.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})
	idC, _ := store.UpsertNode(graph.Node{Name: "sgCycC", Kind: "fn", FilePath: "src/c.ts", LineStart: 1, LineEnd: 5, Language: "typescript"})

	store.UpsertEdge(graph.Edge{SourceID: idA, TargetID: idB, Kind: "calls", FilePath: "src/a.ts", Line: 2})
	store.UpsertEdge(graph.Edge{SourceID: idB, TargetID: idC, Kind: "calls", FilePath: "src/b.ts", Line: 2})
	store.UpsertEdge(graph.Edge{SourceID: idC, TargetID: idA, Kind: "calls", FilePath: "src/c.ts", Line: 2})

	// Should not infinite loop. Should find all 3 nodes and 3 edges.
	sg, _, err := engine.GetSubgraph("sgCycA", 10, nil)
	require.NoError(t, err)
	assert.Len(t, sg.Nodes, 3)
	assert.Len(t, sg.Edges, 3)
}

func TestGetSubgraph_StaleNodeTracking(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Make handler.ts stale.
	os.WriteFile(filepath.Join(dir, "src/handler.ts"), []byte("// modified"), 0644)

	sg, meta, err := engine.GetSubgraph("formatDate", 1, nil)
	require.NoError(t, err)

	// handleRequest should be flagged as stale.
	var staleCount int
	for _, n := range sg.Nodes {
		if n.Stale {
			staleCount++
		}
	}
	assert.GreaterOrEqual(t, staleCount, 1)
	assert.Contains(t, meta.StaleFiles, "src/handler.ts")
}

func TestGetSubgraph_DefaultDepth(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	// Passing 0 should use default depth (2).
	sg, _, err := engine.GetSubgraph("formatDate", 0, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sg.Nodes), 2)
}

func TestGetSubgraph_CompactRepresentation(t *testing.T) {
	engine, store, dir := setupQueryEngine(t)
	populateTestGraph(t, store, dir)

	sg, _, err := engine.GetSubgraph("formatDate", 1, nil)
	require.NoError(t, err)

	// Verify SubgraphNode has the required fields.
	for _, n := range sg.Nodes {
		assert.NotEmpty(t, n.Name)
		assert.NotEmpty(t, n.Kind)
		assert.NotEmpty(t, n.File)
		assert.Greater(t, n.Line, 0)
		assert.Greater(t, n.ID, int64(0))
	}

	// Verify SubgraphEdge has the required fields.
	for _, e := range sg.Edges {
		assert.Greater(t, e.SourceID, int64(0))
		assert.Greater(t, e.TargetID, int64(0))
		assert.NotEmpty(t, e.Kind)
	}
}
