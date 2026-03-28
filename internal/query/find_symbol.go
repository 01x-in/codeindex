package query

// FindSymbol locates where a symbol is defined across the codebase.
// If kind is non-empty, filters by symbol kind.
func (e *Engine) FindSymbol(name string, kind string) ([]SymbolResult, QueryMetadata, error) {
	nodes, err := e.store.FindNodesByName(name)
	if err != nil {
		return nil, QueryMetadata{}, err
	}

	var results []SymbolResult
	staleFilesMap := map[string]bool{}

	for _, n := range nodes {
		if kind != "" && n.Kind != kind {
			continue
		}

		stale := e.isFileStale(n.FilePath)
		if stale {
			staleFilesMap[n.FilePath] = true
		}

		results = append(results, SymbolResult{
			Name:     n.Name,
			Kind:     n.Kind,
			File:     n.FilePath,
			Line:     n.LineStart,
			Exported: n.Exported,
			Stale:    stale,
		})
	}

	var staleFiles []string
	for f := range staleFilesMap {
		staleFiles = append(staleFiles, f)
	}

	return results, QueryMetadata{StaleFiles: staleFiles}, nil
}
