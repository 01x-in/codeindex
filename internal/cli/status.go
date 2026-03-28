package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index health summary",
	Long: `Prints: total files indexed, stale file count, last full reindex timestamp,
and list of changed files since last index.`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().Bool("json", false, "Output as JSON")
}

// StatusOutput is the JSON output for status.
type StatusOutput struct {
	FilesIndexed int      `json:"files_indexed"`
	FilesFresh   int      `json:"files_fresh"`
	FilesStale   int      `json:"files_stale"`
	Nodes        int      `json:"nodes"`
	Edges        int      `json:"edges"`
	LastReindex  string   `json:"last_reindex"`
	StaleFiles   []string `json:"stale_files"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Load config.
	cfg, _, err := config.LoadOrDetect(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Open the graph store.
	dbPath := filepath.Join(dir, cfg.IndexPath, "graph.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if jsonFlag {
			data, _ := json.MarshalIndent(StatusOutput{}, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No index found. Run 'code-index init' to get started.")
		}
		return nil
	}

	store, err := graph.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		return fmt.Errorf("migrating schema: %w", err)
	}

	// Gather metadata.
	allMeta, err := store.GetAllFileMetadata()
	if err != nil {
		return fmt.Errorf("reading metadata: %w", err)
	}

	// Check staleness.
	staleFiles, err := indexer.GetStaleFiles(store, dir)
	if err != nil {
		return fmt.Errorf("checking staleness: %w", err)
	}

	filesIndexed := len(allMeta)
	filesStale := len(staleFiles)
	filesFresh := filesIndexed - filesStale

	nodeCount, _ := store.NodeCount()
	edgeCount, _ := store.EdgeCount()

	lastReindex := "never"
	if val, err := store.GetIndexMetadata("last_full_reindex"); err == nil && val != "" {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			lastReindex = humanizeTime(t)
		}
	}

	output := StatusOutput{
		FilesIndexed: filesIndexed,
		FilesFresh:   filesFresh,
		FilesStale:   filesStale,
		Nodes:        nodeCount,
		Edges:        edgeCount,
		LastReindex:  lastReindex,
		StaleFiles:   staleFiles,
	}

	if jsonFlag {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Human-readable output.
	fmt.Fprintln(cmd.OutOrStdout(), "Code Index Status")
	fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 17))
	fmt.Fprintf(cmd.OutOrStdout(), "Files indexed:  %d\n", filesIndexed)
	fmt.Fprintf(cmd.OutOrStdout(), "  Fresh:        %d\n", filesFresh)
	fmt.Fprintf(cmd.OutOrStdout(), "  Stale:        %d\n", filesStale)
	fmt.Fprintf(cmd.OutOrStdout(), "Nodes:          %d\n", nodeCount)
	fmt.Fprintf(cmd.OutOrStdout(), "Edges:          %d\n", edgeCount)
	fmt.Fprintf(cmd.OutOrStdout(), "Last reindex:   %s\n", lastReindex)

	if len(staleFiles) > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Stale files:")
		limit := 20
		for i, f := range staleFiles {
			if i >= limit {
				fmt.Fprintf(cmd.OutOrStdout(), "  ... and %d more\n", len(staleFiles)-limit)
				break
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f)
		}
	}

	return nil
}

// humanizeTime returns a human-readable relative time.
func humanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return t.Format("2006-01-02 15:04:05")
	}
}
