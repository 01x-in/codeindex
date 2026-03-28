package query

import "fmt"

// GetCallers traces the call graph upstream from a function.
// It performs a BFS traversal following "calls" edges in reverse (who calls this?).
// Depth is configurable: default 3, max 10. Cycles are handled via a visited set.
func (e *Engine) GetCallers(symbolName string, depth int) ([]CallerResult, QueryMetadata, error) {
	// Clamp depth.
	if depth < 1 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	// Find the target symbol nodes.
	nodes, err := e.store.FindNodesByName(symbolName)
	if err != nil {
		return nil, QueryMetadata{}, fmt.Errorf("finding symbol %q: %w", symbolName, err)
	}

	if len(nodes) == 0 {
		return []CallerResult{}, QueryMetadata{}, nil
	}

	var results []CallerResult
	staleFilesMap := map[string]bool{}

	// Collect all target node IDs as the initial frontier.
	visited := map[int64]bool{}
	type bfsEntry struct {
		nodeID int64
		depth  int
	}

	var frontier []bfsEntry
	for _, n := range nodes {
		visited[n.ID] = true
		frontier = append(frontier, bfsEntry{nodeID: n.ID, depth: 0})
	}

	// BFS: walk "calls" edges in reverse.
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]

		if current.depth >= depth {
			continue
		}

		// Find edges where this node is the target (callers).
		edges, err := e.store.GetEdgesTo(current.nodeID, "calls")
		if err != nil {
			continue
		}

		for _, edge := range edges {
			if visited[edge.SourceID] {
				continue
			}
			visited[edge.SourceID] = true

			caller, err := e.store.GetNode(edge.SourceID)
			if err != nil {
				continue
			}

			callerDepth := current.depth + 1
			stale := e.isFileStale(caller.FilePath)
			if stale {
				staleFilesMap[caller.FilePath] = true
			}

			results = append(results, CallerResult{
				Name:  caller.Name,
				Kind:  caller.Kind,
				File:  caller.FilePath,
				Line:  caller.LineStart,
				Depth: callerDepth,
				Stale: stale,
			})

			// Add to frontier for further traversal.
			frontier = append(frontier, bfsEntry{nodeID: edge.SourceID, depth: callerDepth})
		}
	}

	var staleFiles []string
	for f := range staleFilesMap {
		staleFiles = append(staleFiles, f)
	}

	return results, QueryMetadata{StaleFiles: staleFiles}, nil
}
