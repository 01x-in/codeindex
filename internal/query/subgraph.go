package query

import "fmt"

// edgeInfo is an internal type for edge traversal results.
type edgeInfo struct {
	SourceID int64
	TargetID int64
	Kind     string
	FilePath string
	Line     int
}

// GetSubgraph retrieves a bounded neighborhood of the knowledge graph around a symbol.
// It performs a BFS traversal from the symbol, following edges in both directions.
// Depth is configurable (default 2, max 10). Edge kinds can be filtered.
func (e *Engine) GetSubgraph(symbolName string, depth int, edgeKinds []string) (Subgraph, QueryMetadata, error) {
	// Clamp depth.
	if depth < 1 {
		depth = 2
	}
	if depth > 10 {
		depth = 10
	}

	// Find the root symbol nodes.
	nodes, err := e.store.FindNodesByName(symbolName)
	if err != nil {
		return Subgraph{}, QueryMetadata{}, fmt.Errorf("finding symbol %q: %w", symbolName, err)
	}

	if len(nodes) == 0 {
		return Subgraph{Nodes: []SubgraphNode{}, Edges: []SubgraphEdge{}}, QueryMetadata{}, nil
	}

	staleFilesMap := map[string]bool{}
	visitedNodes := map[int64]bool{}
	visitedEdges := map[string]bool{} // "sourceID-targetID-kind" dedup key

	var resultNodes []SubgraphNode
	var resultEdges []SubgraphEdge

	type bfsEntry struct {
		nodeID int64
		depth  int
	}

	var frontier []bfsEntry
	for _, n := range nodes {
		visitedNodes[n.ID] = true
		frontier = append(frontier, bfsEntry{nodeID: n.ID, depth: 0})

		stale := e.isFileStale(n.FilePath)
		if stale {
			staleFilesMap[n.FilePath] = true
		}
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

	// BFS: traverse edges in both directions.
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]

		if current.depth >= depth {
			continue
		}

		// Outgoing edges (this node -> targets).
		outEdges, err := e.getFilteredEdges(current.nodeID, edgeKinds, true)
		if err == nil {
			for _, edge := range outEdges {
				edgeKey := fmt.Sprintf("%d-%d-%s", edge.SourceID, edge.TargetID, edge.Kind)
				if !visitedEdges[edgeKey] {
					visitedEdges[edgeKey] = true
					resultEdges = append(resultEdges, SubgraphEdge{
						SourceID: edge.SourceID,
						TargetID: edge.TargetID,
						Kind:     edge.Kind,
						File:     edge.FilePath,
						Line:     edge.Line,
					})
				}

				if !visitedNodes[edge.TargetID] {
					visitedNodes[edge.TargetID] = true
					target, err := e.store.GetNode(edge.TargetID)
					if err != nil {
						continue
					}
					stale := e.isFileStale(target.FilePath)
					if stale {
						staleFilesMap[target.FilePath] = true
					}
					resultNodes = append(resultNodes, SubgraphNode{
						ID:       target.ID,
						Name:     target.Name,
						Kind:     target.Kind,
						File:     target.FilePath,
						Line:     target.LineStart,
						Exported: target.Exported,
						Stale:    stale,
					})
					frontier = append(frontier, bfsEntry{nodeID: target.ID, depth: current.depth + 1})
				}
			}
		}

		// Incoming edges (sources -> this node).
		inEdges, err := e.getFilteredEdges(current.nodeID, edgeKinds, false)
		if err == nil {
			for _, edge := range inEdges {
				edgeKey := fmt.Sprintf("%d-%d-%s", edge.SourceID, edge.TargetID, edge.Kind)
				if !visitedEdges[edgeKey] {
					visitedEdges[edgeKey] = true
					resultEdges = append(resultEdges, SubgraphEdge{
						SourceID: edge.SourceID,
						TargetID: edge.TargetID,
						Kind:     edge.Kind,
						File:     edge.FilePath,
						Line:     edge.Line,
					})
				}

				if !visitedNodes[edge.SourceID] {
					visitedNodes[edge.SourceID] = true
					source, err := e.store.GetNode(edge.SourceID)
					if err != nil {
						continue
					}
					stale := e.isFileStale(source.FilePath)
					if stale {
						staleFilesMap[source.FilePath] = true
					}
					resultNodes = append(resultNodes, SubgraphNode{
						ID:       source.ID,
						Name:     source.Name,
						Kind:     source.Kind,
						File:     source.FilePath,
						Line:     source.LineStart,
						Exported: source.Exported,
						Stale:    stale,
					})
					frontier = append(frontier, bfsEntry{nodeID: source.ID, depth: current.depth + 1})
				}
			}
		}
	}

	var staleFiles []string
	for f := range staleFilesMap {
		staleFiles = append(staleFiles, f)
	}

	return Subgraph{
		Nodes: resultNodes,
		Edges: resultEdges,
	}, QueryMetadata{StaleFiles: staleFiles}, nil
}

// getFilteredEdges returns edges filtered by kind.
// If edgeKinds is empty, returns all edges.
// If outgoing is true, returns edges from nodeID; otherwise returns edges to nodeID.
func (e *Engine) getFilteredEdges(nodeID int64, edgeKinds []string, outgoing bool) ([]edgeInfo, error) {
	if len(edgeKinds) == 0 {
		if outgoing {
			edges, err := e.store.GetEdgesFrom(nodeID, "")
			if err != nil {
				return nil, err
			}
			result := make([]edgeInfo, len(edges))
			for i, eg := range edges {
				result[i] = edgeInfo{SourceID: eg.SourceID, TargetID: eg.TargetID, Kind: eg.Kind, FilePath: eg.FilePath, Line: eg.Line}
			}
			return result, nil
		}
		edges, err := e.store.GetEdgesTo(nodeID, "")
		if err != nil {
			return nil, err
		}
		result := make([]edgeInfo, len(edges))
		for i, eg := range edges {
			result[i] = edgeInfo{SourceID: eg.SourceID, TargetID: eg.TargetID, Kind: eg.Kind, FilePath: eg.FilePath, Line: eg.Line}
		}
		return result, nil
	}

	var all []edgeInfo
	for _, kind := range edgeKinds {
		if outgoing {
			edges, err := e.store.GetEdgesFrom(nodeID, kind)
			if err != nil {
				return nil, err
			}
			for _, eg := range edges {
				all = append(all, edgeInfo{SourceID: eg.SourceID, TargetID: eg.TargetID, Kind: eg.Kind, FilePath: eg.FilePath, Line: eg.Line})
			}
		} else {
			edges, err := e.store.GetEdgesTo(nodeID, kind)
			if err != nil {
				return nil, err
			}
			for _, eg := range edges {
				all = append(all, edgeInfo{SourceID: eg.SourceID, TargetID: eg.TargetID, Kind: eg.Kind, FilePath: eg.FilePath, Line: eg.Line})
			}
		}
	}
	return all, nil
}
