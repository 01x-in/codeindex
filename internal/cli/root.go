package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via -ldflags.
	Version = "dev"

	rootCmd = &cobra.Command{
		Use:   "code-index",
		Short: "Persistent structural knowledge graph for codebases",
		Long: `Code Index builds a persistent knowledge graph of codebase structure
using ast-grep's tree-sitter parsing, exposing MCP tool primitives for
AI coding agents and a CLI tree explorer for developers.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("code-index %s\n", Version)
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(treeCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
