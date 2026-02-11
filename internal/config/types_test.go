package config

import (
	"testing"
	"time"
)

func TestClientConfig_GetDefaultEditorName(t *testing.T) {
	tests := []struct {
		name   string
		config ClientConfig
		want   string
	}{
		{
			name: "default editor set",
			config: ClientConfig{
				DefaultEditor: "nvim",
			},
			want: "nvim",
		},
		{
			name: "no default editor",
			config: ClientConfig{
				DefaultEditor: "",
			},
			want: "",
		},
		{
			name: "cursor default",
			config: ClientConfig{
				DefaultEditor: "cursor",
			},
			want: "cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetDefaultEditorName()
			if got != tt.want {
				t.Errorf("GetDefaultEditorName() = %s, want %s", got, tt.want)
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
