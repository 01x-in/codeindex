package graph

import "time"

// Node represents a symbol in the knowledge graph.
type Node struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`       // fn, class, type, interface, var, export
	FilePath  string    `json:"file_path"`
	LineStart int       `json:"line_start"`
	LineEnd   int       `json:"line_end"`
	ColStart  int       `json:"col_start"`
	ColEnd    int       `json:"col_end"`
	Scope     string    `json:"scope"`     // parent scope (e.g., class name for methods)
	Signature string    `json:"signature"` // type signature if available
	Exported  bool      `json:"exported"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Edge represents a relationship between two symbols.
type Edge struct {
	ID        int64     `json:"id"`
	SourceID  int64     `json:"source_id"`
	TargetID  int64     `json:"target_id"`
	Kind      string    `json:"kind"`      // calls, imports, implements, extends, references
	FilePath  string    `json:"file_path"` // file where the reference occurs
	Line      int       `json:"line"`
	CreatedAt time.Time `json:"created_at"`
}

// FileMetadata tracks per-file indexing state.
type FileMetadata struct {
	FilePath      string    `json:"file_path"`
	ContentHash   string    `json:"content_hash"`   // SHA-256
	LastIndexedAt time.Time `json:"last_indexed_at"`
	Language      string    `json:"language"`
	NodeCount     int       `json:"node_count"`
	EdgeCount     int       `json:"edge_count"`
	IndexStatus   string    `json:"index_status"`  // ok, error, partial
	ErrorMessage  string    `json:"error_message"`
}
