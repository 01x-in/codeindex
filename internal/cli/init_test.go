package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/01x-in/codeindex/internal/cli"
	"github.com/01x-in/codeindex/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStdin creates a temporary file with the given content to simulate stdin.
func mockStdin(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "stdin-*")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	_, err = f.Seek(0, 0)
	require.NoError(t, err)
	t.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	return f
}

func TestInitWithYesFlag_TypeScript(t *testing.T) {
	dir := t.TempDir()

	// Create TypeScript markers.
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	// Verify config was written.
	cfg, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Contains(t, cfg.Languages, "typescript")

	// Verify output mentions typescript.
	assert.Contains(t, stdout.String(), "typescript")
	assert.Contains(t, stdout.String(), "Wrote .codeindex.yaml")
}

func TestInitWithYesFlag_Go(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	cfg, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Contains(t, cfg.Languages, "go")
}

func TestInitWithYesFlag_Python(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool]"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	cfg, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Contains(t, cfg.Languages, "python")
}

func TestInitWithYesFlag_Rust(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	cfg, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Contains(t, cfg.Languages, "rust")
}

func TestInitNoLanguagesDetected(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "No languages detected")
	assert.Contains(t, stdout.String(), "Wrote .codeindex.yaml")

	// Config should have empty languages.
	cfg, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Empty(t, cfg.Languages)
}

func TestInitPrintsAgentIntegrationHint(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Agent integration:")
	assert.Contains(t, output, "codeindex query <subcommand>")
	assert.Contains(t, output, "npx skills add codeindex/skills")
	assert.NotContains(t, output, "MCP config")
	assert.NotContains(t, output, "claude mcp add")
}

func TestInitCreatesGitignore(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	// .gitignore should exist and contain .codeindex/
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codeindex/")
}

func TestInitAppendsToExistingGitignore(t *testing.T) {
	dir := t.TempDir()

	// Write existing .gitignore.
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "node_modules/")
	assert.Contains(t, content, ".codeindex/")
}

func TestInitDoesNotDuplicateGitignoreEntry(t *testing.T) {
	dir := t.TempDir()

	// .gitignore already has the entry.
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".codeindex/\n"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	// Should only appear once.
	count := 0
	for _, line := range bytes.Split(data, []byte("\n")) {
		if string(bytes.TrimSpace(line)) == ".codeindex/" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestInitExistingConfigAbortWithNo(t *testing.T) {
	dir := t.TempDir()

	// Write existing config.
	cfg := config.DefaultConfig()
	cfg.Languages = []string{"go"}
	cfg.Save(filepath.Join(dir, config.ConfigFileName))

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "n\n")

	err := cli.RunInit(dir, false, stdin, &stdout, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Aborted")

	// Original config should be untouched.
	loaded, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Equal(t, []string{"go"}, loaded.Languages)
}

func TestInitExistingConfigOverwriteWithYes(t *testing.T) {
	dir := t.TempDir()

	// Write existing config.
	cfg := config.DefaultConfig()
	cfg.Languages = []string{"go"}
	cfg.Save(filepath.Join(dir, config.ConfigFileName))

	// Also add a rust marker.
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := mockStdin(t, "")

	err := cli.RunInit(dir, true, stdin, &stdout, &stderr)
	require.NoError(t, err)

	// Config should be overwritten with detected languages.
	loaded, err := config.Load(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Contains(t, loaded.Languages, "rust")
}
