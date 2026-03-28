package indexer

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

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
// Returns true if:
//   - File is not in metadata (new file)
//   - File has been deleted from disk
//   - File content hash differs from stored hash
func (idx *Indexer) IsStale(filePath string) (bool, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(idx.repoRoot, filePath)
	}

	meta, err := idx.store.GetFileMetadata(filePath)
	if err != nil {
		// File not indexed yet = stale.
		return true, nil
	}

	currentHash, err := hash.File(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File deleted from disk = stale.
			return true, nil
		}
		return false, err
	}

	return currentHash != meta.ContentHash, nil
}

// IsStaleFile checks staleness for a file path against the graph store.
// This is a standalone function for use outside the Indexer context.
func IsStaleFile(store graph.Store, repoRoot string, filePath string) (bool, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(repoRoot, filePath)
	}

	meta, err := store.GetFileMetadata(filePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		// Any other error — treat as stale since we can't verify.
		return true, nil
	}

	currentHash, err := hash.File(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	return currentHash != meta.ContentHash, nil
}

// GetStaleFiles returns a list of file paths that are stale.
func GetStaleFiles(store graph.Store, repoRoot string) ([]string, error) {
	allMeta, err := store.GetAllFileMetadata()
	if err != nil {
		return nil, err
	}

	var staleFiles []string
	for _, meta := range allMeta {
		stale, err := IsStaleFile(store, repoRoot, meta.FilePath)
		if err != nil {
			return nil, err
		}
		if stale {
			staleFiles = append(staleFiles, meta.FilePath)
		}
	}
	return staleFiles, nil
}
