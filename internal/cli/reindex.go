package cli

import (
	"github.com/spf13/cobra"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex [file]",
	Short: "Re-index stale files or a specific file",
	Long: `Re-index all stale files (incremental via hash comparison) or a single file.
Use --watch to start a watcher that auto-reindexes on file save.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: M1-S6 implementation
		return nil
	},
}

func init() {
	reindexCmd.Flags().Bool("watch", false, "Watch mode: auto-reindex on file save")
}
