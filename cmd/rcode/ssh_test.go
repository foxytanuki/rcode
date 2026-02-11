package main

import (
	"os"
	"testing"
)

//nolint:gocyclo // Test function covers multiple scenarios
func TestExtractSSHInfo(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(t *testing.T, info SSHInfo)
	}{
		{
			name: "with SSH_CONNECTION",
			envVars: map[string]string{
				"SSH_CONNECTION": "192.168.1.100 54321 192.168.1.10 22",
				"USER":           "testuser",
			},
			wantErr: false,
			check: func(t *testing.T, info SSHInfo) {
				if info.ClientIP != "192.168.1.100" {
					t.Errorf("ClientIP = %v, want 192.168.1.100", info.ClientIP)
				}
				if info.ClientPort != "54321" {
					t.Errorf("ClientPort = %v, want 54321", info.ClientPort)
				}
				if info.ServerIP != "192.168.1.10" {
					t.Errorf("ServerIP = %v, want 192.168.1.10", info.ServerIP)
				}
				if info.ServerPort != "22" {
					t.Errorf("ServerPort = %v, want 22", info.ServerPort)
				}
				if info.User != "testuser" {
					t.Errorf("User = %v, want testuser", info.User)
				}
			},
		},
		{
			name: "with SSH_CLIENT",
			envVars: map[string]string{
				"SSH_CLIENT": "192.168.1.100 54321 22",
				"LOGNAME":    "testuser",
			},
			wantErr: false,
			check: func(t *testing.T, info SSHInfo) {
				if info.ClientIP != "192.168.1.100" {
					t.Errorf("ClientIP = %v, want 192.168.1.100", info.ClientIP)
				}
				if info.User != "testuser" {
					t.Errorf("User = %v, want testuser", info.User)
				}
			},
		},
		{
			name:    "not in SSH session",
			envVars: map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and clear ALL SSH-related environment variables
			envVarsToSave := []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY", "USER", "LOGNAME"}
			oldEnv := make(map[string]string)

			// Save and clear all relevant environment variables
			for _, k := range envVarsToSave {
				oldEnv[k] = os.Getenv(k)
				_ = os.Unsetenv(k)
			}

			defer func() {
				// Restore original environment
				for k, v := range oldEnv {
					if v != "" {
						_ = os.Setenv(k, v)
					} else {
						_ = os.Unsetenv(k)
					}
				}
			}()

			// Set test environment
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Test
			info, err := ExtractSSHInfo()
			if tt.wantErr {
				if err == nil {
					t.Error("ExtractSSHInfo() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("ExtractSSHInfo() error = %v, want nil", err)
				}
				if tt.check != nil {
					tt.check(t, info)
				}
			}
		})
	}
}

func TestSSHInfo_IsValid(t *testing.T) {
	tests := []struct {
		name string
		info SSHInfo
		want bool
	}{
		{
			name: "valid info",
			info: SSHInfo{
				User: "alice",
				Host: "server.com",
			},
			want: true,
		},
		{
			name: "missing user",
			info: SSHInfo{
				Host: "server.com",
			},
			want: false,
		},
		{
			name: "missing host",
			info: SSHInfo{
				User: "alice",
			},
			want: false,
		},
		{
			name: "empty info",
			info: SSHInfo{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.IsValid()
			if got != tt.want {
				t.Errorf("SSHInfo.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSHInfo_String(t *testing.T) {
	info := SSHInfo{
		User:       "alice",
		Host:       "server.com",
		ClientIP:   "192.168.1.100",
		ClientPort: "54321",
		ServerIP:   "192.168.1.10",
		ServerPort: "22",
	}

	want := "user=alice host=server.com client=192.168.1.100:54321 server=192.168.1.10:22"
	got := info.String()

	if got != want {
		t.Errorf("SSHInfo.String() = %v, want %v", got, want)
	}
}
