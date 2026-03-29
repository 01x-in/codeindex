package query

import "fmt"

// GetSubgraph retrieves a bounded neighborhood of the knowledge graph around a symbol.
// It uses a recursive CTE to perform the traversal in a single SQL query,
// following edges in both directions. Depth is configurable (default 2, max 10).
// Edge kinds can be filtered. Cycles are handled by the CTE's UNION.
func (e *Engine) GetSubgraph(symbolName string, depth int, edgeKinds []string) (Subgraph, QueryMetadata, error) {
	if depth < 1 {
		depth = 2
	}
	if depth > 10 {
		depth = 10
	}

	nodes, err := e.store.FindNodesByName(symbolName)
	if err != nil {
		return Subgraph{}, QueryMetadata{}, fmt.Errorf("finding symbol %q: %w", symbolName, err)
	}

	if len(nodes) == 0 {
		return Subgraph{Nodes: []SubgraphNode{}, Edges: []SubgraphEdge{}}, QueryMetadata{}, nil
	}

	seedIDs := make([]int64, len(nodes))
	for i, n := range nodes {
		seedIDs[i] = n.ID
	}

	cteNodes, cteEdges, err := e.store.GetNeighborhoodCTE(seedIDs, depth, edgeKinds)
	if err != nil {
		return Subgraph{}, QueryMetadata{}, fmt.Errorf("CTE neighborhood traversal: %w", err)
	}

	sc := newStalenessCache(e)
	resultNodes := make([]SubgraphNode, 0, len(cteNodes))
	for _, n := range cteNodes {
		stale := sc.isStale(n.FilePath)
		resultNodes = append(resultNodes, SubgraphNode{
			ID:       n.ID,
			Name:     n.Name,
			Kind:     n.Kind,
			File:     n.FilePath,
			Line:     n.LineStart,
			Exported: n.Exported,
			Stale:    stale,
		})
	}

	resultEdges := make([]SubgraphEdge, 0, len(cteEdges))
	for _, e := range cteEdges {
		resultEdges = append(resultEdges, SubgraphEdge{
			SourceID: e.SourceID,
			TargetID: e.TargetID,
			Kind:     e.Kind,
			File:     e.FilePath,
			Line:     e.Line,
		})
	}

	return Subgraph{
		Nodes: resultNodes,
		Edges: resultEdges,
	}, QueryMetadata{StaleFiles: sc.staleFiles()}, nil
}
