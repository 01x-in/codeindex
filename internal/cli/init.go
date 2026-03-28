package cli

import (
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Auto-detect languages and initialize .code-index.yaml",
	Long: `Detects languages from project markers (package.json, go.mod, pyproject.toml,
Cargo.toml), proposes detected config, and writes .code-index.yaml on confirmation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: M1-S2 implementation
		return nil
	},
}

func init() {
	initCmd.Flags().Bool("yes", false, "Accept defaults without prompting")
}
