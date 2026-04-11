package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/01x-in/codeindex/internal/config"
	"github.com/01x-in/codeindex/internal/graph"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTreeWithoutArgsOutputsRepoTreeJSON(t *testing.T) {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".codeindex")
	require.NoError(t, os.MkdirAll(indexDir, 0755))

	cfg := config.DefaultConfig()
	cfg.Languages = []string{"go"}
	require.NoError(t, cfg.Save(filepath.Join(dir, config.ConfigFileName)))

	store, err := graph.NewSQLiteStore(filepath.Join(indexDir, "graph.db"))
	require.NoError(t, err)
	defer store.Close()
	require.NoError(t, store.Migrate())

	_, err = store.UpsertNode(graph.Node{
		Name: "Run", Kind: "fn", FilePath: "cmd/codeindex/main.go",
		LineStart: 5, LineEnd: 12, ColStart: 0, ColEnd: 10,
		Exported: true, Language: "go",
	})
	require.NoError(t, err)
	require.NoError(t, store.SetFileMetadata(graph.FileMetadata{
		FilePath:    "cmd/codeindex/main.go",
		ContentHash: "repo",
		Language:    "go",
		NodeCount:   1,
		EdgeCount:   0,
		IndexStatus: "ok",
	}))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	cmd := &cobra.Command{}
	cmd.Flags().String("file", "", "")
	cmd.Flags().Bool("json", true, "")
	cmd.Flags().Bool("color", false, "")

	var output bytes.Buffer
	cmd.SetOut(&output)

	err = runTree(cmd, nil)
	require.NoError(t, err)

	assert.Contains(t, output.String(), `"root"`)
	assert.Contains(t, output.String(), `"cmd/codeindex/main.go"`)
}
