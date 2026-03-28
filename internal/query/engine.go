package query

import (
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
)

// Engine provides high-level query operations over the knowledge graph.
type Engine struct {
	store    graph.Store
	repoRoot string
}

// NewEngine creates a new query engine.
func NewEngine(store graph.Store, repoRoot string) *Engine {
	return &Engine{store: store, repoRoot: repoRoot}
}

// FileStructure represents the structural skeleton of a file.
type FileStructure struct {
	File    string       `json:"file"`
	Stale   bool         `json:"stale"`
	Symbols []SymbolInfo `json:"symbols"`
	Imports []ImportInfo `json:"imports"`
}

// SymbolInfo is a summary of a symbol for file structure responses.
type SymbolInfo struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Line      int    `json:"line"`
	Exported  bool   `json:"exported"`
	Signature string `json:"signature,omitempty"`
}

// ImportInfo represents an import in a file.
type ImportInfo struct {
	Name string `json:"name"`
	From string `json:"from"`
}

// SymbolResult represents a found symbol.
type SymbolResult struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Exported bool   `json:"exported"`
	Stale    bool   `json:"stale"`
}

// ReferenceResult represents a reference to a symbol.
type ReferenceResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Kind    string `json:"kind"`
	Context string `json:"context"`
	Stale   bool   `json:"stale"`
}

// CallerResult represents a caller in the call graph.
type CallerResult struct {
	Name  string `json:"name"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	Stale bool   `json:"stale"`
}

// Subgraph represents a neighborhood of the knowledge graph.
type Subgraph struct {
	Nodes []graph.Node `json:"nodes"`
	Edges []graph.Edge `json:"edges"`
}

// QueryMetadata holds metadata about a query response.
type QueryMetadata struct {
	StaleFiles      []string `json:"stale_files"`
	QueryDurationMs int64    `json:"query_duration_ms"`
	IndexAgeSeconds int64    `json:"index_age_seconds,omitempty"`
}

// isFileStale checks if a file is stale using the indexer's staleness detection.
func (e *Engine) isFileStale(filePath string) bool {
	stale, err := indexer.IsStaleFile(e.store, e.repoRoot, filePath)
	if err != nil {
		return true // err on the side of caution
	}
	return stale
}
