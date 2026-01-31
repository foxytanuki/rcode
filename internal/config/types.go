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

// HostsConfig represents the new unified host configuration.
type HostsConfig struct {
	Server ServerHostConfig `yaml:"server" json:"server"` // Server connection settings
	SSH    SSHHostConfig    `yaml:"ssh" json:"ssh"`       // SSH connection settings
}

// ServerHostConfig represents server connection configuration.
type ServerHostConfig struct {
	Primary  string `yaml:"primary" json:"primary"`   // Primary server host (e.g., LAN IP)
	Fallback string `yaml:"fallback" json:"fallback"` // Fallback server host (e.g., Tailscale IP)
}

// SSHHostConfig represents SSH host configuration for editor connections.
type SSHHostConfig struct {
	Host       string           `yaml:"host,omitempty" json:"host,omitempty"` // Explicit SSH host (empty = auto-detect)
	AutoDetect AutoDetectConfig `yaml:"auto_detect" json:"auto_detect"`       // Auto-detection settings
}

// AutoDetectConfig represents auto-detection settings.
type AutoDetectConfig struct {
	Tailscale        bool   `yaml:"tailscale" json:"tailscale"`                                     // Enable Tailscale auto-detection
	TailscalePattern string `yaml:"tailscale_pattern,omitempty" json:"tailscale_pattern,omitempty"` // Pattern for Tailscale hostname
}

// FallbackEditorsConfig stores editor command templates for fallback use.
// Used when the server is unreachable.
type FallbackEditorsConfig map[string]string

// ClientConfig represents client-specific configuration
// Note: Editor definitions are centralized on the server. The client only stores
// the name of the default editor to use, not the command templates.
type ClientConfig struct {
	// New unified host configuration
	Hosts           HostsConfig           `yaml:"hosts,omitempty" json:"hosts,omitempty"`                       // Unified host configuration
	FallbackEditors FallbackEditorsConfig `yaml:"fallback_editors,omitempty" json:"fallback_editors,omitempty"` // Fallback editor commands

	// Legacy fields (for backward compatibility, migrated at load time)
	Network              NetworkConfig `yaml:"network" json:"network"`                                                   // Network configuration
	DefaultEditor        string        `yaml:"default_editor" json:"default_editor"`                                     // Default editor name (validated against server)
	Logging              LogConfig     `yaml:"logging" json:"logging"`                                                   // Logging configuration
	SSHHost              string        `yaml:"ssh_host,omitempty" json:"ssh_host,omitempty"`                             // Override SSH host for editor connection (e.g., LAN IP when using Tailscale SSH)
	AutoDetectTailscale  bool          `yaml:"auto_detect_tailscale,omitempty" json:"auto_detect_tailscale,omitempty"`   // Enable automatic Tailscale detection
	TailscaleHostPattern string        `yaml:"tailscale_host_pattern,omitempty" json:"tailscale_host_pattern,omitempty"` // Pattern for Tailscale hostname (e.g., "{hostname}tail")
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
	DefaultServerPort    = 3339
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

// GetDefaultEditorName returns the default editor name for client config
// Note: The client no longer stores editor command templates locally.
// Editor validation and command templates are fetched from the server.
func (c *ClientConfig) GetDefaultEditorName() string {
	return c.DefaultEditor
}
