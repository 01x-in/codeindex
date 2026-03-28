package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/tui"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [symbol]",
	Short: "Interactive TUI tree view of the knowledge graph",
	Long: `Renders an interactive tree view rooted at a symbol or file.
Navigate with arrow keys, expand/collapse with Enter, search with /.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTree,
}

func init() {
	treeCmd.Flags().String("file", "", "Show file structure tree instead of symbol tree")
	treeCmd.Flags().Bool("json", false, "Output tree as JSON (non-interactive)")
	treeCmd.Flags().Bool("color", false, "Force color output")
}

func runTree(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fileFlag, _ := cmd.Flags().GetString("file")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	colorFlag, _ := cmd.Flags().GetBool("color")

	// Determine color mode: explicit flag or TTY detection.
	useColor := colorFlag || isTerminal()

	// Require either a symbol argument or --file flag.
	if len(args) == 0 && fileFlag == "" {
		return fmt.Errorf("provide a symbol name or use --file <path>")
	}

	// Load config.
	cfg, _, err := config.LoadOrDetect(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Open graph store.
	dbPath := filepath.Join(dir, cfg.IndexPath, "graph.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("no index found. Run 'codeindex init' to get started")
	}

	store, err := graph.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		return fmt.Errorf("migrating schema: %w", err)
	}

	builder := tui.NewSymbolTreeBuilder(store, dir)

	var root *tui.TreeNode
	var title string

	if fileFlag != "" {
		root, err = builder.BuildFileTree(fileFlag)
		if err != nil {
			return err
		}
		title = fmt.Sprintf("file: %s", fileFlag)
	} else {
		symbolName := args[0]
		root, err = builder.BuildSymbolTree(symbolName)
		if err != nil {
			return err
		}
		title = fmt.Sprintf("tree: %s", symbolName)
	}

	// JSON mode: output and exit (no TUI).
	if jsonFlag {
		return tui.PrintJSON(root, cmd.OutOrStdout())
	}

	// Interactive TUI mode.
	return tui.Run(root, title, useColor)
}

// isTerminal checks if stdout is a terminal.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
