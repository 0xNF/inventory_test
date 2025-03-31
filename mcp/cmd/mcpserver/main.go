package main

import (
	mcpserver "0xnfwtiventory/internal/app"
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wtinventorymcp",
	Short: "Runs the WTInventory MCP server",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func main() {
	rootCmd.Execute()
	loadServer, err := mcpserver.LoadServer()
	if err != nil {
		fmt.Printf("Server error: %w\n", err.Error())
	}
	// Start the server
	if err := server.ServeStdio(loadServer.Mcp); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
