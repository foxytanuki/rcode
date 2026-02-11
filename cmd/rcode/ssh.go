package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// SSHInfo contains information about the SSH connection
type SSHInfo struct {
	User       string
	Host       string
	ClientIP   string
	ClientPort string
	ServerIP   string
	ServerPort string
}

// ExtractSSHInfo extracts SSH connection information from environment variables
func ExtractSSHInfo() (SSHInfo, error) {
	info := SSHInfo{}

	// Check SSH_CONNECTION environment variable
	// Format: client_ip client_port server_ip server_port
	sshConnection := os.Getenv("SSH_CONNECTION")
	if sshConnection != "" {
		parts := strings.Fields(sshConnection)
		if len(parts) >= 4 {
			info.ClientIP = parts[0]
			info.ClientPort = parts[1]
			info.ServerIP = parts[2]
			info.ServerPort = parts[3]
		}
	}

	// Check SSH_CLIENT environment variable as fallback
	// Format: client_ip client_port server_port
	if info.ClientIP == "" {
		sshClient := os.Getenv("SSH_CLIENT")
		if sshClient != "" {
			parts := strings.Fields(sshClient)
			if len(parts) >= 3 {
				info.ClientIP = parts[0]
				info.ClientPort = parts[1]
				// Note: SSH_CLIENT doesn't provide server IP, only server port
				if info.ServerPort == "" {
					info.ServerPort = parts[2]
				}
			}
		}
	}

	// Extract username from USER or LOGNAME
	info.User = os.Getenv("USER")
	if info.User == "" {
		info.User = os.Getenv("LOGNAME")
	}

	// Note: Host is NOT set here - it will be determined in main.go
	// based on priority: config ssh_host > ClientIP > hostname > localhost

	// Check if we're actually in an SSH session
	if sshConnection == "" && os.Getenv("SSH_CLIENT") == "" && os.Getenv("SSH_TTY") == "" {
		return SSHInfo{}, errors.New("not in an SSH session")
	}

	return info, nil
}

// String returns a string representation of SSHInfo
func (s SSHInfo) String() string {
	return fmt.Sprintf("user=%s host=%s client=%s:%s server=%s:%s",
		s.User, s.Host, s.ClientIP, s.ClientPort, s.ServerIP, s.ServerPort)
}

// IsValid checks if the SSH info contains minimum required information
func (s *SSHInfo) IsValid() bool {
	return s.User != "" && s.Host != ""
}
