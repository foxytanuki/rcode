package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
	"github.com/foxytanuki/rcode/pkg/api"
)

func TestClient_OpenEditor(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-editor" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Parse request
		var req api.OpenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Send success response
		resp := api.OpenResponse{
			Success: true,
			Message: "Editor opened",
			Editor:  req.Editor,
			Command: "test command",
		}
		resp.SetTimestamp()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Extract host and port from test server URL
	serverHost := server.URL[7:] // Remove "http://"

	// Create client with test configuration
	cfg := &config.ClientConfig{
		Network: config.NetworkConfig{
			PrimaryHost:   serverHost,
			Timeout:       2 * time.Second,
			RetryAttempts: 1,
		},
		DefaultEditor: "test-editor",
		Logging: config.LogConfig{
			Level: "error",
		},
	}

	client := NewClient(cfg, createTestLogger())

	// Test opening editor
	sshInfo := SSHInfo{
		User: "testuser",
		Host: "testhost",
	}

	err := client.OpenEditor("/test/path", "test-editor", &sshInfo)
	if err != nil {
		t.Errorf("OpenEditor() error = %v, want nil", err)
	}
}

func TestClient_OpenEditor_WithFallback(t *testing.T) {
	// Create primary server that fails
	primaryFailed := false
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		primaryFailed = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primaryServer.Close()

	// Create fallback server that succeeds
	fallbackUsed := false
	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fallbackUsed = true
		resp := api.OpenResponse{
			Success: true,
			Message: "Editor opened via fallback",
			Editor:  "test-editor",
			Command: "test command",
		}
		resp.SetTimestamp()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer fallbackServer.Close()

	// Create client with fallback configuration
	cfg := &config.ClientConfig{
		Network: config.NetworkConfig{
			PrimaryHost:   primaryServer.URL[7:],
			FallbackHost:  fallbackServer.URL[7:],
			Timeout:       2 * time.Second,
			RetryAttempts: 1,
		},
		DefaultEditor: "test-editor",
		Logging: config.LogConfig{
			Level: "error",
		},
	}

	client := NewClient(cfg, createTestLogger())

	// Test opening editor
	sshInfo := SSHInfo{
		User: "testuser",
		Host: "testhost",
	}

	err := client.OpenEditor("/test/path", "", &sshInfo)
	if err != nil {
		t.Errorf("OpenEditor() error = %v, want nil", err)
	}

	// Verify fallback was used
	if !primaryFailed {
		t.Error("Primary server was not attempted")
	}
	if !fallbackUsed {
		t.Error("Fallback server was not used")
	}
}

func TestClient_ListEditors(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/editors" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Send editors response
		resp := api.EditorsResponse{
			Editors: []api.EditorInfo{
				{Name: "editor1", Command: "cmd1 {path}", Available: true, Default: true},
				{Name: "editor2", Command: "cmd2 {path}", Available: false, Default: false},
			},
			DefaultEditor: "editor1",
		}
		resp.SetTimestamp()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	cfg := &config.ClientConfig{
		Network: config.NetworkConfig{
			PrimaryHost: server.URL[7:],
			Timeout:     2 * time.Second,
		},
		Logging: config.LogConfig{
			Level: "error",
		},
	}

	client := NewClient(cfg, createTestLogger())

	// Test listing editors
	err := client.ListEditors()
	if err != nil {
		t.Errorf("ListEditors() error = %v, want nil", err)
	}
}

func TestClient_CheckHealth(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Send health response
		resp := api.HealthResponse{
			Status:  "healthy",
			Version: "1.0.0",
			Uptime:  3600,
		}
		resp.SetTimestamp()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	cfg := &config.ClientConfig{
		Network: config.NetworkConfig{
			PrimaryHost: server.URL[7:],
			Timeout:     2 * time.Second,
		},
		Logging: config.LogConfig{
			Level: "error",
		},
	}

	client := NewClient(cfg, createTestLogger())

	// Test health check
	err := client.CheckHealth()
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil", err)
	}
}

func TestClient_GetManualCommand(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		editor  string
		sshInfo SSHInfo
		editors []config.EditorConfig
		want    string
	}{
		{
			name:   "configured editor",
			path:   "/home/project",
			editor: "test-editor",
			sshInfo: SSHInfo{
				User: "alice",
				Host: "server.com",
			},
			editors: []config.EditorConfig{
				{Name: "test-editor", Command: "editor --remote {user}@{host} {path}"},
			},
			want: "editor --remote alice@server.com /home/project",
		},
		{
			name:   "default cursor command",
			path:   "/home/project",
			editor: "cursor",
			sshInfo: SSHInfo{
				User: "bob",
				Host: "example.com",
			},
			editors: []config.EditorConfig{},
			want:    "cursor --remote ssh-remote+bob@example.com /home/project",
		},
		{
			name:   "default vscode command",
			path:   "/home/project",
			editor: "vscode",
			sshInfo: SSHInfo{
				User: "charlie",
				Host: "dev.local",
			},
			editors: []config.EditorConfig{},
			want:    "code --remote ssh-remote+charlie@dev.local /home/project",
		},
		{
			name:   "default nvim command",
			path:   "/home/project",
			editor: "nvim",
			sshInfo: SSHInfo{
				User: "dave",
				Host: "remote.net",
			},
			editors: []config.EditorConfig{},
			want:    "nvim scp://dave@remote.net//home/project",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ClientConfig{
				Editors: tt.editors,
				Logging: config.LogConfig{
					Level: "error",
				},
			}

			client := NewClient(cfg, createTestLogger())
			got := client.GetManualCommand(tt.path, tt.editor, &tt.sshInfo)

			if got != tt.want {
				t.Errorf("GetManualCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	// Create test server that fails first attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < maxAttempts {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success on final attempt
		resp := api.OpenResponse{
			Success: true,
			Message: "Success after retries",
			Editor:  "test-editor",
		}
		resp.SetTimestamp()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client with retry configuration
	cfg := &config.ClientConfig{
		Network: config.NetworkConfig{
			PrimaryHost:   server.URL[7:],
			Timeout:       2 * time.Second,
			RetryAttempts: maxAttempts,
			RetryDelay:    10 * time.Millisecond,
		},
		DefaultEditor: "test-editor",
		Logging: config.LogConfig{
			Level: "error",
		},
	}

	client := NewClient(cfg, createTestLogger())

	// Test with retries
	sshInfo := SSHInfo{
		User: "testuser",
		Host: "testhost",
	}

	err := client.OpenEditor("/test/path", "", &sshInfo)
	if err != nil {
		t.Errorf("OpenEditor() with retries error = %v, want nil", err)
	}

	if attempts != maxAttempts {
		t.Errorf("Retry attempts = %d, want %d", attempts, maxAttempts)
	}
}

// Helper function
func createTestLogger() *logger.Logger {
	return logger.New(&logger.Config{
		Level:   "error",
		Console: false,
	})
}
