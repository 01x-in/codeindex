package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// IndexResult holds the result of indexing a file.
type IndexResult struct {
	FilePath  string
	NodeCount int
	EdgeCount int
	Status    string // ok, error, partial
	Error     string
}

// IndexFile indexes a single file, updating the graph store.
func (idx *Indexer) IndexFile(filePath string) (IndexResult, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(idx.repoRoot, filePath)
	}

	relPath, err := filepath.Rel(idx.repoRoot, absPath)
	if err != nil {
		relPath = filePath
	}

	result := IndexResult{FilePath: relPath, Status: "ok"}

	// Compute content hash.
	contentHash, err := hash.File(absPath)
	if err != nil {
		return result, fmt.Errorf("hashing file %s: %w", relPath, err)
	}

	// Get the rules for this language.
	rules, ok := LanguageRules[idx.language]
	if !ok {
		return result, fmt.Errorf("no rules for language %q", idx.language)
	}

	// Run ast-grep.
	matches, err := idx.runner.ScanWithInlineRules(rules, absPath)
	if err != nil {
		// Mark as error but don't fail completely — preserve old data.
		result.Status = "error"
		result.Error = err.Error()

		idx.store.SetFileMetadata(graph.FileMetadata{
			FilePath:     relPath,
			ContentHash:  contentHash,
			Language:     idx.language,
			IndexStatus:  "error",
			ErrorMessage: err.Error(),
		})
		return result, nil
	}

	// Parse matches into nodes and edges.
	parsed := ParseMatches(matches, relPath, idx.language)

	// Delete old data for this file.
	if err := idx.store.DeleteFileData(relPath); err != nil {
		return result, fmt.Errorf("deleting old data for %s: %w", relPath, err)
	}

	// Upsert nodes.
	nodeIDs := map[string]int64{} // name -> id for edge resolution
	for _, node := range parsed.Nodes {
		id, err := idx.store.UpsertNode(node)
		if err != nil {
			return result, fmt.Errorf("upserting node %s: %w", node.Name, err)
		}
		nodeIDs[node.Name] = id
	}

	// Resolve and upsert edges.
	edgeCount := 0
	for _, pe := range parsed.Edges {
		targetID, ok := idx.resolveTarget(pe.TargetName, nodeIDs)
		if !ok {
			continue // Target not found in the graph — skip.
		}

		// For the source: if it's a call or import, try to find the enclosing function.
		sourceID := idx.resolveSource(pe, nodeIDs)
		if sourceID == 0 {
			continue
		}

		err := idx.store.UpsertEdge(graph.Edge{
			SourceID: sourceID,
			TargetID: targetID,
			Kind:     pe.Kind,
			FilePath: pe.FilePath,
			Line:     pe.Line,
		})
		if err != nil {
			// Non-fatal: skip this edge.
			continue
		}
		edgeCount++
	}

	result.NodeCount = len(parsed.Nodes)
	result.EdgeCount = edgeCount

	// Update file metadata.
	if err := idx.store.SetFileMetadata(graph.FileMetadata{
		FilePath:    relPath,
		ContentHash: contentHash,
		Language:    idx.language,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		IndexStatus: "ok",
	}); err != nil {
		return result, fmt.Errorf("setting metadata for %s: %w", relPath, err)
	}

	return result, nil
}

// resolveTarget looks up a target node ID by name, first in the current file's
// nodes, then in the full graph.
func (idx *Indexer) resolveTarget(name string, localNodes map[string]int64) (int64, bool) {
	// Check local nodes first.
	if id, ok := localNodes[name]; ok {
		return id, true
	}

	// Handle dotted names (e.g., "this.logger") — extract the method name.
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		simpleName := parts[len(parts)-1]
		if id, ok := localNodes[simpleName]; ok {
			return id, true
		}
		// Search the global graph.
		nodes, err := idx.store.FindNodesByName(simpleName)
		if err == nil && len(nodes) > 0 {
			return nodes[0].ID, true
		}
	}

	// Search the global graph.
	nodes, err := idx.store.FindNodesByName(name)
	if err == nil && len(nodes) > 0 {
		return nodes[0].ID, true
	}

	return 0, false
}

