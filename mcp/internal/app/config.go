package mcpserver

import (
	"encoding/json"
	"inventory_shared/wtlogger"
	"inventory_shared/xdg"
	"os"

	"github.com/rs/zerolog"
)

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

func (c Config) Compose(other Config) Config {
	logger := wtlogger.GetLogger()
	logger.Debug("Composing Config file...")
	if c.LogPath == nil && other.LogPath != nil {
		logger.Debug("Found overriden logPath")
		c.LogPath = other.LogPath
	}
	if c.MinLogLevel == nil && other.MinLogLevel != nil {
		logger.Debug("Found overriden Min Log Level")
		c.MinLogLevel = other.MinLogLevel
	}
	if c.CLIPath == nil && other.CLIPath != nil {
		logger.Debug("Found overriden CLI Path")
		c.CLIPath = other.CLIPath
	}
	if c.WebServerAddress == nil && other.WebServerAddress != nil {
		logger.Debug("Found overriden WebServerAddress")
		c.WebServerAddress = other.WebServerAddress
	}
	return c
}

// LogLevel returns the zerolog.Level version of the log level of the `MinLogLevel` string for this Conf object
//
// `nil`, or any other uninterpretable value is considered to be `zerolog.InfoLevel`
func (conf *Config) LogLevel() zerolog.Level {
	level := conf.MinLogLevel
	if level == nil {
		return zerolog.InfoLevel
	} else if *level == zerolog.Disabled.String() {
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
	config_paths := xdg.GetConfigPaths()
	for _, p := range config_paths {
		var c Config
		bytes, err := os.ReadFile(p)
		if err == nil {
			err = json.Unmarshal(bytes, &c)
			if err != nil {
				conf = conf.Compose(c)
			}
		}
	}

	return conf
}

// NewConfig returns a new, blank configuration object
func NewConfig() Config {
	return Config{}
}
