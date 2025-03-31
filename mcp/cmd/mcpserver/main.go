package main

import (
	mcpserver "0xnfwtiventory/internal/app"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd.Execute()
}

var c mcpserver.Config
var logger zerolog.Logger = zerolog.New(os.Stdout)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wtInventoryMcp",
	Short: "MCP server configuration",
	RunE: func(cmd *cobra.Command, args []string) error {

		// Only set config values for flags that were actually provided
		if cmd.Flags().Changed("log-path") {
			logPath, _ := cmd.Flags().GetString("log-path")
			c.LogPath = &logPath
		}
		if cmd.Flags().Changed("min-log-level") {
			minLogLevel, _ := cmd.Flags().GetString("min-log-level")
			c.MinLogLevel = &minLogLevel
		}
		if cmd.Flags().Changed("cli-path") {
			cliPath, _ := cmd.Flags().GetString("cli-path")
			c.CLIPath = &cliPath
		}
		if cmd.Flags().Changed("web-server-address") {
			webServerAddress, _ := cmd.Flags().GetString("web-server-address")
			c.WebServerAddress = &webServerAddress
		}

		logger.Info().Msg("Loading configurations...")
		cc := c.Compose(mcpserver.ComposeConfig())
		logger.Info().Msg("Loading server...")
		loadServer, err := mcpserver.LoadServer(cc)
		if err != nil {
			return fmt.Errorf("server error: %w\n", err)
		}
		// Start the server
		logger.Info().Msg("Starting WTInventory MCP Server")
		if err := server.ServeStdio(loadServer.Mcp); err != nil {
			return fmt.Errorf("server error: %w\n", err)
		}

		return nil
	},
}

func init() {

	// Define flags in init()
	rootCmd.Flags().String("log-path", "", "Path on the local filesystem for logging information")
	rootCmd.Flags().String("min-log-level", "", "Minimum level to log items (defaults to Info if not set)")
	rootCmd.Flags().String("cli-path", "", "Path for the CLI")
	rootCmd.Flags().String("web-server-address", "", "Address for the web server")

}
