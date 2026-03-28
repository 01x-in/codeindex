package cli

import (
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [symbol]",
	Short: "Interactive TUI tree view of the knowledge graph",
	Long: `Renders an interactive tree view rooted at a symbol or file.
Navigate with arrow keys, expand/collapse with Enter, search with /.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: M2 implementation
		return nil
	},
}

func init() {
	treeCmd.Flags().String("file", "", "Show file structure tree instead of symbol tree")
	treeCmd.Flags().Bool("json", false, "Output tree as JSON (non-interactive)")
}
