package query

import "fmt"

// GetCallers traces the call graph upstream from a function.
// It uses a recursive CTE to perform the traversal in a single SQL query,
// avoiding N+1 query patterns. Depth is configurable: default 3, max 10.
// Cycles are handled by the CTE's UNION (deduplicates visited nodes).
func (e *Engine) GetCallers(symbolName string, depth int) ([]CallerResult, QueryMetadata, error) {
	if depth < 1 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	nodes, err := e.store.FindNodesByName(symbolName)
	if err != nil {
		return nil, QueryMetadata{}, fmt.Errorf("finding symbol %q: %w", symbolName, err)
	}

	if len(nodes) == 0 {
		return []CallerResult{}, QueryMetadata{}, nil
	}

	seedIDs := make([]int64, len(nodes))
	for i, n := range nodes {
		seedIDs[i] = n.ID
	}

	entries, err := e.store.GetCallersCTE(seedIDs, depth)
	if err != nil {
		return nil, QueryMetadata{}, fmt.Errorf("CTE callers traversal: %w", err)
	}

	sc := newStalenessCache(e)
	var results []CallerResult

	for _, entry := range entries {
		stale := sc.isStale(entry.CallerNode.FilePath)
		results = append(results, CallerResult{
			Name:  entry.CallerNode.Name,
			Kind:  entry.CallerNode.Kind,
			File:  entry.CallerNode.FilePath,
			Line:  entry.CallerNode.LineStart,
			Depth: entry.Depth,
			Stale: stale,
		})
	}

	return results, QueryMetadata{StaleFiles: sc.staleFiles()}, nil
}
