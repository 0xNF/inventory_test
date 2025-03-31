package mcpserver

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// import logger

// Config holds configuration for how the MCP server should behave
type Config struct {
	// LogPath denotes where on the local filesystem the logging informastion file should be
	LogPath *string `json:"LogPath"`
	// MinLogLevel is the minimum level to log items to file for.
	// N.B.: If this field is set to null, it is interprted as Info. If this field is set to Debug or Trace, then the
	// whole applicatiom is considered to be running in Debug Mode
	MinLogLevel      *string `json:"MinLogLevel"`
	CLIPath          *string `json:"CLIPath"`
	WebServerAddress *string `json:"WebServerAddress"`
}

func (c Config) compose(other Config) Config {
	if c.LogPath == nil && other.LogPath != nil {
		c.LogPath = other.LogPath
	}
	if c.MinLogLevel == nil && other.MinLogLevel != nil {
		c.MinLogLevel = other.MinLogLevel
	}
	if c.CLIPath == nil && other.CLIPath != nil {
		c.CLIPath = other.CLIPath
	}
	if c.WebServerAddress == nil && other.WebServerAddress != nil {
		c.WebServerAddress = other.WebServerAddress
	}
	return c
}

// LogLevel returns the zerolog.Level version of the log level of the `MinLogLevel` string for this Conf object
//
// `nil`, or any other uninterpretable value is considered to be `zerolog.InfoLevel`
func (conf *Config) LogLevel() zerolog.Level {
	level := conf.MinLogLevel
	if *level == zerolog.Disabled.String() {
		return zerolog.Disabled
	} else if *level == zerolog.TraceLevel.String() {
		return zerolog.TraceLevel
	} else if *level == zerolog.DebugLevel.String() {
		return zerolog.DebugLevel
	} else if *level == zerolog.WarnLevel.String() {
		return zerolog.WarnLevel
	} else if *level == zerolog.ErrorLevel.String() {
		return zerolog.ErrorLevel
	} else if *level == zerolog.FatalLevel.String() {
		return zerolog.FatalLevel
	} else if *level == zerolog.PanicLevel.String() {
		return zerolog.PanicLevel
	} else {
		return zerolog.InfoLevel
	}
}

// IsDebugMode returns whether this configuration is launched in Debug Mode,
// which is deinfed as the MinLogLevel being Debug or Trace (aka <= 0)
func (conf *Config) IsDebugMode() bool {
	return int8(conf.LogLevel()) <= 0
}

func ComposeConfig() Config {
	conf := NewConfig()

	// iterate over the patsh in reverse order, composing as we go down
	config_paths := GetConfigPaths()
	for _, p := range config_paths {
		var c Config
		bytes, err := os.ReadFile(p)
		if err == nil {
			err = json.Unmarshal(bytes, &c)
			if err != nil {
				conf = conf.compose(c)
			}
		}
	}

	return conf
}

// LoadConfig loads a configuration object, using XDG rules to find the values.
//
// If no config is found, or loading returned an error, `NewConfig` will be returned
func LoadConfigFromDisk() Config {
	conf := NewConfig()

	// Try multiple locations in order of priority
	config_paths := GetConfigPaths()
	for _, p := range config_paths {
		f, err := os.Open(p)
		if err != nil {
			bytes, err := io.ReadAll(f)
			if err != nil {
				err = json.Unmarshal(bytes, &conf)
				if err != nil {
					return conf
				}
			}
		}
	}

	// Return default if no config file found
	return conf
}

// GetConfigPaths returns the list of possible config file paths following XDG convention
func GetConfigPaths() []string {
	const envKeyConfig = "0XNFWT_INVENTORY_MCP_CONFIG"
	const configJSONName = "0xnfwt_inventory_mcp.json"
	const xdgConfigDir = "0xnfwt_inventory_mcp"

	var paths []string

	// First check for environment variable
	if path, exists := os.LookupEnv(envKeyConfig); exists {
		paths = append(paths, path)
	}

	// Then check XDG_CONFIG_HOME or ~/.config
	if configDir := getConfigDir(); configDir != "" {
		xdgPath := filepath.Join(configDir, xdgConfigDir, configJSONName)
		paths = append(paths, xdgPath)
	}

	// Check home directory
	if homeDir := getHomeDir(); homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, configJSONName))
	}

	// Check relative to executable
	paths = append(paths, configJSONName)

	// Check current directory
	paths = append(paths, configJSONName)

	return paths
}

// getConfigDir returns the XDG_CONFIG_HOME directory or ~/.config if not set
func getConfigDir() string {
	if xdgConfigHome, exists := os.LookupEnv("XDG_CONFIG_HOME"); exists {
		return xdgConfigHome
	}
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".config")
	}
	return ""
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

// NewConfig returns a new, blank configuration object
func NewConfig() Config {
	return Config{}
}
