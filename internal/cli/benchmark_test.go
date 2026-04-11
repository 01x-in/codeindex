package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptBenchmarkInputs_UsesArgsWithoutPrompt(t *testing.T) {
	var stdout bytes.Buffer

	repo, symbol, err := promptBenchmarkInputs(
		[]string{"https://github.com/vercel/next.js", "createServer"},
		strings.NewReader(""),
		&stdout,
		false,
	)
	require.NoError(t, err)

	assert.Equal(t, "https://github.com/vercel/next.js", repo)
	assert.Equal(t, "createServer", symbol)
	assert.Empty(t, stdout.String())
}

func TestPromptBenchmarkInputs_PromptsForMissingValues(t *testing.T) {
	var stdout bytes.Buffer

	repo, symbol, err := promptBenchmarkInputs(
		nil,
		strings.NewReader("/tmp/example-repo\ncreateServer\n"),
		&stdout,
		true,
	)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/example-repo", repo)
	assert.Equal(t, "createServer", symbol)
	assert.Contains(t, stdout.String(), "Repository URL or local path")
	assert.Contains(t, stdout.String(), "e.g. createServer")
}

func TestPromptBenchmarkInputs_ErrorsWhenPromptingDisabled(t *testing.T) {
	var stdout bytes.Buffer

	_, _, err := promptBenchmarkInputs(nil, strings.NewReader(""), &stdout, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo URL or local path")
}

func TestNormalizeMarkdownOutputPath_AppendsExtension(t *testing.T) {
	assert.Equal(t, "benchmarks/results/next.js.md", normalizeMarkdownOutputPath("benchmarks/results/next.js"))
	assert.Equal(t, "benchmarks/results/next.js.md", normalizeMarkdownOutputPath("benchmarks/results/next.js.md"))
}
