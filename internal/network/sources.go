package network

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strings"
)

// Priority constants for host sources.
const (
	PriorityCommandLine = 10  // Highest priority - explicit user input
	PriorityEnvVar      = 20  // Environment variable overrides
	PriorityConfig      = 30  // Configuration file values
	PriorityTailscale   = 40  // Auto-detected Tailscale
	PrioritySSHEnv      = 50  // SSH_CONNECTION environment
	PriorityHostname    = 100 // Fallback to hostname
)

// CommandLineSource provides hosts from command-line flags.
type CommandLineSource struct {
	Host string // Value from --host flag
}

// Name returns the source name.
func (s *CommandLineSource) Name() string { return "command-line" }

// Priority returns the source priority.
func (s *CommandLineSource) Priority() int { return PriorityCommandLine }

// Resolve returns the host if set via command-line.
func (s *CommandLineSource) Resolve(_ HostType) string {
	if s.Host == "" {
		return ""
	}
	// Command-line host applies to both server and SSH when specified
	return s.Host
}

// EnvSource provides hosts from environment variables.
type EnvSource struct {
	// ServerHostEnv is the env var name for server host (e.g., "RCODE_SERVER_HOST").
	ServerHostEnv string
	// SSHHostEnv is the env var name for SSH host (e.g., "RCODE_SSH_HOST").
	SSHHostEnv string
	// LegacyHostEnv is the legacy env var name (e.g., "RCODE_HOST").
	LegacyHostEnv string
}

// Name returns the source name.
func (s *EnvSource) Name() string { return "environment" }

// Priority returns the source priority.
func (s *EnvSource) Priority() int { return PriorityEnvVar }

// Resolve returns the host from environment variables.
func (s *EnvSource) Resolve(hostType HostType) string {
	switch hostType {
	case ServerHost:
		if host := os.Getenv(s.ServerHostEnv); host != "" {
			return host
		}
		// Fall back to legacy env var for server host
		if host := os.Getenv(s.LegacyHostEnv); host != "" {
			return host
		}
	case SSHHost:
		if host := os.Getenv(s.SSHHostEnv); host != "" {
			return host
		}
	}
	return ""
}

// ConfigSource provides hosts from configuration.
type ConfigSource struct {
	// ServerPrimary is the primary server host from config.
	ServerPrimary string
	// ServerFallback is the fallback server host from config.
	ServerFallback string
	// SSHHost is the explicit SSH host from config.
	SSHHost string
}

// Name returns the source name.
func (s *ConfigSource) Name() string { return "config" }

// Priority returns the source priority.
func (s *ConfigSource) Priority() int { return PriorityConfig }

// Resolve returns the host from configuration.
func (s *ConfigSource) Resolve(hostType HostType) string {
	switch hostType {
	case ServerHost:
		// Note: fallback is handled separately in Resolver
		return s.ServerPrimary
	case SSHHost:
		return s.SSHHost
	}
	return ""
}

// ConfigFallbackSource provides the fallback server host from configuration.
type ConfigFallbackSource struct {
	ServerFallback string
}

// Name returns the source name.
func (s *ConfigFallbackSource) Name() string { return "config-fallback" }

// Priority returns the source priority (slightly lower than config).
func (s *ConfigFallbackSource) Priority() int { return PriorityConfig + 1 }

// Resolve returns the fallback server host.
func (s *ConfigFallbackSource) Resolve(hostType HostType) string {
	if hostType == ServerHost {
		return s.ServerFallback
	}
	return ""
}

// TailscaleSource provides hosts via Tailscale auto-detection.
type TailscaleSource struct {
	// Enabled indicates whether Tailscale detection is enabled.
	Enabled bool
	// HostPattern is the pattern for generating Tailscale hostname (e.g., "{hostname-}tail").
	HostPattern string
	// ClientIP is the SSH client IP for detecting Tailscale connection.
	ClientIP string
}

// Name returns the source name.
func (s *TailscaleSource) Name() string { return "tailscale" }

// Priority returns the source priority.
func (s *TailscaleSource) Priority() int { return PriorityTailscale }

