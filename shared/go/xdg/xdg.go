package xdg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type XDGKeys struct {
	AppName        string
	ConfigJsonName string
}

var XdgKeys XDGKeys

func InitializeXDG(keys XDGKeys) {
	XdgKeys = keys
}

// GetConfigPaths returns the list of possible config file paths following XDG convention
func GetConfigPaths() []string {
	envKeyConfig := fmt.Sprintf("%s_CONFIG", strings.ToUpper(XdgKeys.AppName))
	configJSONName := XdgKeys.ConfigJsonName
	xdgConfigDir := XdgKeys.AppName

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

// GetDataPaths returns the list of possible data file paths following XDG convention
func GetDataPaths() []string {
	envKeyData := fmt.Sprintf("%s_DATA", strings.ToUpper(XdgKeys.AppName))
	xdgDataDir := XdgKeys.AppName

	var paths []string

	// First check for environment variable
	if path, exists := os.LookupEnv(envKeyData); exists {
		paths = append(paths, path)
	}

	// Then check XDG_DATA_HOME or ~/.local/share
	if dataDir := getDataDir(); dataDir != "" {
		xdgPath := filepath.Join(dataDir, xdgDataDir)
		paths = append(paths, xdgPath)
	}

	// Check home directory
	if homeDir := getHomeDir(); homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, xdgDataDir))
	}

	// Check relative to executable
	paths = append(paths, xdgDataDir)

	return paths
}

// GetCachePaths returns the list of possible cache folder paths following XDG convention
func GetCachePaths() []string {
	envKeyCache := fmt.Sprintf("%s_CACHE", strings.ToUpper(XdgKeys.AppName))
	xdgCacheDir := XdgKeys.AppName

	var paths []string

	// First check for environment variable
	if path, exists := os.LookupEnv(envKeyCache); exists {
		paths = append(paths, path)
	}

	// Then check XDG_CACHE_HOME or ~/.cache
	if cacheDir := getCacheDir(); cacheDir != "" {
		xdgPath := filepath.Join(cacheDir, xdgCacheDir)
		paths = append(paths, xdgPath)
	}

	// Check home directory
	if homeDir := getHomeDir(); homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, xdgCacheDir))
	}

	// Check relative to executable
	paths = append(paths, xdgCacheDir)

	return paths
}

// GetRuntimePaths returns the list of possible runtime file paths following XDG convention
func GetRuntimePaths() []string {
	envKeyRuntime := fmt.Sprintf("%s_RUNTIME", strings.ToUpper(XdgKeys.AppName))
	xdgRuntimeDir := XdgKeys.AppName

	var paths []string

	// First check for environment variable
	if path, exists := os.LookupEnv(envKeyRuntime); exists {
		paths = append(paths, path)
	}

	// Then check XDG_RUNTIME_DIR
	if runtimeDir := getRuntimeDir(); runtimeDir != "" {
		xdgPath := filepath.Join(runtimeDir, xdgRuntimeDir)
		paths = append(paths, xdgPath)
	}

	// Check /tmp as fallback when XDG_RUNTIME_DIR is not available
	tmpPath := filepath.Join(os.TempDir(), xdgRuntimeDir)
	paths = append(paths, tmpPath)

	return paths
}

// getDataDir returns the XDG_DATA_HOME directory or ~/.local/share if not set
func getDataDir() string {
	if xdgDataHome, exists := os.LookupEnv("XDG_DATA_HOME"); exists {
		return xdgDataHome
	}
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".local", "share")
	}
	return ""
}

// getCacheDir returns the XDG_CACHE_HOME directory or ~/.cache if not set
func getCacheDir() string {
	if xdgCacheHome, exists := os.LookupEnv("XDG_CACHE_HOME"); exists {
		return xdgCacheHome
	}
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".cache")
	}
	return ""
}

// getRuntimeDir returns the XDG_RUNTIME_DIR directory
func getRuntimeDir() string {
	if xdgRuntimeDir, exists := os.LookupEnv("XDG_RUNTIME_DIR"); exists {
		return xdgRuntimeDir
	}
	return ""
}
