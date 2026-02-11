package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfigFile
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ServerConfigFile{
				Server: ServerConfig{
					Host:         "0.0.0.0",
					Port:         3339,
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor {path}", Default: true},
				},
				Logging: LogConfig{
					Level:      "info",
					File:       "/tmp/test.log",
					MaxSize:    10,
					MaxBackups: 5,
					MaxAge:     30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port: 70000,
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor {path}"},
				},
				Logging: LogConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name: "invalid IP in whitelist",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port:       3339,
					AllowedIPs: []string{"192.168.1.1", "invalid-ip"},
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor {path}"},
				},
				Logging: LogConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "invalid IP or CIDR",
		},
		{
			name: "no editors",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port: 3339,
				},
				Editors: []EditorConfig{},
				Logging: LogConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "at least one editor must be configured",
		},
		{
			name: "multiple default editors",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port: 3339,
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor {path}", Default: true},
					{Name: "vscode", Command: "code {path}", Default: true},
				},
				Logging: LogConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "only one editor can be marked as default",
		},
		{
			name: "duplicate editor names",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port: 3339,
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor {path}"},
					{Name: "cursor", Command: "cursor2 {path}"},
				},
				Logging: LogConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "duplicate editor name",
		},
		{
			name: "missing required placeholder",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port: 3339,
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor --remote"},
				},
				Logging: LogConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "missing required placeholder: {path}",
		},
		{
			name: "invalid log level",
			config: ServerConfigFile{
				Server: ServerConfig{
					Port: 3339,
				},
				Editors: []EditorConfig{
					{Name: "cursor", Command: "cursor {path}"},
				},
				Logging: LogConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerConfig(&tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateServerConfig() error = nil, want error containing %q", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateServerConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateServerConfig() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateClientConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ClientConfig{
				Hosts: HostsConfig{
					Server: ServerHostConfig{
						Primary:  "192.168.1.100",
						Fallback: "100.64.0.1",
					},
				},
				Network: ClientNetworkConfig{
					Timeout:       2 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    500 * time.Millisecond,
				},
				DefaultEditor: "cursor",
				Logging: LogConfig{
					Level: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with any default editor name",
			config: ClientConfig{
				Hosts: HostsConfig{
					Server: ServerHostConfig{
						Primary: "192.168.1.100",
					},
				},
				DefaultEditor: "sublime", // Any editor name is valid - validation happens on server
				Logging: LogConfig{
					Level: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "missing primary host",
			config: ClientConfig{
				Hosts: HostsConfig{
					Server: ServerHostConfig{
						Primary: "",
					},
				},
				Logging: LogConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "primary server host cannot be empty",
		},
		{
			name: "negative timeout",
			config: ClientConfig{
				Hosts: HostsConfig{
					Server: ServerHostConfig{
						Primary: "192.168.1.100",
					},
				},
				Network: ClientNetworkConfig{
					Timeout: -1 * time.Second,
				},
				Logging: LogConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "timeout cannot be negative",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientConfig(&tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateClientConfig() error = nil, want error containing %q", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateClientConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateClientConfig() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateCommandTemplate(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid with all placeholders",
			command: "cursor --remote ssh-remote+{user}@{host} {path}",
			wantErr: false,
		},
		{
			name:    "valid with path only",
			command: "nvim {path}",
			wantErr: false,
		},
		{
			name:    "missing path placeholder",
			command: "cursor --remote ssh-remote+{user}@{host}",
			wantErr: true,
			errMsg:  "missing required placeholder: {path}",
		},
		{
			name:    "invalid placeholder",
			command: "cursor {invalid} {path}",
			wantErr: true,
			errMsg:  "unknown placeholder {invalid}",
		},
		{
			name:    "valid with multiple occurrences",
			command: "ssh {user}@{host} editor {path}",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandTemplate(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateCommandTemplate() error = nil, want error containing %q", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateCommandTemplate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateCommandTemplate() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "test.field",
		Message: "test message",
	}

	expected := "config validation error: test.field - test message"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		{Field: "field1", Message: "error1"},
		{Field: "field2", Message: "error2"},
	}

	errStr := errors.Error()
	if !strings.Contains(errStr, "field1") || !strings.Contains(errStr, "error1") {
		t.Errorf("ValidationErrors.Error() missing field1 error: %s", errStr)
	}
	if !strings.Contains(errStr, "field2") || !strings.Contains(errStr, "error2") {
		t.Errorf("ValidationErrors.Error() missing field2 error: %s", errStr)
	}

	// Test empty errors
	emptyErrors := ValidationErrors{}
	if emptyErrors.Error() != "" {
		t.Errorf("Empty ValidationErrors.Error() = %q, want empty string", emptyErrors.Error())
	}
}
