package network

import (
	"testing"
)

func TestHostTypeString(t *testing.T) {
	tests := []struct {
		hostType HostType
		want     string
	}{
		{ServerHost, "ServerHost"},
		{SSHHost, "SSHHost"},
		{HostType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.hostType.String()
		if got != tt.want {
			t.Errorf("HostType(%d).String() = %q, want %q", tt.hostType, got, tt.want)
		}
	}
}

func TestNewResolver(t *testing.T) {
	// Create sources with different priorities
	src1 := &ConfigSource{ServerPrimary: "config-host"}
	src2 := &CommandLineSource{Host: "cli-host"}
	src3 := &HostnameSource{}

	// Create resolver with sources in wrong order
	resolver := NewResolver(src1, src3, src2)

	// Verify sources are sorted by priority
	if len(resolver.sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(resolver.sources))
	}

	// CommandLine should be first (priority 10)
	if resolver.sources[0].Name() != "command-line" {
		t.Errorf("expected command-line first, got %s", resolver.sources[0].Name())
	}
	// Config should be second (priority 30)
	if resolver.sources[1].Name() != "config" {
		t.Errorf("expected config second, got %s", resolver.sources[1].Name())
	}
	// Hostname should be last (priority 100)
	if resolver.sources[2].Name() != "hostname" {
		t.Errorf("expected hostname last, got %s", resolver.sources[2].Name())
	}
}

func TestResolver_Resolve(t *testing.T) {
	tests := []struct {
		name       string
		sources    []HostSource
		wantServer string
		wantSSH    string
		wantSource string
	}{
		{
			name: "command line takes priority",
			sources: []HostSource{
				&CommandLineSource{Host: "cli-host"},
				&ConfigSource{ServerPrimary: "config-server", SSHHost: "config-ssh"},
			},
			wantServer: "cli-host",
			wantSSH:    "cli-host",
			wantSource: "command-line",
		},
		{
			name: "config when no command line",
			sources: []HostSource{
				&CommandLineSource{Host: ""},
				&ConfigSource{ServerPrimary: "config-server", SSHHost: "config-ssh"},
			},
			wantServer: "config-server",
			wantSSH:    "config-ssh",
			wantSource: "config",
		},
		{
			name: "hostname fallback",
			sources: []HostSource{
				&CommandLineSource{Host: ""},
				&ConfigSource{ServerPrimary: "", SSHHost: ""},
				&HostnameSource{},
			},
			wantServer: "",
			wantSSH:    "", // Will be actual hostname, checked separately
			wantSource: "hostname",
		},
		{
			name: "server fallback from config",
			sources: []HostSource{
				&ConfigSource{ServerPrimary: "primary"},
				&ConfigFallbackSource{ServerFallback: "fallback"},
			},
			wantServer: "primary",
			wantSSH:    "",
			wantSource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewResolver(tt.sources...)
			result := resolver.Resolve()

			if tt.wantServer != "" && result.Server != tt.wantServer {
				t.Errorf("Server = %q, want %q", result.Server, tt.wantServer)
			}
			if tt.wantSSH != "" && result.SSH != tt.wantSSH {
				t.Errorf("SSH = %q, want %q", result.SSH, tt.wantSSH)
			}
			if tt.wantSource != "" && result.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", result.Source, tt.wantSource)
			}
		})
	}
}

func TestResolver_ResolveSSH(t *testing.T) {
	resolver := NewResolver(
		&CommandLineSource{Host: ""},
		&ConfigSource{SSHHost: "config-ssh"},
		&SSHConnectionSource{ClientIP: "192.168.1.50"},
	)

	host, source := resolver.ResolveSSH()
	if host != "config-ssh" {
		t.Errorf("ResolveSSH() host = %q, want %q", host, "config-ssh")
	}
	if source != "config" {
		t.Errorf("ResolveSSH() source = %q, want %q", source, "config")
	}
}

