package indexer

import (
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/hash"
)

// Indexer orchestrates ast-grep parsing and populates the graph store.
type Indexer struct {
	store    graph.Store
	runner   AstGrepRunner
	repoRoot string
	language string
}

// NewIndexer creates a new Indexer.
func NewIndexer(store graph.Store, runner AstGrepRunner, repoRoot string, language string) *Indexer {
	return &Indexer{
		store:    store,
		runner:   runner,
		repoRoot: repoRoot,
		language: language,
	}
}

// IndexFile indexes a single file, updating the graph store.
func (idx *Indexer) IndexFile(filePath string) error {
	// TODO: M1-S5 implementation
	// 1. Run ast-grep on file
	// 2. Parse output into nodes/edges
	// 3. Delete old data for file
	// 4. Upsert new nodes/edges
	// 5. Update file metadata with content hash
	return nil
}

// IndexAll indexes all files matching the configured language.
func (idx *Indexer) IndexAll() error {
	// TODO: M1-S6 implementation
	return nil
}

// IsStale checks if a file has changed since last indexing.
func (idx *Indexer) IsStale(filePath string) (bool, error) {
	meta, err := idx.store.GetFileMetadata(filePath)
	if err != nil {
		// File not indexed yet = stale.
		return true, nil
	}

	currentHash, err := hash.File(filePath)
	if err != nil {
		return false, err
	}

	return currentHash != meta.ContentHash, nil
}
