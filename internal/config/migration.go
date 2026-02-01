package config

import (
	"fmt"
	"os"
)

// MigrationWarning represents a deprecation warning during migration.
type MigrationWarning struct {
	Field   string
	Message string
}

// legacyClientConfig is used to parse old config format.
// It contains all legacy fields that may exist in old config files.
type legacyClientConfig struct {
	// Old network fields (v0.2.x)
	Network struct {
		PrimaryHost   string `yaml:"primary_host"`
		FallbackHost  string `yaml:"fallback_host"`
		Timeout       string `yaml:"timeout"`
		RetryAttempts int    `yaml:"retry_attempts"`
		RetryDelay    string `yaml:"retry_delay"`
	} `yaml:"network"`

	// Old SSH fields (v0.2.x)
	SSHHost              string `yaml:"ssh_host"`
	AutoDetectTailscale  bool   `yaml:"auto_detect_tailscale"`
	TailscaleHostPattern string `yaml:"tailscale_host_pattern"`
}

// MigrateClientConfig migrates old config format to new format.
// This function reads legacy fields that may have been parsed into a raw structure
// and applies them to the new ClientConfig.
func MigrateClientConfig(cfg *ClientConfig) []MigrationWarning {
	var warnings []MigrationWarning

	// Set default fallback editors if not configured
	if cfg.FallbackEditors == nil {
		cfg.FallbackEditors = GetDefaultFallbackEditors()
	}

	return warnings
}

// MigrateFromLegacy migrates a legacy config structure to the new format.
// This is called during LoadClientConfig before applying defaults.
func MigrateFromLegacy(legacy *legacyClientConfig, cfg *ClientConfig) []MigrationWarning {
	var warnings []MigrationWarning

	// Migrate network.primary_host → hosts.server.primary
	if legacy.Network.PrimaryHost != "" && cfg.Hosts.Server.Primary == "" {
		cfg.Hosts.Server.Primary = legacy.Network.PrimaryHost
		warnings = append(warnings, MigrationWarning{
			Field:   "network.primary_host",
			Message: "Migrated to hosts.server.primary",
		})
	}

	// Migrate network.fallback_host → hosts.server.fallback
	if legacy.Network.FallbackHost != "" && cfg.Hosts.Server.Fallback == "" {
		cfg.Hosts.Server.Fallback = legacy.Network.FallbackHost
		warnings = append(warnings, MigrationWarning{
			Field:   "network.fallback_host",
			Message: "Migrated to hosts.server.fallback",
		})
	}

	// Migrate ssh_host → hosts.ssh.host
	if legacy.SSHHost != "" && cfg.Hosts.SSH.Host == "" {
		cfg.Hosts.SSH.Host = legacy.SSHHost
		warnings = append(warnings, MigrationWarning{
			Field:   "ssh_host",
			Message: "Migrated to hosts.ssh.host",
		})
	}

	// Migrate auto_detect_tailscale → hosts.ssh.auto_detect.tailscale
	if legacy.AutoDetectTailscale && !cfg.Hosts.SSH.AutoDetect.Tailscale {
		cfg.Hosts.SSH.AutoDetect.Tailscale = legacy.AutoDetectTailscale
		warnings = append(warnings, MigrationWarning{
			Field:   "auto_detect_tailscale",
			Message: "Migrated to hosts.ssh.auto_detect.tailscale",
		})
	}

	// Migrate tailscale_host_pattern → hosts.ssh.auto_detect.tailscale_pattern
	if legacy.TailscaleHostPattern != "" && cfg.Hosts.SSH.AutoDetect.TailscalePattern == "" {
		cfg.Hosts.SSH.AutoDetect.TailscalePattern = legacy.TailscaleHostPattern
		warnings = append(warnings, MigrationWarning{
			Field:   "tailscale_host_pattern",
			Message: "Migrated to hosts.ssh.auto_detect.tailscale_pattern",
		})
	}

	return warnings
}

// MigrateClientEnvironment checks for deprecated environment variables
// and returns warnings. It also applies the new env vars to config.
func MigrateClientEnvironment(cfg *ClientConfig) []MigrationWarning {
	var warnings []MigrationWarning

	// Check for legacy RCODE_HOST (client-side meaning: server connection)
	if legacyHost := os.Getenv("RCODE_HOST"); legacyHost != "" {
		// Check if new env var is also set
		if os.Getenv("RCODE_SERVER_HOST") == "" {
			cfg.Hosts.Server.Primary = legacyHost
			warnings = append(warnings, MigrationWarning{
				Field:   "RCODE_HOST",
				Message: "RCODE_HOST is deprecated for client, use RCODE_SERVER_HOST instead",
			})
		}
	}

	// Apply new environment variables
	if serverHost := os.Getenv("RCODE_SERVER_HOST"); serverHost != "" {
		cfg.Hosts.Server.Primary = serverHost
	}

	if sshHost := os.Getenv("RCODE_SSH_HOST"); sshHost != "" {
		cfg.Hosts.SSH.Host = sshHost
	}

	return warnings
}

// MigrateServerEnvironment checks for deprecated environment variables
// on the server side and returns warnings.
func MigrateServerEnvironment(cfg *ServerConfigFile) []MigrationWarning {
	var warnings []MigrationWarning

	// Check for legacy RCODE_HOST (server-side meaning: bind address)
	if legacyHost := os.Getenv("RCODE_HOST"); legacyHost != "" {
		// Check if new env var is also set
		if os.Getenv("RCODE_SERVER_BIND") == "" {
			cfg.Server.Host = legacyHost
			warnings = append(warnings, MigrationWarning{
				Field:   "RCODE_HOST",
				Message: "RCODE_HOST is deprecated for server, use RCODE_SERVER_BIND instead",
			})
		}
	}

	// Apply new environment variable
	if bindHost := os.Getenv("RCODE_SERVER_BIND"); bindHost != "" {
		cfg.Server.Host = bindHost
	}

	return warnings
}

// GetDefaultFallbackEditors returns default fallback editor commands.
func GetDefaultFallbackEditors() FallbackEditorsConfig {
	return FallbackEditorsConfig{
		"cursor": "cursor --remote ssh-remote+{user}@{host} {path}",
		"vscode": "code --remote ssh-remote+{user}@{host} {path}",
		"code":   "code --remote ssh-remote+{user}@{host} {path}",
		"zed":    "zed ssh://{user}@{host}/{path}",
		"nvim":   "nvim scp://{user}@{host}/{path}",
		"neovim": "nvim scp://{user}@{host}/{path}",
	}
}

// PrintMigrationWarnings prints migration warnings to stderr.
func PrintMigrationWarnings(warnings []MigrationWarning) {
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s - %s\n", w.Field, w.Message)
	}
}