func TestResolver_ResolveServer(t *testing.T) {
	resolver := NewResolver(
		&ConfigSource{ServerPrimary: "primary-host"},
		&ConfigFallbackSource{ServerFallback: "fallback-host"},
	)

	primary, fallback := resolver.ResolveServer()
	if primary != "primary-host" {
		t.Errorf("ResolveServer() primary = %q, want %q", primary, "primary-host")
	}
	if fallback != "fallback-host" {
		t.Errorf("ResolveServer() fallback = %q, want %q", fallback, "fallback-host")
	}
}

func TestCommandLineSource(t *testing.T) {
	src := &CommandLineSource{Host: "test-host"}

	if src.Name() != "command-line" {
		t.Errorf("Name() = %q, want %q", src.Name(), "command-line")
	}
	if src.Priority() != PriorityCommandLine {
		t.Errorf("Priority() = %d, want %d", src.Priority(), PriorityCommandLine)
	}

	// Should resolve for both host types
	if got := src.Resolve(ServerHost); got != "test-host" {
		t.Errorf("Resolve(ServerHost) = %q, want %q", got, "test-host")
	}
	if got := src.Resolve(SSHHost); got != "test-host" {
		t.Errorf("Resolve(SSHHost) = %q, want %q", got, "test-host")
	}

	// Empty host should return empty
	src.Host = ""
	if got := src.Resolve(ServerHost); got != "" {
		t.Errorf("Resolve(ServerHost) with empty host = %q, want empty", got)
	}
}

func TestEnvSource(t *testing.T) {
	src := &EnvSource{
		ServerHostEnv: "TEST_SERVER_HOST",
		SSHHostEnv:    "TEST_SSH_HOST",
		LegacyHostEnv: "TEST_LEGACY_HOST",
	}

	if src.Name() != "environment" {
		t.Errorf("Name() = %q, want %q", src.Name(), "environment")
	}
	if src.Priority() != PriorityEnvVar {
		t.Errorf("Priority() = %d, want %d", src.Priority(), PriorityEnvVar)
	}

	// Test with env vars set
	t.Setenv("TEST_SERVER_HOST", "server-from-env")
	t.Setenv("TEST_SSH_HOST", "ssh-from-env")

	if got := src.Resolve(ServerHost); got != "server-from-env" {
		t.Errorf("Resolve(ServerHost) = %q, want %q", got, "server-from-env")
	}
	if got := src.Resolve(SSHHost); got != "ssh-from-env" {
		t.Errorf("Resolve(SSHHost) = %q, want %q", got, "ssh-from-env")
	}

	// Test legacy fallback
	t.Setenv("TEST_SERVER_HOST", "")
	t.Setenv("TEST_LEGACY_HOST", "legacy-host")

	if got := src.Resolve(ServerHost); got != "legacy-host" {
		t.Errorf("Resolve(ServerHost) with legacy = %q, want %q", got, "legacy-host")
	}
}

func TestConfigSource(t *testing.T) {
	src := &ConfigSource{
		ServerPrimary:  "config-server",
		ServerFallback: "config-fallback",
		SSHHost:        "config-ssh",
	}

	if src.Name() != "config" {
		t.Errorf("Name() = %q, want %q", src.Name(), "config")
	}
	if src.Priority() != PriorityConfig {
		t.Errorf("Priority() = %d, want %d", src.Priority(), PriorityConfig)
	}

	if got := src.Resolve(ServerHost); got != "config-server" {
		t.Errorf("Resolve(ServerHost) = %q, want %q", got, "config-server")
	}
	if got := src.Resolve(SSHHost); got != "config-ssh" {
		t.Errorf("Resolve(SSHHost) = %q, want %q", got, "config-ssh")
	}
}

func TestSSHConnectionSource(t *testing.T) {
	src := &SSHConnectionSource{ClientIP: "192.168.1.100"}

	if src.Name() != "ssh-connection" {
		t.Errorf("Name() = %q, want %q", src.Name(), "ssh-connection")
	}
	if src.Priority() != PrioritySSHEnv {
		t.Errorf("Priority() = %d, want %d", src.Priority(), PrioritySSHEnv)
	}

	// Should only resolve SSHHost
	if got := src.Resolve(ServerHost); got != "" {
		t.Errorf("Resolve(ServerHost) = %q, want empty", got)
	}
	if got := src.Resolve(SSHHost); got != "192.168.1.100" {
		t.Errorf("Resolve(SSHHost) = %q, want %q", got, "192.168.1.100")
	}
}

