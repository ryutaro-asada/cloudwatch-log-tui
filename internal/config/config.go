// Package config provides configuration management for the CloudWatch Log TUI application.
// It handles application settings, logging configuration, and other runtime parameters.
package config

import (
	"os"
	// "path/filepath"
)

// Config holds the application configuration
type Config struct {
	LogFile string
}

// New creates a new configuration with default values
func New() *Config {
	// homeDir, err := os.UserHomeDir()
	// if err != nil {
	// 	homeDir = "."
	// }

	return &Config{
		LogFile: "cloudwatch-log-tui.log",
	}
}

// InitLogging initializes the application logging
func (c *Config) InitLogging() (*os.File, error) {
	return os.OpenFile(c.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
}
