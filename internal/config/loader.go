package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigPaths defines standard configuration file paths
type ConfigPaths struct {
	ServerConfig string
	ClientConfig string
	LogDir       string
}

// GetDefaultPaths returns the default configuration paths
func GetDefaultPaths() ConfigPaths {
	homeDir, _ := os.UserHomeDir()

	return ConfigPaths{
		ServerConfig: filepath.Join(homeDir, ".config", "rcode", "server-config.yaml"),
		ClientConfig: filepath.Join(homeDir, ".config", "rcode", "config.yaml"),
		LogDir:       filepath.Join(homeDir, ".local", "share", "rcode", "logs"),
	}
}

// LoadServerConfig loads server configuration from file
func LoadServerConfig(path string) (*ServerConfigFile, error) {
	if path == "" {
		path = GetDefaultPaths().ServerConfig
	}

	// Create default config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		config := GetDefaultServerConfig()
		if err := SaveServerConfig(path, config); err != nil {
			return config, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
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
	if path == "" {
		path = GetDefaultPaths().ClientConfig
	}

	// Create default config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		config := GetDefaultClientConfig()
		if err := SaveClientConfig(path, config); err != nil {
			return config, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values
	applyClientDefaults(&config)

	return &config, nil
}

// SaveServerConfig saves server configuration to file
func SaveServerConfig(path string, config *ServerConfigFile) error {
	if path == "" {
		path = GetDefaultPaths().ServerConfig
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// MergeWithEnvironment merges environment variables into configuration
func MergeWithEnvironment(config *Config) {
	// Server configuration
	if host := os.Getenv("RCODE_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("RCODE_PORT"); port != "" {
		if p, err := parseInt(port); err == nil {
			config.Server.Port = p
		}
	}

	// Network configuration
	if primaryHost := os.Getenv("RCODE_PRIMARY_HOST"); primaryHost != "" {
		config.Network.PrimaryHost = primaryHost
	}
	if fallbackHost := os.Getenv("RCODE_FALLBACK_HOST"); fallbackHost != "" {
		config.Network.FallbackHost = fallbackHost
	}
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
	if logFile := os.Getenv("RCODE_LOG_FILE"); logFile != "" {
		config.Logging.File = logFile
	}
}

// MergeClientWithEnvironment merges environment variables into client configuration
func MergeClientWithEnvironment(config *ClientConfig) {
	// Network configuration
	if primaryHost := os.Getenv("RCODE_HOST"); primaryHost != "" {
		config.Network.PrimaryHost = primaryHost
	}
	if fallbackHost := os.Getenv("RCODE_FALLBACK_HOST"); fallbackHost != "" {
		config.Network.FallbackHost = fallbackHost
	}
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
		Network: NetworkConfig{
			PrimaryHost:   "192.168.1.100",
			FallbackHost:  "100.64.0.1",
			Timeout:       DefaultTimeout,
			RetryAttempts: DefaultRetryAttempts,
			RetryDelay:    DefaultRetryDelay,
		},
		DefaultEditor: "cursor",
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

// parseInt is a helper to parse integer from string
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
