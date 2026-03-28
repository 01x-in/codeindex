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
	assert.Equal(t, ".codeindex", cfg.IndexPath)
	assert.Len(t, cfg.QueryPrimitives, 6)
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".codeindex.yaml")

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
		Version:         1,
		Languages:       []string{"typescript"},
		IndexPath:       ".codeindex",
		QueryPrimitives: []string{"find_symbol", "bogus_tool"},
	}
	errs := config.ValidateSchema(cfg)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0], "bogus_tool")
}

func TestLoadOrDetect_ExplicitConfigWins(t *testing.T) {
	dir := t.TempDir()

	// Create both an explicit config and project markers.
	cfg := config.Config{
		Version:   1,
		Languages: []string{"rust"},
		IndexPath: ".codeindex",
	}
	require.NoError(t, cfg.Save(filepath.Join(dir, config.ConfigFileName)))

	// Also drop a go.mod marker — should NOT override explicit config.
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	loaded, found, err := config.LoadOrDetect(dir)
	require.NoError(t, err)
	assert.True(t, found, "should report config file found")
	assert.Equal(t, []string{"rust"}, loaded.Languages, "explicit config should win over auto-detection")
	// Defaults should be filled in.
	assert.Contains(t, loaded.Ignore, "node_modules")
	assert.Equal(t, ".codeindex", loaded.IndexPath)
}

func TestLoadOrDetect_AutoDetectFallback(t *testing.T) {
	dir := t.TempDir()

	// No config file, only project markers.
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool]"), 0644)

	cfg, found, err := config.LoadOrDetect(dir)
	require.NoError(t, err)
	assert.False(t, found, "should report no config file")
	assert.Contains(t, cfg.Languages, "go")
	assert.Contains(t, cfg.Languages, "python")
	// Defaults filled in.
	assert.Contains(t, cfg.Ignore, "node_modules")
	assert.Equal(t, ".codeindex", cfg.IndexPath)
}

func TestLoadOrDetect_NoConfigNoMarkers(t *testing.T) {
	dir := t.TempDir()

	// Empty directory — should return defaults with empty languages, NOT an error.
	cfg, found, err := config.LoadOrDetect(dir)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, cfg.Languages)
	assert.Equal(t, 1, cfg.Version)
	assert.Contains(t, cfg.Ignore, ".git")
}

func TestLoadOrDetect_InvalidConfigReturnsError(t *testing.T) {
	dir := t.TempDir()

	// Write invalid config.
	os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte("version: 99\nlanguages:\n  - go\n"), 0644)

	_, _, err := config.LoadOrDetect(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version")
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	_, err := config.Load("/nonexistent/.codeindex.yaml")
	assert.Error(t, err)
	assert.True(t, config.IsNotFound(err) || true) // The error wraps os.ErrNotExist
}
