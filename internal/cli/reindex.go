package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/01x/codeindex/internal/watcher"
	"github.com/spf13/cobra"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex [file]",
	Short: "Re-index stale files or a specific file",
	Long: `Re-index all stale files (incremental via hash comparison) or a single file.
Use --watch to start a watcher that auto-reindexes on file save.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReindex,
}

func init() {
	reindexCmd.Flags().Bool("watch", false, "Watch mode: auto-reindex on file save")
	reindexCmd.Flags().Bool("json", false, "Output as JSON")
}

// ReindexResult is the JSON output for reindex.
type ReindexResult struct {
	FilesReindexed int                    `json:"files_reindexed"`
	DurationMs     int64                  `json:"duration_ms"`
	Files          []indexer.IndexResult  `json:"files"`
}

func runReindex(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Load config.
	cfg, found, err := config.LoadOrDetect(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if !found && len(cfg.Languages) == 0 {
		return fmt.Errorf("no .codeindex.yaml found and no languages detected. Run 'codeindex init' first")
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

	// Watch mode.
	watchFlag, _ := cmd.Flags().GetBool("watch")
	if watchFlag {
		return runWatch(cmd, dir, cfg, store)
	}

	start := time.Now()
	runner := indexer.NewSubprocessRunner()

	if len(args) == 1 {
		// Single file reindex.
		return reindexSingleFile(cmd, dir, cfg, store, runner, args[0], jsonFlag, start)
	}

	// Full reindex (all stale files across all configured languages).
	return reindexAll(cmd, dir, cfg, store, runner, jsonFlag, start)
}

func reindexSingleFile(cmd *cobra.Command, dir string, cfg config.Config, store *graph.SQLiteStore, runner *indexer.SubprocessRunner, filePath string, jsonOutput bool, start time.Time) error {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(dir, filePath)
	}

	// Determine language from extension.
	lang := languageForFile(absPath, cfg.Languages)
	if lang == "" {
		return fmt.Errorf("cannot determine language for %s", filePath)
	}

	idx := indexer.NewIndexer(store, runner, dir, lang)
	result, err := idx.IndexFile(absPath)
	if err != nil {
		return fmt.Errorf("indexing %s: %w", filePath, err)
	}

	duration := time.Since(start)

	if jsonOutput {
		out := ReindexResult{
			FilesReindexed: 1,
			DurationMs:     duration.Milliseconds(),
			Files:          []indexer.IndexResult{result},
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Reindexed %s in %dms (+%d nodes, +%d edges)\n",
			result.FilePath, duration.Milliseconds(), result.NodeCount, result.EdgeCount)
	}

	return nil
}

func reindexAll(cmd *cobra.Command, dir string, cfg config.Config, store *graph.SQLiteStore, runner *indexer.SubprocessRunner, jsonOutput bool, start time.Time) error {
	var allResults []indexer.IndexResult

	// Clean up deleted files first.
	allMeta, err := store.GetAllFileMetadata()
	if err == nil {
		for _, meta := range allMeta {
			absPath := filepath.Join(dir, meta.FilePath)
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				store.DeleteFileData(meta.FilePath)
				allResults = append(allResults, indexer.IndexResult{
					FilePath: meta.FilePath,
					Status:   "deleted",
				})
			}
		}
	}

	// Index stale files for each configured language.
	for _, lang := range cfg.Languages {
		idx := indexer.NewIndexer(store, runner, dir, lang)
		results, err := idx.IndexStale()
		if err != nil {
			return fmt.Errorf("indexing %s files: %w", lang, err)
		}
		allResults = append(allResults, results...)
	}

	// Update last full reindex timestamp.
	store.SetIndexMetadata("last_full_reindex", time.Now().UTC().Format(time.RFC3339))

	duration := time.Since(start)

	if jsonOutput {
		out := ReindexResult{
			FilesReindexed: len(allResults),
			DurationMs:     duration.Milliseconds(),
			Files:          allResults,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Reindexed %d files in %dms\n", len(allResults), duration.Milliseconds())
		for _, r := range allResults {
			if r.Status == "deleted" {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s (deleted)\n", r.FilePath)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s (+%d nodes, +%d edges)\n", r.FilePath, r.NodeCount, r.EdgeCount)
			}
		}
	}

	return nil
}

// runWatch starts the fsnotify-based watch mode, auto-reindexing on file save.
func runWatch(cmd *cobra.Command, dir string, cfg config.Config, store *graph.SQLiteStore) error {
	runner := indexer.NewSubprocessRunner()

	fmt.Fprintln(cmd.OutOrStdout(), "Watching for changes... (Ctrl+C to stop)")

	onChange := func(absPath string) {
		lang := languageForFile(absPath, cfg.Languages)
		if lang == "" {
			return
		}

		start := time.Now()
		idx := indexer.NewIndexer(store, runner, dir, lang)
		result, err := idx.IndexFile(absPath)
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  error reindexing %s: %v\n", absPath, err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  -> Reindexed %s in %dms (+%d nodes, +%d edges)\n",
			result.FilePath, duration.Milliseconds(), result.NodeCount, result.EdgeCount)
	}

	w, err := watcher.New(dir, cfg, onChange)
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT / SIGTERM for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(cmd.OutOrStdout(), "\nStopping watcher...")
		cancel()
	}()

	if err := w.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("watcher error: %w", err)
	}
	return nil
}

// languageForFile determines the language based on file extension.
func languageForFile(path string, configuredLanguages []string) string {
	ext := filepath.Ext(path)
	langMap := map[string]string{
		".ts":  "typescript",
		".tsx": "typescript",
		".go":  "go",
		".py":  "python",
		".rs":  "rust",
	}

	lang, ok := langMap[ext]
	if !ok {
		return ""
	}

	// Check if this language is configured.
	for _, cl := range configuredLanguages {
		if cl == lang {
			return lang
		}
	}

	// If no languages are explicitly configured, allow any detected language.
	if len(configuredLanguages) == 0 {
		return lang
	}

	return ""
}
