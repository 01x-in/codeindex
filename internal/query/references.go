package query

import "fmt"

// GetReferences finds every file and line that uses a given symbol.
func (e *Engine) GetReferences(symbolName string) ([]ReferenceResult, QueryMetadata, error) {
	// Find the symbol nodes.
	nodes, err := e.store.FindNodesByName(symbolName)
	if err != nil {
		return nil, QueryMetadata{}, err
	}

	if len(nodes) == 0 {
		return []ReferenceResult{}, QueryMetadata{}, nil
	}

	var results []ReferenceResult
	staleFilesMap := map[string]bool{}

	for _, n := range nodes {
		// Find edges pointing TO this node (callers, importers, references).
		edges, err := e.store.GetEdgesTo(n.ID, "")
		if err != nil {
			continue
		}

		for _, edge := range edges {
			source, err := e.store.GetNode(edge.SourceID)
			if err != nil {
				continue
			}

			stale := e.isFileStale(edge.FilePath)
			if stale {
				staleFilesMap[edge.FilePath] = true
			}

			context := fmt.Sprintf("%s %s %s", source.Name, edge.Kind, symbolName)

			results = append(results, ReferenceResult{
				File:    edge.FilePath,
				Line:    edge.Line,
				Kind:    edge.Kind,
				Context: context,
				Stale:   stale,
			})
		}
	}

	var staleFiles []string
	for f := range staleFilesMap {
		staleFiles = append(staleFiles, f)
	}

	return results, QueryMetadata{StaleFiles: staleFiles}, nil
}
