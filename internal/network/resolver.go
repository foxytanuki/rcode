// Package network provides network-related utilities for rcode,
// including host resolution with fallback logic.
package network

// HostType represents the type of host being resolved.
type HostType int

const (
	// ServerHost is used for rcode CLI → rcode-server HTTP communication.
	ServerHost HostType = iota
	// SSHHost is used for editor → remote machine SSH connection.
	SSHHost
)

func (t HostType) String() string {
	switch t {
	case ServerHost:
		return "ServerHost"
	case SSHHost:
		return "SSHHost"
	default:
		return "Unknown"
	}
}

// ResolvedHosts contains the resolved hosts for both server and SSH connections.
type ResolvedHosts struct {
	// Server is the host to connect to the rcode server (e.g., "192.168.1.100:3339").
	Server string
	// ServerFallback is the fallback server host (e.g., Tailscale IP).
	ServerFallback string
	// SSH is the host used in editor SSH connection (e.g., "dev-server", "ws01tail").
	SSH string
	// Source indicates which HostSource provided the SSH host.
	Source string
}

// HostSource provides host values for resolution.
// Implementations can read from environment variables, config files,
// or perform detection (e.g., Tailscale).
type HostSource interface {
	// Name returns the source name for logging/debugging.
	Name() string
	// Priority returns the source priority (lower = higher priority).
	Priority() int
	// Resolve attempts to resolve a host of the given type.
	// Returns empty string if this source cannot provide the host.
	Resolve(hostType HostType) string
}

// Resolver resolves hosts using a chain of HostSources.
type Resolver struct {
	sources []HostSource
}

// NewResolver creates a new Resolver with the given sources.
// Sources are automatically sorted by priority.
func NewResolver(sources ...HostSource) *Resolver {
	// Sort sources by priority (lower = higher priority)
	sorted := make([]HostSource, len(sources))
	copy(sorted, sources)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority() < sorted[i].Priority() {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return &Resolver{sources: sorted}
}

// Resolve resolves all hosts using the configured sources.
func (r *Resolver) Resolve() ResolvedHosts {
	result := ResolvedHosts{}

	// Resolve ServerHost
	for _, src := range r.sources {
		if host := src.Resolve(ServerHost); host != "" {
			if result.Server == "" {
				result.Server = host
			} else if result.ServerFallback == "" && host != result.Server {
				result.ServerFallback = host
			}
		}
	}

	// Resolve SSHHost
	for _, src := range r.sources {
		if host := src.Resolve(SSHHost); host != "" {
			result.SSH = host
			result.Source = src.Name()
			break
		}
	}

	return result
}

// ResolveSSH resolves only the SSH host.
func (r *Resolver) ResolveSSH() (host, source string) {
	for _, src := range r.sources {
		if h := src.Resolve(SSHHost); h != "" {
			return h, src.Name()
		}
	}
	return "", ""
}

// ResolveServer resolves only the server host.
func (r *Resolver) ResolveServer() (primary, fallback string) {
	for _, src := range r.sources {
		if host := src.Resolve(ServerHost); host != "" {
			if primary == "" {
				primary = host
			} else if fallback == "" && host != primary {
				fallback = host
			}
		}
	}
	return primary, fallback
}
