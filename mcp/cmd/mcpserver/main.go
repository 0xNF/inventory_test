package main

import (
	mcpserver "0xnfwtiventory/internal/app"
	"inventory_shared/wtlogger"
	"inventory_shared/xdg"
	"path"

	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

const serverName = "0xNFWT Inventory Manager"
const serverVersion = "0.1.0"

func main() {
	rootCmd.Execute()
}

func setup() {
	var keys = xdg.XDGKeys{
		AppName:        "0xnfwt_inventory_mcp",
		ConfigJsonName: "0xnfwt_inventory_mcp.json",
	}
	xdg.InitializeXDG(keys)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wtInventoryMcp",
	Short: "MCP server configuration",
	RunE: func(cmd *cobra.Command, args []string) error {

		setup()
		logger := wtlogger.GetLogger()

		var c mcpserver.WTServerConfig

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

		logger.Info("Loading configurations...")
		c = c.Compose(mcpserver.ComposeConfig())
		if c.LogPath == nil || *c.LogPath == "" {
			lpaths := xdg.GetDataPaths()
			logname := fmt.Sprintf("%s.log", xdg.XdgKeys.AppName)
			if len(lpaths) == 0 {
				c.LogPath = &logname
			} else {
				p := path.Join(lpaths[0], logname)
				c.LogPath = &p
			}
		}
		logger.Initialize(wtlogger.LogConfig{
			FilePath:    c.LogPath,
			MinLogLevel: c.LogLevel(),
			Console:     true,
		})

		logger.Info("Loading server...")
		loadServer, err := mcpserver.LoadServer(serverName, serverVersion, c)
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		// Start the server
		logger.Info("Starting WTInventory MCP Server")
		if err := server.ServeStdio(loadServer.Mcp); err != nil {
			return fmt.Errorf("server error: %w", err)
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
