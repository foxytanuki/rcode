package editor

import (
	"strings"
	"testing"
)

func TestNewTemplate(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid template with all placeholders",
			command: "ssh {user}@{host} 'editor {path}'",
			wantErr: false,
		},
		{
			name:    "valid template with path only",
			command: "editor {path}",
			wantErr: false,
		},
		{
			name:    "missing path placeholder",
			command: "editor file.txt",
			wantErr: true,
			errMsg:  "missing required placeholder",
		},
		{
			name:    "empty command",
			command: "",
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:    "invalid placeholder",
			command: "editor {invalid} {path}",
			wantErr: true,
			errMsg:  "unknown placeholder",
		},
		{
			name:    "unclosed placeholder",
			command: "editor {path {user}",
			wantErr: true,
			errMsg:  "unclosed placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewTemplate(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Error("NewTemplate() error = nil, want error")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewTemplate() error = %v, want containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewTemplate() error = %v, want nil", err)
				}
				if template == nil {
					t.Error("NewTemplate() returned nil template")
				}
			}
		})
	}
}

func TestTemplate_Render(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		vars     TemplateVars
		want     string
		wantErr  bool
	}{
		{
			name:    "all variables",
			command: "ssh {user}@{host} 'editor {path}'",
			vars: TemplateVars{
				User: "alice",
				Host: "server.com",
				Path: "/home/project",
			},
			want:    "ssh alice@server.com 'editor /home/project'",
			wantErr: false,
		},
		{
			name:    "path only",
			command: "editor {path}",
			vars: TemplateVars{
				Path: "/home/project",
			},
			want:    "editor /home/project",
			wantErr: false,
		},
		{
			name:    "missing required user",
			command: "ssh {user}@{host} {path}",
			vars: TemplateVars{
				Host: "server.com",
				Path: "/home/project",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing required path",
			command: "editor {path}",
			vars:    TemplateVars{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "multiple occurrences",
			command: "{user}@{host}:{path} -> {user}",
			vars: TemplateVars{
				User: "bob",
				Host: "example.com",
				Path: "/tmp",
			},
			want:    "bob@example.com:/tmp -> bob",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewTemplate(tt.command)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			result, err := template.Render(tt.vars)
			if tt.wantErr {
				if err == nil {
					t.Error("Render() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("Render() error = %v, want nil", err)
				}
				if result != tt.want {
					t.Errorf("Render() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestTemplate_RenderWithDefaults(t *testing.T) {
	tests := []struct {
		name    string
		command string
		vars    TemplateVars
		want    string
	}{
		{
			name:    "all defaults",
			command: "ssh {user}@{host} {path}",
			vars:    TemplateVars{},
			want:    "ssh user@localhost .",
		},
		{
			name:    "partial defaults",
			command: "ssh {user}@{host} {path}",
			vars: TemplateVars{
				User: "alice",
			},
			want: "ssh alice@localhost .",
		},
		{
			name:    "no defaults needed",
			command: "editor {path}",
			vars: TemplateVars{
				Path: "/home/project",
			},
			want: "editor /home/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewTemplate(tt.command)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			result := template.RenderWithDefaults(tt.vars)
			if result != tt.want {
				t.Errorf("RenderWithDefaults() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestTemplate_Requirements(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		requiresUser bool
		requiresHost bool
		requiresPath bool
	}{
		{
			name:         "all placeholders",
			command:      "ssh {user}@{host} {path}",
			requiresUser: true,
			requiresHost: true,
			requiresPath: true,
		},
		{
			name:         "path only",
			command:      "editor {path}",
			requiresUser: false,
			requiresHost: false,
			requiresPath: true,
		},
		{
			name:         "user and host",
			command:      "ssh {user}@{host} 'editor {path}'",
			requiresUser: true,
			requiresHost: true,
			requiresPath: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewTemplate(tt.command)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			if template.RequiresUser() != tt.requiresUser {
				t.Errorf("RequiresUser() = %v, want %v", template.RequiresUser(), tt.requiresUser)
			}
			if template.RequiresHost() != tt.requiresHost {
				t.Errorf("RequiresHost() = %v, want %v", template.RequiresHost(), tt.requiresHost)
			}
			if template.RequiresPath() != tt.requiresPath {
				t.Errorf("RequiresPath() = %v, want %v", template.RequiresPath(), tt.requiresPath)
			}
		})
	}
}

func TestTemplate_GetPlaceholders(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		placeholders []string
	}{
		{
			name:         "all placeholders",
			command:      "ssh {user}@{host} {path}",
			placeholders: []string{"{user}", "{host}", "{path}"},
		},
		{
			name:         "path only",
			command:      "editor {path}",
			placeholders: []string{"{path}"},
		},
		{
			name:         "repeated placeholders",
			command:      "{user} {path} {user}",
			placeholders: []string{"{user}", "{path}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewTemplate(tt.command)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			placeholders := template.GetPlaceholders()
			if len(placeholders) != len(tt.placeholders) {
				t.Errorf("GetPlaceholders() returned %d items, want %d", len(placeholders), len(tt.placeholders))
			}

			// Check all expected placeholders are present
			for _, expected := range tt.placeholders {
				found := false
				for _, actual := range placeholders {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Placeholder %v not found in result", expected)
				}
			}
		})
	}
}

func TestValidateVars(t *testing.T) {
	tests := []struct {
		name    string
		vars    TemplateVars
		wantErr bool
	}{
		{
			name: "valid vars",
			vars: TemplateVars{
				User: "alice",
				Host: "server.com",
				Path: "/home/project",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			vars: TemplateVars{
				User: "alice",
				Host: "server.com",
				Path: "",
			},
			wantErr: true,
		},
		{
			name: "path traversal",
			vars: TemplateVars{
				Path: "/home/../etc/passwd",
			},
			wantErr: true,
		},
		{
			name: "path with spaces",
			vars: TemplateVars{
				Path: "/home/my project",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVars(tt.vars)
			if tt.wantErr {
				if err == nil {
					t.Error("ValidateVars() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateVars() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestEscapePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple path",
			path: "/home/user/project",
			want: "/home/user/project",
		},
		{
			name: "path with spaces",
			path: "/home/user/my project",
			want: "'/home/user/my project'",
		},
		{
			name: "path with single quote",
			path: "/home/user/it's mine",
			want: "'/home/user/it'\\''s mine'",
		},
		{
			name: "path with special chars",
			path: "/home/user/$project",
			want: "'/home/user/$project'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapePath(tt.path)
			if got != tt.want {
				t.Errorf("EscapePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		executable string
		args       []string
	}{
		{
			name:       "simple command",
			command:    "editor",
			executable: "editor",
			args:       []string{},
		},
		{
			name:       "command with args",
			command:    "editor --remote file.txt",
			executable: "editor",
			args:       []string{"--remote", "file.txt"},
		},
		{
			name:       "empty command",
			command:    "",
			executable: "",
			args:       nil,
		},
		{
			name:       "command with multiple spaces",
			command:    "editor   --flag   value",
			executable: "editor",
			args:       []string{"--flag", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executable, args := ParseCommand(tt.command)
			if executable != tt.executable {
				t.Errorf("ParseCommand() executable = %v, want %v", executable, tt.executable)
			}
			if len(args) != len(tt.args) {
				t.Errorf("ParseCommand() args length = %v, want %v", len(args), len(tt.args))
			} else {
				for i, arg := range args {
					if arg != tt.args[i] {
						t.Errorf("ParseCommand() args[%d] = %v, want %v", i, arg, tt.args[i])
					}
				}
			}
		})
	}
}

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		args       []string
		want       string
	}{
		{
			name:       "no args",
			executable: "editor",
			args:       []string{},
			want:       "editor",
		},
		{
			name:       "with args",
			executable: "editor",
			args:       []string{"--remote", "file.txt"},
			want:       "editor --remote file.txt",
		},
		{
			name:       "nil args",
			executable: "editor",
			args:       nil,
			want:       "editor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCommand(tt.executable, tt.args)
			if got != tt.want {
				t.Errorf("BuildCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}