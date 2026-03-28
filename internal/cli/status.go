package cli

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index health summary",
	Long: `Prints: total files indexed, stale file count, last full reindex timestamp,
and list of changed files since last index.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: M1-S7 implementation
		return nil
	},
}

func init() {
	statusCmd.Flags().Bool("json", false, "Output as JSON")
}
