package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSSHHostAlias(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		config string
		want   string
	}{
		{
			name:   "prefers exact host alias for matching hostname",
			host:   "192.168.100.20",
			config: "Host ws01\n  User foxy\n  HostName 192.168.100.20\n",
			want:   "ws01",
		},
		{
			name:   "ignores wildcard hosts",
			host:   "192.168.100.20",
			config: "Host *\n  HostName 192.168.100.20\nHost ws01\n  HostName 192.168.100.30\n",
			want:   "192.168.100.20",
		},
		{
			name:   "returns original host when no alias matches",
			host:   "192.168.100.20",
			config: "Host ws01\n  HostName 192.168.100.30\n",
			want:   "192.168.100.20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			sshDir := filepath.Join(homeDir, ".ssh")
			if err := os.MkdirAll(sshDir, 0700); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(tt.config), 0600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			t.Setenv("HOME", homeDir)

			got := ResolveSSHHostAlias(tt.host)
			if got != tt.want {
				t.Fatalf("ResolveSSHHostAlias() = %q, want %q", got, tt.want)
			}
		})
	}
}
