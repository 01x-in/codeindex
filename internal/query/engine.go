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
	Kind  string `json:"kind"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	Depth int    `json:"depth"`
	Stale bool   `json:"stale"`
}

// SubgraphNode is a compact node representation for subgraph responses.
type SubgraphNode struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Exported bool   `json:"exported"`
	Stale    bool   `json:"stale"`
}

// SubgraphEdge is a compact edge representation for subgraph responses.
type SubgraphEdge struct {
	SourceID int64  `json:"source_id"`
	TargetID int64  `json:"target_id"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Subgraph represents a neighborhood of the knowledge graph.
type Subgraph struct {
	Nodes []SubgraphNode `json:"nodes"`
	Edges []SubgraphEdge `json:"edges"`
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

// stalenessCache provides per-query caching of file staleness checks.
// Each file is checked at most once, avoiding redundant disk I/O + hashing
// when multiple nodes share the same file.
type stalenessCache struct {
	engine *Engine
	cache  map[string]bool
}

// newStalenessCache creates a cache scoped to a single query.
func newStalenessCache(engine *Engine) *stalenessCache {
	return &stalenessCache{engine: engine, cache: make(map[string]bool)}
}

// isStale checks staleness with deduplication.
func (sc *stalenessCache) isStale(filePath string) bool {
	if v, ok := sc.cache[filePath]; ok {
		return v
	}
	stale := sc.engine.isFileStale(filePath)
	sc.cache[filePath] = stale
	return stale
}

// staleFiles returns all stale file paths found so far.
func (sc *stalenessCache) staleFiles() []string {
	var files []string
	for f, stale := range sc.cache {
		if stale {
			files = append(files, f)
		}
	}
	return files
}
