// Package config provides configuration management for rcode.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Paths defines standard configuration file paths
type Paths struct {
	ServerConfig string
	ClientConfig string
	LogDir       string
}

// GetDefaultPaths returns the default configuration paths
func GetDefaultPaths() Paths {
	homeDir, _ := os.UserHomeDir()

	return Paths{
		ServerConfig: filepath.Join(homeDir, ".config", "rcode", "server-config.yaml"),
		ClientConfig: filepath.Join(homeDir, ".config", "rcode", "config.yaml"),
		LogDir:       filepath.Join(homeDir, ".local", "share", "rcode", "logs"),
	}
}

// loadConfig is a generic function to load configuration from file
func loadConfig(path, defaultPath string, createDefault func() error) ([]byte, error) {
	if path == "" {
		path = defaultPath
	}

	// Create default config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefault(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	// Path is from user configuration or command-line argument
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return data, nil
}

// LoadServerConfig loads server configuration from file
func LoadServerConfig(path string) (*ServerConfigFile, error) {
	defaultPath := GetDefaultPaths().ServerConfig

	data, err := loadConfig(path, defaultPath, func() error {
		config := GetDefaultServerConfig()
		return SaveServerConfig(defaultPath, config)
	})
	if err != nil {
		// If we failed to create default, return the default anyway
		if os.IsNotExist(err) {
			return GetDefaultServerConfig(), nil
		}
		return nil, err
	}

	var config ServerConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values
	applyServerDefaults(&config)

	return &config, nil
}

// LoadClientConfig loads client configuration from file
func LoadClientConfig(path string) (*ClientConfig, error) {
	defaultPath := GetDefaultPaths().ClientConfig
	configPath := path
	if configPath == "" {
		configPath = defaultPath
	}

	data, err := loadConfig(path, defaultPath, func() error {
		config := GetDefaultClientConfig()
		return SaveClientConfig(defaultPath, config)
	})
	if err != nil {
		// If we failed to create default, return the default anyway
		if os.IsNotExist(err) {
			return GetDefaultClientConfig(), nil
		}
		return nil, err
	}

	// First, parse legacy fields from the raw data
	var legacy legacyClientConfig
	_ = yaml.Unmarshal(data, &legacy) // Ignore errors, just capture what we can

	// Parse into new config structure
	var config ClientConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Migrate legacy fields to new format
	legacyWarnings := MigrateFromLegacy(&legacy, &config)

	// Run additional migrations
	warnings := MigrateClientConfig(&config)

	// Apply defaults for missing values
	applyClientDefaults(&config)

	// If legacy fields were migrated, auto-save the new format
	if len(legacyWarnings) > 0 {
		if err := autoMigrateConfigFile(configPath, &config, legacyWarnings); err != nil {
			// Print warnings if auto-migration failed
			fmt.Fprintf(os.Stderr, "Warning: Failed to auto-migrate config file: %v\n", err)
			PrintMigrationWarnings(legacyWarnings)
		}
	}

	// Print any additional migration warnings
	PrintMigrationWarnings(warnings)

	return &config, nil
}

// autoMigrateConfigFile backs up the old config and saves the new format
func autoMigrateConfigFile(configPath string, config *ClientConfig, warnings []MigrationWarning) error {
	// Create backup path
	backupPath := configPath + ".bak"

	// Check if backup already exists (don't overwrite previous backups)
	if _, err := os.Stat(backupPath); err == nil {
		// Backup exists, use numbered backup
		for i := 1; i < 100; i++ {
			backupPath = fmt.Sprintf("%s.bak.%d", configPath, i)
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				break
			}
		}
	}

	// Read original file for backup
	// configPath is from user configuration or default path, not external input
	originalData, err := os.ReadFile(configPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to read original config: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, originalData, 0o600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Save new format
	if err := SaveClientConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	// Print migration success message
	fmt.Fprintf(os.Stderr, "Config file migrated to new format.\n")
	fmt.Fprintf(os.Stderr, "  Backup saved to: %s\n", backupPath)
	fmt.Fprintf(os.Stderr, "  Migrated fields:\n")
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "    - %s: %s\n", w.Field, w.Message)
	}

	return nil
}

