package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
	"github.com/foxytanuki/rcode/internal/version"
	"github.com/foxytanuki/rcode/pkg/api"
)

func TestHandleHealth(t *testing.T) {
	server := createTestServer()

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET request",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST request",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", http.NoBody)
			rec := httptest.NewRecorder()

			server.handleHealth(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("handleHealth() status = %v, want %v", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp api.HealthResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				if resp.Status != "healthy" {
					t.Errorf("Health status = %v, want healthy", resp.Status)
				}

				if resp.Version != version.Version {
					t.Errorf("Version = %v, want %v", resp.Version, version.Version)
				}
			}
		})
	}
}

func TestHandleEditors(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest(http.MethodGet, "/editors", http.NoBody)
	rec := httptest.NewRecorder()

	server.handleEditors(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleEditors() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var resp api.EditorsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(resp.Editors) == 0 {
		t.Error("No editors returned")
	}

	// Check for default editor
	hasDefault := false
	for _, editor := range resp.Editors {
		if editor.Default {
			hasDefault = true
			if resp.DefaultEditor != editor.Name {
				t.Errorf("DefaultEditor mismatch: %v != %v", resp.DefaultEditor, editor.Name)
			}
		}
	}

	if !hasDefault && resp.DefaultEditor == "" {
		t.Error("No default editor configured")
	}
}

func TestHandleOpenEditor(t *testing.T) {
	server := createTestServer()

	tests := []struct {
		name       string
		method     string
		request    *api.OpenRequest
		wantStatus int
	}{
		{
			name:   "valid request",
			method: http.MethodPost,
			request: &api.OpenRequest{
				Path:   "/home/user/project",
				Editor: "test-editor",
				User:   "testuser",
				Host:   "testhost",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "missing path",
			method: http.MethodPost,
			request: &api.OpenRequest{
				Editor: "test-editor",
				User:   "testuser",
				Host:   "testhost",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "missing user",
			method: http.MethodPost,
			request: &api.OpenRequest{
				Path:   "/home/user/project",
				Editor: "test-editor",
				Host:   "testhost",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "GET request",
			method:     http.MethodGet,
			request:    nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.request != nil {
				var err error
				body, err = json.Marshal(tt.request)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/open-editor", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.handleOpenEditor(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("handleOpenEditor() status = %v, want %v", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandleOpenEditorResolvesSSHAlias(t *testing.T) {
	homeDir := t.TempDir()
	sshDir := homeDir + "/.ssh"
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configData := []byte("Host ws01\n  User foxy\n  HostName 192.168.100.20\n")
	if err := os.WriteFile(sshDir+"/config", configData, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("HOME", homeDir)

	server := createTestServer()
	server.editor.RemoveEditor("test-editor")
	if err := server.editor.AddEditor(config.EditorConfig{
		Name:    "test-editor",
		Command: "code --remote ssh-remote+{user}@{host} {path}",
		Default: true,
	}); err != nil {
		t.Fatalf("AddEditor() error = %v", err)
	}
	request := api.OpenRequest{
		Path:   "/home/user/project",
		Editor: "test-editor",
		User:   "foxy",
		Host:   "192.168.100.20",
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/open-editor", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.handleOpenEditor(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handleOpenEditor() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var resp api.OpenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !strings.Contains(resp.Command, "ssh-remote+ws01") {
		t.Fatalf("Command = %q, want alias host", resp.Command)
	}
	if strings.Contains(resp.Command, "ssh-remote+foxy@ws01") {
		t.Fatalf("Command = %q, want alias host", resp.Command)
	}
}

func TestRespondJSON(t *testing.T) {
	server := createTestServer()

	data := map[string]string{
		"test": "value",
		"foo":  "bar",
	}

	rec := httptest.NewRecorder()
	server.respondJSON(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("respondJSON() status = %v, want %v", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if result["test"] != "value" || result["foo"] != "bar" {
		t.Errorf("Unexpected response data: %v", result)
	}
}

func TestRespondError(t *testing.T) {
	server := createTestServer()

	rec := httptest.NewRecorder()
	server.respondError(rec, api.ErrInvalidPath, http.StatusBadRequest, "test details")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("respondError() status = %v, want %v", rec.Code, http.StatusBadRequest)
	}

	var resp api.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.Message != api.ErrInvalidPath.Error() {
		t.Errorf("Error message = %v, want %v", resp.Message, api.ErrInvalidPath.Error())
	}

	if resp.Details != "test details" {
		t.Errorf("Error details = %v, want 'test details'", resp.Details)
	}

	if resp.Code != api.CodeInvalidPath {
		t.Errorf("Error code = %v, want %v", resp.Code, api.CodeInvalidPath)
	}
}

// Helper function to create a test server
func createTestServer() *Server {
	cfg := &config.ServerConfigFile{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         3339,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Editors: []config.EditorConfig{
			{
				Name:      "test-editor",
				Command:   "echo 'Opening {path} for {user}@{host}'",
				Default:   true,
				Available: true,
			},
			{
				Name:      "another-editor",
				Command:   "echo 'Another {path}'",
				Default:   false,
				Available: false,
			},
		},
		Logging: config.LogConfig{
			Level:   "info",
			Console: false,
		},
	}

	log := logger.New(&logger.Config{
		Level:   "error", // Quiet logs for tests
		Console: false,
	})

	srv, err := NewServer(cfg, log)
	if err != nil {
		panic(fmt.Sprintf("createTestServer: %v", err))
	}
	return srv
}
