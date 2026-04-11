package hash_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x-in/codeindex/internal/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytes(t *testing.T) {
	h1 := hash.Bytes([]byte("hello"))
	h2 := hash.Bytes([]byte("hello"))
	h3 := hash.Bytes([]byte("world"))

	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
	assert.Len(t, h1, 64) // SHA-256 hex
}

func TestFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	h, err := hash.File(path)
	require.NoError(t, err)
	assert.Equal(t, hash.Bytes([]byte("hello")), h)
}

func TestFileNotFound(t *testing.T) {
	_, err := hash.File("/nonexistent/path")
	assert.Error(t, err)
}
