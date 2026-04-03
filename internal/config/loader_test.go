package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadClientConfig_LoadsUnifiedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := []byte(`server:
  host: 0.0.0.0
  port: 3339
editors:
  - name: code
    command: code --remote ssh-remote+{user}@{host} {path}
    default: true
    available: true
client:
  hosts:
    server:
      primary: 192.168.100.21
      fallback: 100.64.0.1
    ssh:
      host: 192.168.100.20
      auto_detect:
        tailscale: true
        tailscale_pattern: '{hostname-}tail'
  network:
    timeout: 3s
    retry_attempts: 2
    retry_delay: 250ms
  default_editor: code
  fallback_editors:
    code: code --remote ssh-remote+{user}@{host} {path}
logging:
  level: debug
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadClientConfig(path)
	if err != nil {
		t.Fatalf("LoadClientConfig() error = %v", err)
	}

	if cfg.Hosts.Server.Primary != "192.168.100.21" {
		t.Fatalf("Primary = %q, want %q", cfg.Hosts.Server.Primary, "192.168.100.21")
	}

	if cfg.DefaultEditor != "code" {
		t.Fatalf("DefaultEditor = %q, want %q", cfg.DefaultEditor, "code")
	}

	if cfg.Logging.Level != "debug" {
		t.Fatalf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}

	if cfg.Network.RetryAttempts != 2 {
		t.Fatalf("RetryAttempts = %d, want %d", cfg.Network.RetryAttempts, 2)
	}
}

func TestLoadServerConfig_LoadsUnifiedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := []byte(`server:
  host: 127.0.0.1
  port: 4444
  read_timeout: 5s
  write_timeout: 6s
  idle_timeout: 7s
editors:
  - name: code
    command: code --remote ssh-remote+{user}@{host} {path}
    default: true
    available: true
logging:
  level: warn
client:
  default_editor: code
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}

	if cfg.Server.Port != 4444 {
		t.Fatalf("Port = %d, want %d", cfg.Server.Port, 4444)
	}

	if len(cfg.Editors) != 1 || cfg.Editors[0].Name != "code" {
		t.Fatalf("Editors = %#v, want single code editor", cfg.Editors)
	}

	if cfg.Logging.Level != "warn" {
		t.Fatalf("Logging.Level = %q, want %q", cfg.Logging.Level, "warn")
	}
}

func TestLoadServerConfig_PrefersUnifiedDefaultConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "rcode")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	unifiedPath := filepath.Join(configDir, "config.yaml")
	legacyPath := filepath.Join(configDir, "server-config.yaml")

	unified := []byte(`server:
  host: 0.0.0.0
  port: 4444
editors:
  - name: code
    command: code --remote ssh-remote+{user}@{host} {path}
    default: true
    available: true
`)
	legacy := []byte(`server:
  host: 127.0.0.1
  port: 5555
editors:
  - name: cursor
    command: cursor --remote ssh-remote+{user}@{host} {path}
    default: true
    available: true
`)

	if err := os.WriteFile(unifiedPath, unified, 0o600); err != nil {
		t.Fatalf("WriteFile(unified) error = %v", err)
	}

	if err := os.WriteFile(legacyPath, legacy, 0o600); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}

	cfg, err := LoadServerConfig("")
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}

	if cfg.Server.Port != 4444 {
		t.Fatalf("Port = %d, want %d", cfg.Server.Port, 4444)
	}

	if len(cfg.Editors) != 1 || cfg.Editors[0].Name != "code" {
		t.Fatalf("Editors = %#v, want unified config editors", cfg.Editors)
	}
}
