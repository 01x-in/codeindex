package indexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/hash"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStoreAndDir(t *testing.T) (*graph.SQLiteStore, string) {
	t.Helper()
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	t.Cleanup(func() { store.Close() })

	dir := t.TempDir()
	return store, dir
}

func TestIsStale_FreshFile(t *testing.T) {
	store, dir := setupStoreAndDir(t)

	// Create a file and index it.
	filePath := filepath.Join(dir, "fresh.ts")
	content := []byte("export function hello(): void {}")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	h := hash.Bytes(content)
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    filePath,
		ContentHash: h,
		Language:    "typescript",
		IndexStatus: "ok",
	}))

	idx := indexer.NewIndexer(store, nil, dir, "typescript")
	stale, err := idx.IsStale(filePath)
	require.NoError(t, err)
	assert.False(t, stale, "file with matching hash should not be stale")
}

func TestIsStale_ModifiedFile(t *testing.T) {
	store, dir := setupStoreAndDir(t)

	filePath := filepath.Join(dir, "modified.ts")
	originalContent := []byte("export function hello(): void {}")
	require.NoError(t, os.WriteFile(filePath, originalContent, 0644))

	h := hash.Bytes(originalContent)
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    filePath,
		ContentHash: h,
		Language:    "typescript",
		IndexStatus: "ok",
	}))

	// Modify the file.
	require.NoError(t, os.WriteFile(filePath, []byte("export function hello(): string { return 'hi'; }"), 0644))

	idx := indexer.NewIndexer(store, nil, dir, "typescript")
	stale, err := idx.IsStale(filePath)
	require.NoError(t, err)
	assert.True(t, stale, "file with different hash should be stale")
}

func TestIsStale_DeletedFile(t *testing.T) {
	store, dir := setupStoreAndDir(t)

	filePath := filepath.Join(dir, "deleted.ts")
	content := []byte("export function hello(): void {}")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	h := hash.Bytes(content)
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    filePath,
		ContentHash: h,
		Language:    "typescript",
		IndexStatus: "ok",
	}))

	// Delete the file.
	require.NoError(t, os.Remove(filePath))

	idx := indexer.NewIndexer(store, nil, dir, "typescript")
	stale, err := idx.IsStale(filePath)
	require.NoError(t, err)
	assert.True(t, stale, "deleted file should be stale")
}

func TestIsStale_NewFile(t *testing.T) {
	store, dir := setupStoreAndDir(t)

	// File exists on disk but not in metadata.
	filePath := filepath.Join(dir, "new.ts")
	require.NoError(t, os.WriteFile(filePath, []byte("const x = 1;"), 0644))

	idx := indexer.NewIndexer(store, nil, dir, "typescript")
	stale, err := idx.IsStale(filePath)
	require.NoError(t, err)
	assert.True(t, stale, "file not in metadata should be stale")
}

func TestIsStaleFile_Standalone(t *testing.T) {
	store, dir := setupStoreAndDir(t)

	filePath := filepath.Join(dir, "test.ts")
	content := []byte("export const x = 42;")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	h := hash.Bytes(content)
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    filePath,
		ContentHash: h,
		Language:    "typescript",
		IndexStatus: "ok",
	}))

	// Fresh.
	stale, err := indexer.IsStaleFile(store, dir, filePath)
	require.NoError(t, err)
	assert.False(t, stale)

	// Modify.
	require.NoError(t, os.WriteFile(filePath, []byte("export const x = 99;"), 0644))
	stale, err = indexer.IsStaleFile(store, dir, filePath)
	require.NoError(t, err)
	assert.True(t, stale)
}

func TestGetStaleFiles(t *testing.T) {
	store, dir := setupStoreAndDir(t)

	// Create 3 files, index all, then modify 1.
	files := map[string][]byte{
		"a.ts": []byte("export const a = 1;"),
		"b.ts": []byte("export const b = 2;"),
		"c.ts": []byte("export const c = 3;"),
	}

	for name, content := range files {
		fp := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(fp, content, 0644))
		require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
			FilePath:    fp,
			ContentHash: hash.Bytes(content),
			Language:    "typescript",
			IndexStatus: "ok",
		}))
	}

	// Modify b.ts.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.ts"), []byte("export const b = 99;"), 0644))

	staleFiles, err := indexer.GetStaleFiles(store, dir)
	require.NoError(t, err)
	assert.Len(t, staleFiles, 1)
	assert.Contains(t, staleFiles[0], "b.ts")
}
