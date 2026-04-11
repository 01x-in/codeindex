package benchmark

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/01x/codeindex/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_LocalFixtureCleansUpWorkspace(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "ts-project")

	result, err := Run(Request{
		Source:   fixtureDir,
		Symbol:   "handleRequest",
		TempRoot: t.TempDir(),
	})
	require.NoError(t, err)

	assert.Equal(t, filepath.Clean(fixtureDir), result.OriginalSource)
	assert.Equal(t, "handleRequest", result.QuerySymbol)
	assert.Greater(t, result.FilesIndexed, 0)
	assert.NotEmpty(t, result.SampleFile)
	assert.NotEmpty(t, result.Markdown())

	_, statErr := os.Stat(result.WorkspacePath)
	assert.True(t, os.IsNotExist(statErr), "workspace should be removed by default")
}

func TestRun_LocalFixtureKeepWorkspace(t *testing.T) {
	testutil.SkipIfNoAstGrep(t)

	root := testutil.RepoRoot(t)
	fixtureDir := filepath.Join(root, "testdata", "ts-project")

	result, err := Run(Request{
		Source:        fixtureDir,
		Symbol:        "handleRequest",
		TempRoot:      t.TempDir(),
		KeepWorkspace: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(result.WorkspacePath)
	})

	assert.DirExists(t, result.WorkspacePath)
	assert.Equal(t, filepath.Clean(fixtureDir), result.OriginalSource)
}
