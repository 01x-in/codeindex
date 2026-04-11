package benchmark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/01x-in/codeindex/internal/testutil"
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

func TestCopyLocalRepoSkipsEscapingSymlinks(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "safe.txt"), []byte("safe"), 0644))
	require.NoError(t, os.Symlink("safe.txt", filepath.Join(srcRoot, "safe-link.txt")))
	require.NoError(t, os.Symlink("../outside.txt", filepath.Join(srcRoot, "escape-link.txt")))

	require.NoError(t, copyLocalRepo(srcRoot, dstRoot))

	safeLinkPath := filepath.Join(dstRoot, "safe-link.txt")
	target, err := os.Readlink(safeLinkPath)
	require.NoError(t, err)
	resolved := filepath.Clean(filepath.Join(filepath.Dir(safeLinkPath), target))
	assert.Equal(t, filepath.Join(dstRoot, "safe.txt"), resolved)

	_, err = os.Lstat(filepath.Join(dstRoot, "escape-link.txt"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestCopyLocalRepoRewritesAbsoluteSymlinksIntoWorkspace(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	realFile := filepath.Join(srcRoot, "nested", "real.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(realFile), 0755))
	require.NoError(t, os.WriteFile(realFile, []byte("safe"), 0644))
	require.NoError(t, os.Symlink(realFile, filepath.Join(srcRoot, "absolute-link.txt")))

	require.NoError(t, copyLocalRepo(srcRoot, dstRoot))

	linkPath := filepath.Join(dstRoot, "absolute-link.txt")
	linkTarget, err := os.Readlink(linkPath)
	require.NoError(t, err)
	assert.False(t, filepath.IsAbs(linkTarget))

	resolved := filepath.Clean(filepath.Join(filepath.Dir(linkPath), linkTarget))
	assert.Equal(t, filepath.Join(dstRoot, "nested", "real.txt"), resolved)
}

func TestSanitizeRepoNameFallsBackAfterTrim(t *testing.T) {
	assert.Equal(t, "repo", sanitizeRepoName("!!!"))
	assert.Equal(t, "repo", sanitizeRepoName("   "))
	assert.Equal(t, "good-name", sanitizeRepoName("good-name"))
}

func TestBenchmarkScriptUsesShellFunctionForMCPCalls(t *testing.T) {
	root := testutil.RepoRoot(t)
	scriptPath := filepath.Join(root, "benchmarks", "script.sh")
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	text := string(content)
	assert.Contains(t, text, `time_cmd_quiet \
  mcp_query "get_file_structure"`)
	assert.NotContains(t, text, `| $CODEINDEX serve`)
	assert.NotContains(t, text, `bash -c "printf '{\"jsonrpc\"`)
}

func TestPathWithinRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "repo")

	ok, err := pathWithinRoot(root, filepath.Join(root, "nested", "file.txt"))
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = pathWithinRoot(root, filepath.Join(root, "..", "escape.txt"))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestResolveSymlinkTargetRejectsEscapes(t *testing.T) {
	srcRoot := t.TempDir()
	symlinkPath := filepath.Join(srcRoot, "nested", "link.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(symlinkPath), 0755))

	resolved, err := resolveSymlinkTarget(srcRoot, symlinkPath, "../../outside.txt")
	require.NoError(t, err)
	assert.Equal(t, "", resolved)

	insideTarget := filepath.Join(srcRoot, "nested", "real.txt")
	require.NoError(t, os.WriteFile(insideTarget, []byte("safe"), 0644))
	resolved, err = resolveSymlinkTarget(srcRoot, symlinkPath, insideTarget)
	require.NoError(t, err)
	canonicalTarget, err := filepath.EvalSymlinks(insideTarget)
	require.NoError(t, err)
	assert.Equal(t, canonicalTarget, resolved)
}

func TestRewriteSymlinkTargetProducesRelativePath(t *testing.T) {
	srcRoot := filepath.Join(string(filepath.Separator), "tmp", "src")
	dstRoot := filepath.Join(string(filepath.Separator), "tmp", "dst")
	dstPath := filepath.Join(dstRoot, "links", "alias.txt")
	resolvedTarget := filepath.Join(srcRoot, "nested", "real.txt")

	rewritten, err := rewriteSymlinkTarget(srcRoot, dstRoot, dstPath, resolvedTarget)
	require.NoError(t, err)
	assert.False(t, filepath.IsAbs(rewritten))
	assert.True(t, strings.HasSuffix(rewritten, filepath.Join("..", "nested", "real.txt")))
}

func TestResolveSourceCanonicalizesSymlinkedLocalRepo(t *testing.T) {
	realRoot := t.TempDir()
	linkParent := t.TempDir()
	linkRoot := filepath.Join(linkParent, "repo-link")

	require.NoError(t, os.Symlink(realRoot, linkRoot))

	source, err := resolveSource(linkRoot)
	require.NoError(t, err)

	canonicalPath, err := filepath.EvalSymlinks(realRoot)
	require.NoError(t, err)

	assert.Equal(t, filepath.Clean(linkRoot), source.original)
	assert.Equal(t, canonicalPath, source.local)
	assert.Equal(t, "repo-link", source.repoName)
}

func TestCopyLocalRepoKeepsAbsoluteLinksInsideCanonicalizedRoot(t *testing.T) {
	realRoot := t.TempDir()
	linkParent := t.TempDir()
	linkRoot := filepath.Join(linkParent, "repo-link")
	dstRoot := t.TempDir()

	realFile := filepath.Join(realRoot, "nested", "real.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(realFile), 0755))
	require.NoError(t, os.WriteFile(realFile, []byte("safe"), 0644))
	require.NoError(t, os.Symlink(realFile, filepath.Join(realRoot, "absolute-link.txt")))
	require.NoError(t, os.Symlink(realRoot, linkRoot))

	source, err := resolveSource(linkRoot)
	require.NoError(t, err)
	require.NoError(t, copyLocalRepo(source.local, dstRoot))

	linkPath := filepath.Join(dstRoot, "absolute-link.txt")
	linkTarget, err := os.Readlink(linkPath)
	require.NoError(t, err)

	resolved := filepath.Clean(filepath.Join(filepath.Dir(linkPath), linkTarget))
	assert.Equal(t, filepath.Join(dstRoot, "nested", "real.txt"), resolved)
}
