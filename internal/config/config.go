// Package config provides configuration management for the CloudWatch Log TUI application.
// It handles application settings, logging configuration, and other runtime parameters.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const appName = "cloudwatch-log-tui"

// Config holds the application configuration
type Config struct {
	LogFile string
}

// New creates a new configuration with default values.
// The log file lives in the user cache directory, or the temp directory when
// no cache directory is available.
func New() *Config {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}

	return &Config{
		LogFile: filepath.Join(dir, appName, appName+".log"),
	}
}

// InitLogging opens the log file for appending, creating it and its directory
// if needed. The caller must close the returned file.
func (c *Config) InitLogging() (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(c.LogFile), 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	return os.OpenFile(c.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
}
