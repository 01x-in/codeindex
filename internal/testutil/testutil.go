package testutil

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// SkipIfNoAstGrep skips the test if ast-grep is not found in PATH.
func SkipIfNoAstGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Skip("ast-grep not found in PATH -- skipping integration test")
	}
}

// RepoRoot returns the absolute path to the repository root.
func RepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// testutil is at internal/testutil/testutil.go, so repo root is ../../
	return filepath.Join(filepath.Dir(filename), "..", "..")
}