func TestHostnameSource(t *testing.T) {
	src := &HostnameSource{}

	if src.Name() != "hostname" {
		t.Errorf("Name() = %q, want %q", src.Name(), "hostname")
	}
	if src.Priority() != PriorityHostname {
		t.Errorf("Priority() = %d, want %d", src.Priority(), PriorityHostname)
	}

	// Should only resolve SSHHost
	if got := src.Resolve(ServerHost); got != "" {
		t.Errorf("Resolve(ServerHost) = %q, want empty", got)
	}
	// SSHHost should return something (hostname or localhost)
	if got := src.Resolve(SSHHost); got == "" {
		t.Error("Resolve(SSHHost) returned empty, expected hostname or localhost")
	}
}

func TestIsTailscaleIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"100.63.255.255", false}, // Just below range
		{"100.128.0.0", false},    // Just above range
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		got := isTailscaleIP(tt.ip)
		if got != tt.want {
			t.Errorf("isTailscaleIP(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestApplyTailscalePattern(t *testing.T) {
	tests := []struct {
		hostname string
		pattern  string
		want     string
	}{
		{"ws-01", "", "ws01tail"},                          // Default pattern
		{"my-server", "", "myservertail"},                  // Default pattern with hyphen
		{"ws-01", "{hostname}tail", "ws-01tail"},           // Keep hyphens
		{"ws-01", "{hostname-}tail", "ws01tail"},           // Remove hyphens
		{"ws-01", "remote-{hostname}", "remote-ws-01"},     // Prefix
		{"hostname.ts.net.", "", "hostnametail"},           // Partial suffix stripped
		{"hostname.tail75a81.ts.net.", "", "hostnametail"}, // Full suffix stripped
	}

	for _, tt := range tests {
		got := applyTailscalePattern(tt.hostname, tt.pattern)
		if got != tt.want {
			t.Errorf("applyTailscalePattern(%q, %q) = %q, want %q",
				tt.hostname, tt.pattern, got, tt.want)
		}
	}
}

func TestTailscaleSource(t *testing.T) {
	src := &TailscaleSource{
		Enabled:     false,
		HostPattern: "{hostname-}tail",
		ClientIP:    "100.64.0.1",
	}

	if src.Name() != "tailscale" {
		t.Errorf("Name() = %q, want %q", src.Name(), "tailscale")
	}
	if src.Priority() != PriorityTailscale {
		t.Errorf("Priority() = %d, want %d", src.Priority(), PriorityTailscale)
	}

	// When disabled, should return empty
	if got := src.Resolve(ServerHost); got != "" {
		t.Errorf("Resolve(ServerHost) when disabled = %q, want empty", got)
	}
	if got := src.Resolve(SSHHost); got != "" {
		t.Errorf("Resolve(SSHHost) when disabled = %q, want empty", got)
	}
}

func TestExtractSSHClientIP(t *testing.T) {
	// Test with SSH_CONNECTION
	t.Setenv("SSH_CONNECTION", "192.168.1.50 12345 192.168.1.100 22")
	t.Setenv("SSH_CLIENT", "")

	if got := ExtractSSHClientIP(); got != "192.168.1.50" {
		t.Errorf("ExtractSSHClientIP() with SSH_CONNECTION = %q, want %q", got, "192.168.1.50")
	}

	// Test with SSH_CLIENT fallback
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "10.0.0.5 54321 22")

	if got := ExtractSSHClientIP(); got != "10.0.0.5" {
		t.Errorf("ExtractSSHClientIP() with SSH_CLIENT = %q, want %q", got, "10.0.0.5")
	}

	// Test with no env vars
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	if got := ExtractSSHClientIP(); got != "" {
		t.Errorf("ExtractSSHClientIP() with no env = %q, want empty", got)
	}
}
