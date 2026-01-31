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

// MigrateClientConfig migrates old config format to new format.
// Returns warnings for deprecated fields that were migrated.
func MigrateClientConfig(cfg *ClientConfig) []MigrationWarning {
	var warnings []MigrationWarning

	// Migrate network.primary_host → hosts.server.primary
	if cfg.Network.PrimaryHost != "" && cfg.Hosts.Server.Primary == "" {
		cfg.Hosts.Server.Primary = cfg.Network.PrimaryHost
	}

	// Migrate network.fallback_host → hosts.server.fallback
	if cfg.Network.FallbackHost != "" && cfg.Hosts.Server.Fallback == "" {
		cfg.Hosts.Server.Fallback = cfg.Network.FallbackHost
	}

	// Migrate ssh_host → hosts.ssh.host
	if cfg.SSHHost != "" && cfg.Hosts.SSH.Host == "" {
		cfg.Hosts.SSH.Host = cfg.SSHHost
	}

	// Migrate auto_detect_tailscale → hosts.ssh.auto_detect.tailscale
	if cfg.AutoDetectTailscale && !cfg.Hosts.SSH.AutoDetect.Tailscale {
		cfg.Hosts.SSH.AutoDetect.Tailscale = cfg.AutoDetectTailscale
	}

	// Migrate tailscale_host_pattern → hosts.ssh.auto_detect.tailscale_pattern
	if cfg.TailscaleHostPattern != "" && cfg.Hosts.SSH.AutoDetect.TailscalePattern == "" {
		cfg.Hosts.SSH.AutoDetect.TailscalePattern = cfg.TailscaleHostPattern
	}

	// Set default fallback editors if not configured
	if cfg.FallbackEditors == nil {
		cfg.FallbackEditors = GetDefaultFallbackEditors()
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
			// Apply legacy value
			cfg.Hosts.Server.Primary = legacyHost
			// Also update Network for backward compatibility
			cfg.Network.PrimaryHost = legacyHost
			warnings = append(warnings, MigrationWarning{
				Field:   "RCODE_HOST",
				Message: "RCODE_HOST is deprecated for client, use RCODE_SERVER_HOST instead",
			})
		}
	}

	// Apply new environment variables
	if serverHost := os.Getenv("RCODE_SERVER_HOST"); serverHost != "" {
		cfg.Hosts.Server.Primary = serverHost
		cfg.Network.PrimaryHost = serverHost
	}

	if sshHost := os.Getenv("RCODE_SSH_HOST"); sshHost != "" {
		cfg.Hosts.SSH.Host = sshHost
		cfg.SSHHost = sshHost
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
			// Apply legacy value
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

// SyncLegacyFields ensures legacy fields are kept in sync with new fields.
// This is called after migration to maintain backward compatibility.
func SyncLegacyFields(cfg *ClientConfig) {
	// Sync from new to legacy (in case new fields were set directly)
	if cfg.Hosts.Server.Primary != "" && cfg.Network.PrimaryHost == "" {
		cfg.Network.PrimaryHost = cfg.Hosts.Server.Primary
	}
	if cfg.Hosts.Server.Fallback != "" && cfg.Network.FallbackHost == "" {
		cfg.Network.FallbackHost = cfg.Hosts.Server.Fallback
	}
	if cfg.Hosts.SSH.Host != "" && cfg.SSHHost == "" {
		cfg.SSHHost = cfg.Hosts.SSH.Host
	}
	if cfg.Hosts.SSH.AutoDetect.Tailscale && !cfg.AutoDetectTailscale {
		cfg.AutoDetectTailscale = cfg.Hosts.SSH.AutoDetect.Tailscale
	}
	if cfg.Hosts.SSH.AutoDetect.TailscalePattern != "" && cfg.TailscaleHostPattern == "" {
		cfg.TailscaleHostPattern = cfg.Hosts.SSH.AutoDetect.TailscalePattern
	}

	// Sync from legacy to new (in case legacy fields were set directly)
	if cfg.Network.PrimaryHost != "" && cfg.Hosts.Server.Primary == "" {
		cfg.Hosts.Server.Primary = cfg.Network.PrimaryHost
	}
	if cfg.Network.FallbackHost != "" && cfg.Hosts.Server.Fallback == "" {
		cfg.Hosts.Server.Fallback = cfg.Network.FallbackHost
	}
	if cfg.SSHHost != "" && cfg.Hosts.SSH.Host == "" {
		cfg.Hosts.SSH.Host = cfg.SSHHost
	}
	if cfg.AutoDetectTailscale && !cfg.Hosts.SSH.AutoDetect.Tailscale {
		cfg.Hosts.SSH.AutoDetect.Tailscale = cfg.AutoDetectTailscale
	}
	if cfg.TailscaleHostPattern != "" && cfg.Hosts.SSH.AutoDetect.TailscalePattern == "" {
		cfg.Hosts.SSH.AutoDetect.TailscalePattern = cfg.TailscaleHostPattern
	}
}