// SaveServerConfig saves server configuration to file
func SaveServerConfig(path string, config *ServerConfigFile) error {
	if path == "" {
		path = GetDefaultPaths().ServerConfig
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveClientConfig saves client configuration to file
func SaveClientConfig(path string, config *ClientConfig) error {
	if path == "" {
		path = GetDefaultPaths().ClientConfig
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// MergeClientWithEnvironment merges environment variables into client configuration
func MergeClientWithEnvironment(config *ClientConfig) {
	// Run migration for environment variables (handles deprecation warnings)
	warnings := MigrateClientEnvironment(config)
	PrintMigrationWarnings(warnings)

	// Fallback host
	if fallbackHost := os.Getenv("RCODE_FALLBACK_HOST"); fallbackHost != "" {
		config.Hosts.Server.Fallback = fallbackHost
	}

	// Timeout
	if timeout := os.Getenv("RCODE_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Network.Timeout = d
		}
	}

	// Editor configuration
	if editor := os.Getenv("RCODE_EDITOR"); editor != "" {
		config.DefaultEditor = editor
	}

	// Logging configuration
	if logLevel := os.Getenv("RCODE_LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = strings.ToLower(logLevel)
	}
}

// GetDefaultServerConfig returns default server configuration
func GetDefaultServerConfig() *ServerConfigFile {
	paths := GetDefaultPaths()
	return &ServerConfigFile{
		Server: ServerConfig{
			Host:         DefaultServerHost,
			Port:         DefaultServerPort,
			ReadTimeout:  DefaultReadTimeout,
			WriteTimeout: DefaultWriteTimeout,
			IdleTimeout:  DefaultIdleTimeout,
			AllowedIPs:   []string{},
		},
		Editors: []EditorConfig{
			{
				Name:      "cursor",
				Command:   "cursor --remote ssh-remote+{user}@{host} {path}",
				Default:   true,
				Available: true,
			},
			{
				Name:      "vscode",
				Command:   "code --remote ssh-remote+{user}@{host} {path}",
				Default:   false,
				Available: true,
			},
			{
				Name:      "zed",
				Command:   "zed ssh://{user}@{host}/{path}",
				Default:   false,
				Available: true,
			},
			{
				Name:      "nvim",
				Command:   "nvim scp://{user}@{host}/{path}",
				Default:   false,
				Available: true,
			},
		},
		Logging: LogConfig{
			Level:      DefaultLogLevel,
			File:       filepath.Join(paths.LogDir, "server.log"),
			MaxSize:    DefaultLogMaxSize,
			MaxBackups: DefaultLogMaxBackups,
			MaxAge:     DefaultLogMaxAge,
			Compress:   true,
			Console:    true,
		},
	}
}

// GetDefaultClientConfig returns default client configuration
func GetDefaultClientConfig() *ClientConfig {
	paths := GetDefaultPaths()
	return &ClientConfig{
		Hosts: HostsConfig{
			Server: ServerHostConfig{
				Primary:  "192.168.1.100",
				Fallback: "100.64.0.1",
			},
			SSH: SSHHostConfig{
				Host: "", // Empty = auto-detect
				AutoDetect: AutoDetectConfig{
					Tailscale:        true,
					TailscalePattern: "{hostname-}tail",
				},
			},
		},
		Network: ClientNetworkConfig{
			Timeout:       DefaultTimeout,
			RetryAttempts: DefaultRetryAttempts,
			RetryDelay:    DefaultRetryDelay,
		},
		FallbackEditors: GetDefaultFallbackEditors(),
		DefaultEditor:   "cursor",
		Logging: LogConfig{
			Level:      DefaultLogLevel,
			File:       filepath.Join(paths.LogDir, "client.log"),
			MaxSize:    DefaultLogMaxSize,
			MaxBackups: DefaultLogMaxBackups,
			MaxAge:     DefaultLogMaxAge,
			Compress:   true,
			Console:    true,
		},
	}
}

// applyServerDefaults applies default values to missing server config fields
func applyServerDefaults(config *ServerConfigFile) {
	if config.Server.Host == "" {
		config.Server.Host = DefaultServerHost
	}
	if config.Server.Port == 0 {
		config.Server.Port = DefaultServerPort
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = DefaultReadTimeout
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = DefaultWriteTimeout
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = DefaultIdleTimeout
	}

	applyLogDefaults(&config.Logging, "server.log")
}

// applyClientDefaults applies default values to missing client config fields
func applyClientDefaults(config *ClientConfig) {
	if config.Network.Timeout == 0 {
		config.Network.Timeout = DefaultTimeout
	}
	if config.Network.RetryAttempts == 0 {
		config.Network.RetryAttempts = DefaultRetryAttempts
	}
	if config.Network.RetryDelay == 0 {
		config.Network.RetryDelay = DefaultRetryDelay
	}

	applyLogDefaults(&config.Logging, "client.log")
}

// applyLogDefaults applies default values to logging config
func applyLogDefaults(config *LogConfig, defaultFile string) {
	if config.Level == "" {
		config.Level = DefaultLogLevel
	}
	if config.File == "" {
		paths := GetDefaultPaths()
		config.File = filepath.Join(paths.LogDir, defaultFile)
	}
	if config.MaxSize == 0 {
		config.MaxSize = DefaultLogMaxSize
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = DefaultLogMaxBackups
	}
	if config.MaxAge == 0 {
		config.MaxAge = DefaultLogMaxAge
	}
}
