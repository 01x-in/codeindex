package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/01x/codeindex/internal/mcp"
	"github.com/01x/codeindex/internal/query"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP stdio JSON-RPC server",
	Long:  `Starts the MCP server over stdio for AI agent integration.`,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Load config.
	cfg, _, err := config.LoadOrDetect(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check ast-grep.
	if err := indexer.CheckInstalled(); err != nil {
		return err
	}

	// Open the graph store.
	dbPath := filepath.Join(dir, cfg.IndexPath, "graph.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("creating index directory: %w", err)
	}

	store, err := graph.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		return fmt.Errorf("migrating schema: %w", err)
	}

	engine := query.NewEngine(store, dir)

	// Create reindex callback.
	reindexFn := func(filePath string) error {
		runner := indexer.NewSubprocessRunner()
		if filePath != "" {
			// Validate the path stays within the project root.
			absPath := filepath.Join(dir, filePath)
			rel, err := filepath.Rel(dir, absPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				return fmt.Errorf("path traversal denied: %s is outside the project root", filePath)
			}

			lang := languageForFile(filePath, cfg.Languages)
			if lang == "" {
				return fmt.Errorf("cannot determine language for %s", filePath)
			}
			idx := indexer.NewIndexer(store, runner, dir, lang)
			_, err = idx.IndexFile(absPath)
			return err
		}
		// Full reindex.
		for _, lang := range cfg.Languages {
			idx := indexer.NewIndexer(store, runner, dir, lang)
			_, err := idx.IndexStale()
			if err != nil {
				return err
			}
		}
		return nil
	}

	server := mcp.NewServer(engine, reindexFn)
	return server.Serve()
}
