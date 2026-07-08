package cmd

import (
	"fmt"
	"os"

	"github.com/justmiles/go-markdown2confluence/mcpserver"
	"github.com/spf13/cobra"
)

// mcpCmd starts the MCP server
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as an MCP server over stdio",
	Long: `Run markdown2confluence as a Model Context Protocol (MCP) server.

The server exposes tools that wrap the existing CLI functionality, including a
space-key lookup tool for when only a Confluence page ID is known. Confluence
credentials are read from the usual environment variables:

  CONFLUENCE_USERNAME
  CONFLUENCE_PASSWORD
  CONFLUENCE_ACCESS_TOKEN
  CONFLUENCE_ENDPOINT
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := mcpserver.RunServer(rootCmd.Version); err != nil {
			fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
