package config

import (
	"time"
)

// EditorConfig represents configuration for a single editor
type EditorConfig struct {
	Name      string `yaml:"name" json:"name"`           // Editor name (e.g., "cursor", "vscode")
	Command   string `yaml:"command" json:"command"`     // Command template with placeholders
	Default   bool   `yaml:"default" json:"default"`     // Whether this is the default editor
	Available bool   `yaml:"available" json:"available"` // Whether the editor is available on the system
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	PrimaryHost   string        `yaml:"primary_host" json:"primary_host"`     // Primary host (e.g., LAN IP)
	FallbackHost  string        `yaml:"fallback_host" json:"fallback_host"`   // Fallback host (e.g., Tailscale IP)
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`               // Connection timeout
	RetryAttempts int           `yaml:"retry_attempts" json:"retry_attempts"` // Number of retry attempts
	RetryDelay    time.Duration `yaml:"retry_delay" json:"retry_delay"`       // Delay between retries
}

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	Host         string        `yaml:"host" json:"host"`                   // Server host to bind to
	Port         int           `yaml:"port" json:"port"`                   // Server port
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`   // HTTP read timeout
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"` // HTTP write timeout
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`   // HTTP idle timeout
	AllowedIPs   []string      `yaml:"allowed_ips" json:"allowed_ips"`     // IP whitelist (empty = allow all)
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level      string `yaml:"level" json:"level"`             // Log level (debug, info, warn, error)
	File       string `yaml:"file" json:"file"`               // Log file path
	MaxSize    int    `yaml:"max_size" json:"max_size"`       // Max size in MB before rotation
	MaxBackups int    `yaml:"max_backups" json:"max_backups"` // Max number of old log files
	MaxAge     int    `yaml:"max_age" json:"max_age"`         // Max age in days
	Compress   bool   `yaml:"compress" json:"compress"`       // Whether to compress old logs
	Console    bool   `yaml:"console" json:"console"`         // Whether to also log to console
}

// Config represents the complete configuration
type Config struct {
	// Common configuration
	Editors []EditorConfig `yaml:"editors" json:"editors"` // Available editors
	Logging LogConfig      `yaml:"logging" json:"logging"` // Logging configuration

	// Client-specific configuration
	Network       NetworkConfig `yaml:"network" json:"network"`               // Network configuration
	DefaultEditor string        `yaml:"default_editor" json:"default_editor"` // Default editor name

	// Server-specific configuration
	Server ServerConfig `yaml:"server" json:"server"` // Server configuration

	// Metadata
	Version    string    `yaml:"version" json:"version"`         // Config version
	LastUpdate time.Time `yaml:"last_update" json:"last_update"` // Last update timestamp
}

// ClientConfig represents client-specific configuration
type ClientConfig struct {
	Network       NetworkConfig  `yaml:"network" json:"network"`                       // Network configuration
	DefaultEditor string         `yaml:"default_editor" json:"default_editor"`         // Default editor name
	Editors       []EditorConfig `yaml:"editors,omitempty" json:"editors,omitempty"`   // Editor overrides
	Logging       LogConfig      `yaml:"logging" json:"logging"`                       // Logging configuration
	SSHHost       string         `yaml:"ssh_host,omitempty" json:"ssh_host,omitempty"` // Override SSH host for editor connection (e.g., LAN IP when using Tailscale SSH)
}

// ServerConfigFile represents server configuration file structure
type ServerConfigFile struct {
	Server  ServerConfig   `yaml:"server" json:"server"`   // Server configuration
	Editors []EditorConfig `yaml:"editors" json:"editors"` // Available editors
	Logging LogConfig      `yaml:"logging" json:"logging"` // Logging configuration
}

// Default configuration values
const (
	DefaultServerHost    = "0.0.0.0"
	DefaultServerPort    = 3000
	DefaultTimeout       = 2 * time.Second
	DefaultRetryAttempts = 3
	DefaultRetryDelay    = 500 * time.Millisecond
	DefaultLogLevel      = "info"
	DefaultLogMaxSize    = 10 // MB
	DefaultLogMaxBackups = 5
	DefaultLogMaxAge     = 30 // days
	DefaultReadTimeout   = 10 * time.Second
	DefaultWriteTimeout  = 10 * time.Second
	DefaultIdleTimeout   = 120 * time.Second
)

// GetDefaultEditor returns the default editor from the list
func (c *Config) GetDefaultEditor() *EditorConfig {
	// First check if there's an explicitly marked default
	for i := range c.Editors {
		if c.Editors[i].Default {
			return &c.Editors[i]
		}
	}

	// Then check if DefaultEditor is set
	if c.DefaultEditor != "" {
		for i := range c.Editors {
			if c.Editors[i].Name == c.DefaultEditor {
				return &c.Editors[i]
			}
		}
	}

	// Return first available editor
	for i := range c.Editors {
		if c.Editors[i].Available {
			return &c.Editors[i]
		}
	}

	// Return first editor if any exist
	if len(c.Editors) > 0 {
		return &c.Editors[0]
	}

	return nil
}

// GetEditor returns an editor by name
func (c *Config) GetEditor(name string) *EditorConfig {
	for i := range c.Editors {
		if c.Editors[i].Name == name {
			return &c.Editors[i]
		}
	}
	return nil
}

// GetDefaultEditor returns the default editor for client config
func (c *ClientConfig) GetDefaultEditor() *EditorConfig {
	// Check if DefaultEditor is set
	if c.DefaultEditor != "" {
		for i := range c.Editors {
			if c.Editors[i].Name == c.DefaultEditor {
				return &c.Editors[i]
			}
		}
	}

	// Return first editor if any exist
	if len(c.Editors) > 0 {
		return &c.Editors[0]
	}

	return nil
}
