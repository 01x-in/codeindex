package query

import "fmt"

// GetFileStructure returns the structural skeleton of a file.
func (e *Engine) GetFileStructure(filePath string) (FileStructure, QueryMetadata, error) {
	nodes, err := e.store.FindNodesByFile(filePath)
	if err != nil {
		return FileStructure{}, QueryMetadata{}, fmt.Errorf("finding nodes: %w", err)
	}

	stale := e.isFileStale(filePath)

	var symbols []SymbolInfo
	var imports []ImportInfo

	for _, n := range nodes {
		symbols = append(symbols, SymbolInfo{
			Name:      n.Name,
			Kind:      n.Kind,
			Line:      n.LineStart,
			Exported:  n.Exported,
			Signature: n.Signature,
		})
	}

	// Collect import edges.
	for _, n := range nodes {
		edges, err := e.store.GetEdgesFrom(n.ID, "imports")
		if err != nil {
			continue
		}
		for _, edge := range edges {
			target, err := e.store.GetNode(edge.TargetID)
			if err != nil {
				continue
			}
			imports = append(imports, ImportInfo{
				Name: target.Name,
				From: target.FilePath,
			})
		}
	}

	var staleFiles []string
	if stale {
		staleFiles = append(staleFiles, filePath)
	}

	return FileStructure{
			File:    filePath,
			Stale:   stale,
			Symbols: symbols,
			Imports: imports,
		}, QueryMetadata{
			StaleFiles: staleFiles,
		}, nil
}