// resolveSource finds the source node for an edge. For calls/imports in a file,
// the source is typically the first function defined in this file at or before
// the edge's line number. If no function is found, uses the first node in the file.
func (idx *Indexer) resolveSource(pe ParsedEdge, localNodes map[string]int64) int64 {
	// For imports, the "source" is the file itself — use the first node.
	if pe.Kind == "imports" {
		// Return any local node as the source.
		for _, id := range localNodes {
			return id
		}
		return 0
	}

	// For calls, find the enclosing function (the node with the largest line_start <= pe.Line).
	// This is a simplified heuristic — look through local nodes.
	var bestID int64
	var bestLine int
	nodes, err := idx.store.FindNodesByFile(pe.FilePath)
	if err != nil {
		return 0
	}
	for _, n := range nodes {
		if n.Kind == "fn" && n.LineStart <= pe.Line && n.LineStart > bestLine {
			bestID = n.ID
			bestLine = n.LineStart
		}
	}
	if bestID != 0 {
		return bestID
	}

	// Fall back to any local node.
	for _, id := range localNodes {
		return id
	}
	return 0
}

// IndexAll indexes all files matching the configured language in the repo.
func (idx *Indexer) IndexAll() ([]IndexResult, error) {
	files, err := idx.findFiles()
	if err != nil {
		return nil, fmt.Errorf("finding files: %w", err)
	}

	var results []IndexResult
	for _, file := range files {
		result, err := idx.IndexFile(file)
		if err != nil {
			return results, fmt.Errorf("indexing %s: %w", file, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// IndexStale indexes only stale files.
func (idx *Indexer) IndexStale() ([]IndexResult, error) {
	files, err := idx.findFiles()
	if err != nil {
		return nil, fmt.Errorf("finding files: %w", err)
	}

	var results []IndexResult
	for _, file := range files {
		stale, err := idx.IsStale(file)
		if err != nil {
			return results, err
		}
		if !stale {
			continue
		}

		result, err := idx.IndexFile(file)
		if err != nil {
			return results, fmt.Errorf("indexing %s: %w", file, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// IsStale checks if a file has changed since last indexing.
// A file is considered stale if:
// - It has never been indexed
// - Its content hash has changed since last indexing
// - Its previous index attempt resulted in an error status
func (idx *Indexer) IsStale(filePath string) (bool, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(idx.repoRoot, filePath)
	}

	relPath, err := filepath.Rel(idx.repoRoot, absPath)
	if err != nil {
		relPath = filePath
	}

	meta, err := idx.store.GetFileMetadata(relPath)
	if err != nil {
		// File not indexed yet = stale.
		return true, nil
	}

	// Files with prior index errors should always be retried.
	if meta.IndexStatus == "error" || meta.IndexStatus == "partial" {
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

// IsStaleFile checks staleness for a file path against the graph store.
func IsStaleFile(store graph.Store, repoRoot string, filePath string) (bool, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(repoRoot, filePath)
	}

	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		relPath = filePath
	}

	meta, err := store.GetFileMetadata(relPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return true, nil
	}

	// Files with prior index errors should always be retried.
	if meta.IndexStatus == "error" || meta.IndexStatus == "partial" {
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

// findFiles returns all files matching the configured language in the repo.
func (idx *Indexer) findFiles() ([]string, error) {
	exts := languageExtensions(idx.language)
	if len(exts) == 0 {
		return nil, fmt.Errorf("no file extensions for language %q", idx.language)
	}

	var files []string
	err := filepath.Walk(idx.repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors.
		}

		// Skip ignored directories.
		if info.IsDir() {
			base := filepath.Base(path)
			if isIgnoredDir(base) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension.
		ext := filepath.Ext(path)
		for _, e := range exts {
			if ext == e {
				files = append(files, path)
				break
			}
		}
		return nil
	})

	return files, err
}

// languageExtensions returns the file extensions for a language.
func languageExtensions(language string) []string {
	switch language {
	case "typescript":
		return []string{".ts", ".tsx"}
	case "go":
		return []string{".go"}
	case "python":
		return []string{".py"}
	case "rust":
		return []string{".rs"}
	default:
		return nil
	}
}

// isIgnoredDir returns true for commonly ignored directories.
func isIgnoredDir(name string) bool {
	ignored := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		".git":         true,
		"dist":         true,
		"build":        true,
		".codeindex":  true,
		"__pycache__":  true,
		"target":       true,
	}
	return ignored[name]
}
