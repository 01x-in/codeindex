package cli

import (
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP stdio JSON-RPC server",
	Long:  `Starts the MCP server over stdio for AI agent integration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: M1-S8 implementation
		return nil
	},
}
