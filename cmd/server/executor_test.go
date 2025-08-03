package main

import (
	"strings"
	"testing"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
)

func TestBuildCommand(t *testing.T) {
	executor := createTestExecutor()
	
	tests := []struct {
		name     string
		template string
		user     string
		host     string
		path     string
		want     string
	}{
		{
			name:     "all placeholders",
			template: "ssh {user}@{host} 'editor {path}'",
			user:     "testuser",
			host:     "testhost",
			path:     "/home/project",
			want:     "ssh testuser@testhost 'editor /home/project'",
		},
		{
			name:     "path only",
			template: "editor {path}",
			user:     "testuser",
			host:     "testhost",
			path:     "/home/project",
			want:     "editor /home/project",
		},
		{
			name:     "multiple occurrences",
			template: "{user}@{host}:{path} -> {user}",
			user:     "alice",
			host:     "server",
			path:     "/tmp",
			want:     "alice@server:/tmp -> alice",
		},
		{
			name:     "no placeholders",
			template: "static command",
			user:     "testuser",
			host:     "testhost",
			path:     "/home/project",
			want:     "static command",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.buildCommand(tt.template, tt.user, tt.host, tt.path)
			if got != tt.want {
				t.Errorf("buildCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultEditor(t *testing.T) {
	tests := []struct {
		name    string
		editors []config.EditorConfig
		want    string
	}{
		{
			name: "explicitly marked default",
			editors: []config.EditorConfig{
				{Name: "editor1", Command: "cmd1 {path}", Default: false},
				{Name: "editor2", Command: "cmd2 {path}", Default: true},
				{Name: "editor3", Command: "cmd3 {path}", Default: false},
			},
			want: "editor2",
		},
		{
			name: "first available when no default",
			editors: []config.EditorConfig{
				{Name: "editor1", Command: "unavailable {path}", Available: false},
				{Name: "editor2", Command: "echo {path}", Available: true},
				{Name: "editor3", Command: "cmd3 {path}", Available: false},
			},
			want: "editor2",
		},
		{
			name: "first editor when none available",
			editors: []config.EditorConfig{
				{Name: "editor1", Command: "cmd1 {path}", Available: false},
				{Name: "editor2", Command: "cmd2 {path}", Available: false},
			},
			want: "editor1",
		},
		{
			name:    "no editors",
			editors: []config.EditorConfig{},
			want:    "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &Executor{
				editors:      tt.editors,
				log:          createTestLogger(),
				availability: make(map[string]bool),
			}
			
			// Set availability based on Available field
			for _, editor := range tt.editors {
				executor.availability[editor.Name] = editor.Available
			}
			
			editor := executor.getDefaultEditor()
			if tt.want == "" {
				if editor != nil {
					t.Errorf("getDefaultEditor() = %v, want nil", editor.Name)
				}
			} else {
				if editor == nil {
					t.Errorf("getDefaultEditor() = nil, want %v", tt.want)
				} else if editor.Name != tt.want {
					t.Errorf("getDefaultEditor() = %v, want %v", editor.Name, tt.want)
				}
			}
		})
	}
}

func TestValidateEditorConfig(t *testing.T) {
	executor := createTestExecutor()
	
	tests := []struct {
		name    string
		editor  config.EditorConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			editor: config.EditorConfig{
				Name:    "test",
				Command: "editor {path}",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			editor: config.EditorConfig{
				Name:    "",
				Command: "editor {path}",
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "missing command",
			editor: config.EditorConfig{
				Name:    "test",
				Command: "",
			},
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name: "missing path placeholder",
			editor: config.EditorConfig{
				Name:    "test",
				Command: "editor file.txt",
			},
			wantErr: true,
			errMsg:  "must contain {path}",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateEditorConfig(tt.editor)
			if tt.wantErr {
				if err == nil {
					t.Error("ValidateEditorConfig() error = nil, want error")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEditorConfig() error = %v, want containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEditorConfig() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestIsEditorAvailable(t *testing.T) {
	executor := createTestExecutor()
	
	// Mock availability
	executor.availability["available"] = true
	executor.availability["unavailable"] = false
	
	tests := []struct {
		name string
		editor string
		want   bool
	}{
		{"available editor", "available", true},
		{"unavailable editor", "unavailable", false},
		{"unknown editor", "unknown", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.IsEditorAvailable(tt.editor)
			if got != tt.want {
				t.Errorf("IsEditorAvailable(%v) = %v, want %v", tt.editor, got, tt.want)
			}
		})
	}
}

func TestOpenEditor(t *testing.T) {
	executor := createTestExecutor()
	
	// Add a test editor that uses echo command (should be available on most systems)
	executor.editors = []config.EditorConfig{
		{
			Name:      "echo-editor",
			Command:   "echo Opening {path} for {user}@{host}",
			Default:   true,
			Available: true,
		},
		{
			Name:      "unknown-editor",
			Command:   "nonexistent-command {path}",
			Default:   false,
			Available: false,
		},
	}
	executor.availability["echo-editor"] = true
	executor.availability["unknown-editor"] = false
	
	tests := []struct {
		name       string
		editorName string
		user       string
		host       string
		path       string
		wantErr    bool
	}{
		{
			name:       "valid editor",
			editorName: "echo-editor",
			user:       "testuser",
			host:       "testhost",
			path:       "/test/path",
			wantErr:    false,
		},
		{
			name:       "default editor (empty name)",
			editorName: "",
			user:       "testuser",
			host:       "testhost",
			path:       "/test/path",
			wantErr:    false,
		},
		{
			name:       "non-existent editor",
			editorName: "nonexistent",
			user:       "testuser",
			host:       "testhost",
			path:       "/test/path",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, err := executor.OpenEditor(tt.editorName, tt.user, tt.host, tt.path)
			
			if tt.wantErr {
				if err == nil {
					t.Error("OpenEditor() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("OpenEditor() error = %v, want nil", err)
				}
				if command == "" {
					t.Error("OpenEditor() returned empty command")
				}
			}
		})
	}
}

// Helper functions for testing
func createTestExecutor() *Executor {
	editors := []config.EditorConfig{
		{
			Name:      "test-editor",
			Command:   "test-cmd {path}",
			Default:   true,
			Available: true,
		},
	}
	
	return NewExecutor(editors, createTestLogger())
}

func createTestLogger() *logger.Logger {
	return logger.New(&logger.Config{
		Level:   "error", // Quiet for tests
		Console: false,
	})
}