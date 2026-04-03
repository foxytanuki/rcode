package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateToUnifiedConfig_MergesClientAndServerFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	clientPath := filepath.Join(dir, "config.yaml")
	serverPath := filepath.Join(dir, "server-config.yaml")

	clientData := []byte(`hosts:
  server:
    primary: 192.168.100.21
  ssh:
    host: 192.168.100.20
network:
  timeout: 2s
default_editor: code
logging:
  level: debug
`)
	serverData := []byte(`server:
  host: 0.0.0.0
  port: 3339
editors:
  - name: code
    command: /Applications/Visual Studio Code.app/Contents/Resources/app/bin/code --remote ssh-remote+{user}@{host} {path}
    default: true
    available: true
logging:
  level: info
`)

	if err := os.WriteFile(clientPath, clientData, 0o600); err != nil {
		t.Fatalf("WriteFile(client) error = %v", err)
	}
	if err := os.WriteFile(serverPath, serverData, 0o600); err != nil {
		t.Fatalf("WriteFile(server) error = %v", err)
	}

	result, err := MigrateToUnifiedConfig(clientPath, serverPath)
	if err != nil {
		t.Fatalf("MigrateToUnifiedConfig() error = %v", err)
	}

	if result.ClientBackupPath == "" || result.ServerBackupPath == "" {
		t.Fatalf("expected both backup paths, got %#v", result)
	}

	if _, err := os.Stat(serverPath); !os.IsNotExist(err) {
		t.Fatalf("server config should be removed after migration, stat err = %v", err)
	}

	data, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("ReadFile(unified) error = %v", err)
	}

	if !hasNestedClientConfig(data) || !hasNestedServerConfig(data) {
		t.Fatalf("expected unified config to contain client and server sections:\n%s", data)
	}

	clientCfg, err := LoadClientConfig(clientPath)
	if err != nil {
		t.Fatalf("LoadClientConfig() error = %v", err)
	}
	if clientCfg.DefaultEditor != "code" {
		t.Fatalf("DefaultEditor = %q, want %q", clientCfg.DefaultEditor, "code")
	}
	if clientCfg.Logging.Level != "debug" {
		t.Fatalf("client Logging.Level = %q, want %q", clientCfg.Logging.Level, "debug")
	}

	serverCfg, err := LoadServerConfig(clientPath)
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}
	if serverCfg.Logging.Level != "info" {
		t.Fatalf("server Logging.Level = %q, want %q", serverCfg.Logging.Level, "info")
	}
	if len(serverCfg.Editors) != 1 || serverCfg.Editors[0].Name != "code" {
		t.Fatalf("Editors = %#v, want single code editor", serverCfg.Editors)
	}
}

func TestMigrateToUnifiedConfig_ServerOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	clientPath := filepath.Join(dir, "config.yaml")
	serverPath := filepath.Join(dir, "server-config.yaml")

	serverData := []byte(`server:
  host: 0.0.0.0
  port: 3339
editors:
  - name: cursor
    command: cursor --remote ssh-remote+{user}@{host} {path}
    default: true
    available: true
logging:
  level: info
`)

	if err := os.WriteFile(serverPath, serverData, 0o600); err != nil {
		t.Fatalf("WriteFile(server) error = %v", err)
	}

	result, err := MigrateToUnifiedConfig(clientPath, serverPath)
	if err != nil {
		t.Fatalf("MigrateToUnifiedConfig() error = %v", err)
	}

	if result.ClientBackupPath != "" {
		t.Fatalf("expected no client backup for missing client config, got %q", result.ClientBackupPath)
	}

	if _, err := os.Stat(clientPath); err != nil {
		t.Fatalf("expected unified config to be created, stat err = %v", err)
	}

	serverCfg, err := LoadServerConfig(clientPath)
	if err != nil {
		t.Fatalf("LoadServerConfig() error = %v", err)
	}

	if serverCfg.Server.Port != 3339 {
		t.Fatalf("Port = %d, want %d", serverCfg.Server.Port, 3339)
	}
	if len(serverCfg.Editors) != 1 || serverCfg.Editors[0].Name != "cursor" {
		t.Fatalf("Editors = %#v, want single cursor editor", serverCfg.Editors)
	}
}
