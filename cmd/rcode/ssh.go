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

// IsSSHSession checks if the current process is running in an SSH session
func IsSSHSession() bool {
	// Check various SSH-related environment variables
	return os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("SSH_CLIENT") != "" ||
		os.Getenv("SSH_TTY") != ""
}

// GetSSHUser returns the SSH username, with fallbacks
func GetSSHUser() string {
	// Try SSH_USER first (some systems set this)
	user := os.Getenv("SSH_USER")
	if user != "" {
		return user
	}

	// Try USER
	user = os.Getenv("USER")
	if user != "" {
		return user
	}

	// Try LOGNAME
	user = os.Getenv("LOGNAME")
	if user != "" {
		return user
	}

	// Last resort - try to get from whoami command
	// This is not implemented here to avoid exec dependencies
	return "unknown"
}

// GetSSHHost returns the SSH host information (defaults to client IP)
func GetSSHHost() string {
	// First check if we have client IP from SSH_CONNECTION
	// This is the IP we want - where we SSHed FROM
	sshConnection := os.Getenv("SSH_CONNECTION")
	if sshConnection != "" {
		parts := strings.Fields(sshConnection)
		if len(parts) >= 4 {
			// Return the client IP (where we're SSHing from)
			return parts[0]
		}
	}

	// Check SSH_CLIENT as fallback
	sshClient := os.Getenv("SSH_CLIENT")
	if sshClient != "" {
		parts := strings.Fields(sshClient)
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	// Last resort - use hostname
	hostname, err := os.Hostname()
	if err == nil {
		return hostname
	}

	return "localhost"
}

// ParseSSHDestination parses an SSH destination string (user@host)
func ParseSSHDestination(destination string) (user, host string, err error) {
	if destination == "" {
		return "", "", errors.New("empty destination")
	}

	// Check for user@host format
	if strings.Contains(destination, "@") {
		parts := strings.SplitN(destination, "@", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH destination format: %s", destination)
		}
		user = parts[0]
		host = parts[1]
	} else {
		// No user specified, use current user
		user = GetSSHUser()
		host = destination
	}

	// Remove port if present (host:port)
	if strings.Contains(host, ":") {
		parts := strings.SplitN(host, ":", 2)
		host = parts[0]
	}

	return user, host, nil
}

// FormatSSHCommand formats an SSH command with the given parameters
func FormatSSHCommand(user, host, command string) string {
	if user == "" {
		return fmt.Sprintf("ssh %s '%s'", host, command)
	}
	return fmt.Sprintf("ssh %s@%s '%s'", user, host, command)
}

// GetRemoteInfo returns comprehensive information about the remote environment
func GetRemoteInfo() map[string]string {
	info := make(map[string]string)

	// SSH-related environment variables
	sshVars := []string{
		"SSH_CONNECTION",
		"SSH_CLIENT",
		"SSH_TTY",
		"SSH_AUTH_SOCK",
	}

	for _, v := range sshVars {
		if value := os.Getenv(v); value != "" {
			info[v] = value
		}
	}

	// User and host information
	info["USER"] = os.Getenv("USER")
	info["LOGNAME"] = os.Getenv("LOGNAME")
	info["HOME"] = os.Getenv("HOME")

	hostname, err := os.Hostname()
	if err == nil {
		info["HOSTNAME"] = hostname
	}

	// Working directory
	cwd, err := os.Getwd()
	if err == nil {
		info["PWD"] = cwd
	}

	return info
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
