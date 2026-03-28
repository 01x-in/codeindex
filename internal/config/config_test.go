package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	assert.Equal(t, 1, cfg.Version)
	assert.Contains(t, cfg.Ignore, "node_modules")
	assert.Equal(t, ".code-index", cfg.IndexPath)
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".code-index.yaml")

	cfg := config.DefaultConfig()
	cfg.Languages = []string{"typescript", "go"}

	err := cfg.Save(path)
	require.NoError(t, err)

	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, cfg.Version, loaded.Version)
	assert.Equal(t, cfg.Languages, loaded.Languages)
}

func TestValidateRejectsUnknownLanguage(t *testing.T) {
	cfg := config.Config{Version: 1, Languages: []string{"cobol"}}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cobol")
}

func TestValidateRejectsWrongVersion(t *testing.T) {
	cfg := config.Config{Version: 99, Languages: []string{"go"}}
	err := cfg.Validate()
	assert.Error(t, err)
}

func TestDetectLanguages(t *testing.T) {
	dir := t.TempDir()

	// Create marker files.
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	results, err := config.DetectLanguages(dir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)

	langs := make(map[string]bool)
	for _, r := range results {
		langs[r.Language] = true
	}
	assert.True(t, langs["go"])
	assert.True(t, langs["typescript"])
}

func TestValidateSchema(t *testing.T) {
	cfg := config.Config{
		Version:   1,
		Languages: []string{"typescript"},
		IndexPath: ".code-index",
		QueryPrimitives: []string{"find_symbol", "bogus_tool"},
	}
	errs := config.ValidateSchema(cfg)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0], "bogus_tool")
}
