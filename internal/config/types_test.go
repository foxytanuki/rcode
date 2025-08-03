package config

import (
	"testing"
	"time"
)

func TestConfig_GetDefaultEditor(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name: "explicitly marked default",
			config: Config{
				Editors: []EditorConfig{
					{Name: "vscode", Command: "code", Available: true, Default: false},
					{Name: "cursor", Command: "cursor", Available: true, Default: true},
					{Name: "nvim", Command: "nvim", Available: true, Default: false},
				},
			},
			want: "cursor",
		},
		{
			name: "default editor by name",
			config: Config{
				DefaultEditor: "nvim",
				Editors: []EditorConfig{
					{Name: "vscode", Command: "code", Available: true},
					{Name: "cursor", Command: "cursor", Available: true},
					{Name: "nvim", Command: "nvim", Available: true},
				},
			},
			want: "nvim",
		},
		{
			name: "first available when no default",
			config: Config{
				Editors: []EditorConfig{
					{Name: "vscode", Command: "code", Available: false},
					{Name: "cursor", Command: "cursor", Available: true},
					{Name: "nvim", Command: "nvim", Available: true},
				},
			},
			want: "cursor",
		},
		{
			name: "first editor when none available",
			config: Config{
				Editors: []EditorConfig{
					{Name: "vscode", Command: "code", Available: false},
					{Name: "cursor", Command: "cursor", Available: false},
				},
			},
			want: "vscode",
		},
		{
			name: "no editors",
			config: Config{
				Editors: []EditorConfig{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := tt.config.GetDefaultEditor()
			if tt.want == "" {
				if editor != nil {
					t.Errorf("GetDefaultEditor() = %v, want nil", editor.Name)
				}
			} else {
				if editor == nil {
					t.Errorf("GetDefaultEditor() = nil, want %s", tt.want)
				} else if editor.Name != tt.want {
					t.Errorf("GetDefaultEditor() = %s, want %s", editor.Name, tt.want)
				}
			}
		})
	}
}

func TestConfig_GetEditor(t *testing.T) {
	config := Config{
		Editors: []EditorConfig{
			{Name: "vscode", Command: "code"},
			{Name: "cursor", Command: "cursor"},
			{Name: "nvim", Command: "nvim"},
		},
	}

	tests := []struct {
		name      string
		editorName string
		wantFound bool
	}{
		{"existing editor", "cursor", true},
		{"non-existing editor", "sublime", false},
		{"case sensitive", "VSCode", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := config.GetEditor(tt.editorName)
			if tt.wantFound {
				if editor == nil {
					t.Errorf("GetEditor(%s) = nil, want editor", tt.editorName)
				} else if editor.Name != tt.editorName {
					t.Errorf("GetEditor(%s) returned wrong editor: %s", tt.editorName, editor.Name)
				}
			} else {
				if editor != nil {
					t.Errorf("GetEditor(%s) = %v, want nil", tt.editorName, editor)
				}
			}
		})
	}
}

func TestClientConfig_GetDefaultEditor(t *testing.T) {
	tests := []struct {
		name   string
		config ClientConfig
		want   string
	}{
		{
			name: "default editor by name",
			config: ClientConfig{
				DefaultEditor: "nvim",
				Editors: []EditorConfig{
					{Name: "vscode", Command: "code"},
					{Name: "nvim", Command: "nvim"},
				},
			},
			want: "nvim",
		},
		{
			name: "first editor when no default",
			config: ClientConfig{
				Editors: []EditorConfig{
					{Name: "vscode", Command: "code"},
					{Name: "cursor", Command: "cursor"},
				},
			},
			want: "vscode",
		},
		{
			name: "no editors",
			config: ClientConfig{
				Editors: []EditorConfig{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := tt.config.GetDefaultEditor()
			if tt.want == "" {
				if editor != nil {
					t.Errorf("GetDefaultEditor() = %v, want nil", editor.Name)
				}
			} else {
				if editor == nil {
					t.Errorf("GetDefaultEditor() = nil, want %s", tt.want)
				} else if editor.Name != tt.want {
					t.Errorf("GetDefaultEditor() = %s, want %s", editor.Name, tt.want)
				}
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	// Test that default constants have sensible values
	if DefaultServerPort < 1 || DefaultServerPort > 65535 {
		t.Errorf("DefaultServerPort = %d, want valid port number", DefaultServerPort)
	}
	
	if DefaultTimeout < time.Second {
		t.Errorf("DefaultTimeout = %v, want at least 1 second", DefaultTimeout)
	}
	
	if DefaultRetryAttempts < 1 {
		t.Errorf("DefaultRetryAttempts = %d, want at least 1", DefaultRetryAttempts)
	}
	
	if DefaultLogMaxSize < 1 {
		t.Errorf("DefaultLogMaxSize = %d, want at least 1 MB", DefaultLogMaxSize)
	}
	
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[DefaultLogLevel] {
		t.Errorf("DefaultLogLevel = %s, want valid log level", DefaultLogLevel)
	}
}