// Resolve attempts to detect and return Tailscale host.
func (s *TailscaleSource) Resolve(hostType HostType) string {
	if !s.Enabled {
		return ""
	}

	// Check if Tailscale is available
	tailscaleIP := getTailscaleInterfaceIP()
	if tailscaleIP == "" {
		return ""
	}

	// Check if connection is via Tailscale
	isViaTS := isTailscaleIP(s.ClientIP) || (tailscaleIP != "" && os.Getenv("SSH_TTY") != "")
	if !isViaTS {
		return ""
	}

	switch hostType {
	case ServerHost:
		// Return the Tailscale IP for server connection
		return tailscaleIP
	case SSHHost:
		// Generate Tailscale hostname for SSH
		hostname := getTailscaleHostname()
		if hostname == "" {
			return ""
		}
		return applyTailscalePattern(hostname, s.HostPattern)
	}
	return ""
}

// SSHConnectionSource provides hosts from SSH_CONNECTION environment.
type SSHConnectionSource struct {
	// ClientIP is extracted from SSH_CONNECTION.
	ClientIP string
}

// Name returns the source name.
func (s *SSHConnectionSource) Name() string { return "ssh-connection" }

// Priority returns the source priority.
func (s *SSHConnectionSource) Priority() int { return PrioritySSHEnv }

// Resolve returns the SSH client IP.
func (s *SSHConnectionSource) Resolve(hostType HostType) string {
	if hostType == SSHHost && s.ClientIP != "" {
		return s.ClientIP
	}
	return ""
}

// HostnameSource provides the local hostname as fallback.
type HostnameSource struct{}

// Name returns the source name.
func (s *HostnameSource) Name() string { return "hostname" }

// Priority returns the source priority.
func (s *HostnameSource) Priority() int { return PriorityHostname }

// Resolve returns the local hostname.
func (s *HostnameSource) Resolve(hostType HostType) string {
	if hostType == SSHHost {
		if hostname, err := os.Hostname(); err == nil {
			return hostname
		}
		return "localhost"
	}
	return ""
}

// Helper functions for Tailscale detection

// isTailscaleIP checks if an IP is in the Tailscale range (100.64.0.0/10).
func isTailscaleIP(ipStr string) bool {
	if ipStr == "" {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	_, tailscaleNet, _ := net.ParseCIDR("100.64.0.0/10")
	return tailscaleNet != nil && tailscaleNet.Contains(ip)
}

// getTailscaleInterfaceIP returns the Tailscale interface IP if available.
func getTailscaleInterfaceIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Name == "tailscale0" || strings.HasPrefix(iface.Name, "utun") {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					if isTailscaleIP(ipnet.IP.String()) {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	return ""
}

// tailscaleStatus represents the minimal Tailscale status structure.
type tailscaleStatus struct {
	Self struct {
		HostName string `json:"HostName"`
		DNSName  string `json:"DNSName"`
	} `json:"Self"`
}

// getTailscaleHostname retrieves the hostname from Tailscale.
func getTailscaleHostname() string {
	cmd := exec.Command("tailscale", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	var status tailscaleStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return ""
	}

	if status.Self.HostName != "" {
		return status.Self.HostName
	}
	if status.Self.DNSName != "" {
		return status.Self.DNSName
	}
	return ""
}

// applyTailscalePattern applies the hostname pattern for Tailscale.
func applyTailscalePattern(hostname, pattern string) string {
	// Strip common Tailscale suffixes
	baseName := strings.TrimSuffix(hostname, ".tail75a81.ts.net.")
	baseName = strings.TrimSuffix(baseName, ".ts.net.")

	if pattern == "" {
		// Default pattern: ws-01 -> ws01tail
		return strings.ReplaceAll(baseName, "-", "") + "tail"
	}

	// Apply pattern
	result := strings.ReplaceAll(pattern, "{hostname}", baseName)
	result = strings.ReplaceAll(result, "{hostname-}", strings.ReplaceAll(baseName, "-", ""))
	return result
}

// ExtractSSHClientIP extracts the client IP from SSH environment variables.
func ExtractSSHClientIP() string {
	// Check SSH_CONNECTION first: client_ip client_port server_ip server_port
	if conn := os.Getenv("SSH_CONNECTION"); conn != "" {
		parts := strings.Fields(conn)
		if len(parts) >= 1 {
			return parts[0]
		}
	}
	// Check SSH_CLIENT: client_ip client_port server_port
	if client := os.Getenv("SSH_CLIENT"); client != "" {
		parts := strings.Fields(client)
		if len(parts) >= 1 {
			return parts[0]
		}
	}
	return ""
}
