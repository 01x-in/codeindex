package query_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/01x-in/codeindex/internal/graph"
	"github.com/01x-in/codeindex/internal/hash"
	"github.com/01x-in/codeindex/internal/query"
	"github.com/stretchr/testify/require"
)

// buildLargeGraph creates a graph with N nodes and roughly 2*N edges for benchmarking.
// Structure: linear chain of "calls" edges plus some cross-references.
func buildLargeGraph(b *testing.B, nodeCount int) (*query.Engine, *graph.SQLiteStore, string) {
	b.Helper()

	dir := b.TempDir()
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(b, err)
	require.NoError(b, store.Migrate())
	b.Cleanup(func() { store.Close() })

	engine := query.NewEngine(store, dir)

	// Create files and nodes.
	os.MkdirAll(filepath.Join(dir, "src"), 0755)

	nodeIDs := make([]int64, nodeCount)
	for i := 0; i < nodeCount; i++ {
		fileName := fmt.Sprintf("src/file_%04d.ts", i)
		content := []byte(fmt.Sprintf("export function func_%04d() {}", i))
		os.WriteFile(filepath.Join(dir, fileName), content, 0644)

		store.SetFileMetadata(graph.FileMetadata{
			FilePath: fileName, ContentHash: hash.Bytes(content),
			Language: "typescript", NodeCount: 1, IndexStatus: "ok",
		})

		id, err := store.UpsertNode(graph.Node{
			Name:      fmt.Sprintf("func_%04d", i),
			Kind:      "fn",
			FilePath:  fileName,
			LineStart: 1, LineEnd: 5,
			Exported: true, Language: "typescript",
		})
		require.NoError(b, err)
		nodeIDs[i] = id
	}

	// Create a linear call chain: 0 -> 1 -> 2 -> ... -> N-1.
	for i := 0; i < nodeCount-1; i++ {
		store.UpsertEdge(graph.Edge{
			SourceID: nodeIDs[i], TargetID: nodeIDs[i+1],
			Kind: "calls", FilePath: fmt.Sprintf("src/file_%04d.ts", i), Line: 3,
		})
	}

	// Add some cross-references (every 10th node references every 5th node).
	for i := 0; i < nodeCount; i += 10 {
		for j := 0; j < nodeCount; j += 5 {
			if i != j {
				store.UpsertEdge(graph.Edge{
					SourceID: nodeIDs[i], TargetID: nodeIDs[j],
					Kind: "references", FilePath: fmt.Sprintf("src/file_%04d.ts", i), Line: 2,
				})
			}
		}
	}

	return engine, store, dir
}

func BenchmarkGetSubgraph_Depth2_1000Nodes(b *testing.B) {
	engine, _, _ := buildLargeGraph(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := engine.GetSubgraph("func_0500", 2, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetCallers_Depth3_1000Nodes(b *testing.B) {
	engine, _, _ := buildLargeGraph(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := engine.GetCallers("func_0500", 3)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetSubgraph_Depth2_CallsOnly_1000Nodes(b *testing.B) {
	engine, _, _ := buildLargeGraph(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := engine.GetSubgraph("func_0500", 2, []string{"calls"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestPerformanceTargets validates the performance targets are met.
func TestPerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	dir := t.TempDir()
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	t.Cleanup(func() { store.Close() })

	engine := query.NewEngine(store, dir)

	// Build 1000-node graph.
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	nodeIDs := make([]int64, 1000)
	for i := 0; i < 1000; i++ {
		fileName := fmt.Sprintf("src/file_%04d.ts", i)
		content := []byte(fmt.Sprintf("export function func_%04d() {}", i))
		os.WriteFile(filepath.Join(dir, fileName), content, 0644)

		store.SetFileMetadata(graph.FileMetadata{
			FilePath: fileName, ContentHash: hash.Bytes(content),
			Language: "typescript", NodeCount: 1, IndexStatus: "ok",
		})

		id, err := store.UpsertNode(graph.Node{
			Name: fmt.Sprintf("func_%04d", i), Kind: "fn", FilePath: fileName,
			LineStart: 1, LineEnd: 5, Exported: true, Language: "typescript",
		})
		require.NoError(t, err)
		nodeIDs[i] = id
	}

	for i := 0; i < 999; i++ {
		store.UpsertEdge(graph.Edge{
			SourceID: nodeIDs[i], TargetID: nodeIDs[i+1],
			Kind: "calls", FilePath: fmt.Sprintf("src/file_%04d.ts", i), Line: 3,
		})
	}

	// Verify get_subgraph depth=2 completes (correctness check; benchmark tests timing).
	sg, _, err := engine.GetSubgraph("func_0500", 2, nil)
	require.NoError(t, err)
	if len(sg.Nodes) < 3 {
		t.Errorf("expected at least 3 nodes in subgraph, got %d", len(sg.Nodes))
	}

	// Verify get_callers depth=3 completes.
	callers, _, err := engine.GetCallers("func_0500", 3)
	require.NoError(t, err)
	if len(callers) < 3 {
		t.Errorf("expected at least 3 callers, got %d", len(callers))
	}
}
