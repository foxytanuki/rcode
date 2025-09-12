package main

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strings"
)

// TailscaleStatus represents the minimal Tailscale status we need
type TailscaleStatus struct {
	Self struct {
		HostName string `json:"HostName"`
		DNSName  string `json:"DNSName"`
		TailAddr string `json:"TailscaleIPs,omitempty"`
	} `json:"Self"`
}

// DetectTailscaleHost attempts to detect if we're connected via Tailscale
// and returns an appropriate hostname to use
func DetectTailscaleHost(sshClientIP string, pattern string) (string, bool) {
	// First check if Tailscale is running and we have a Tailscale IP
	tailscaleIP := GetTailscaleInterface()
	if tailscaleIP == "" {
		return "", false
	}

	// Check multiple indicators for Tailscale connection:
	// 1. SSH client IP is in Tailscale range
	// 2. We have a Tailscale interface
	// 3. SSH TTY exists (we're in SSH session)
	isViaTS := isTailscaleIP(sshClientIP) || (tailscaleIP != "" && os.Getenv("SSH_TTY") != "")

	if !isViaTS {
		return "", false
	}

	// Try to get Tailscale hostname
	hostname := getTailscaleHostname()
	if hostname != "" {
		// Try common suffixes used in Tailscale networks
		// Most users append "tail" to their hostname for Tailscale
		baseName := strings.TrimSuffix(hostname, ".tail75a81.ts.net.")
		baseName = strings.TrimSuffix(baseName, ".ts.net.")

		// Apply pattern if provided
		var tailName string
		if pattern != "" {
			// Replace {hostname} placeholder with the base name
			tailName = strings.ReplaceAll(pattern, "{hostname}", baseName)
			// Also support {hostname-} which removes hyphens
			tailName = strings.ReplaceAll(tailName, "{hostname-}", strings.ReplaceAll(baseName, "-", ""))
		} else {
			// Default pattern: ws-01 -> ws01tail
			tailName = strings.ReplaceAll(baseName, "-", "") + "tail"
		}

		return tailName, true
	}

	return "", false
}

// isTailscaleIP checks if an IP is in the Tailscale range (100.64.0.0/10)
func isTailscaleIP(ipStr string) bool {
	if ipStr == "" {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Tailscale uses 100.64.0.0/10 (CGNAT range)
	_, tailscaleNet, _ := net.ParseCIDR("100.64.0.0/10")
	return tailscaleNet != nil && tailscaleNet.Contains(ip)
}

// getTailscaleHostname retrieves the hostname from Tailscale
func getTailscaleHostname() string {
	// Try to run tailscale status command
	cmd := exec.Command("tailscale", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	var status TailscaleStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return ""
	}

	// Return the hostname from Tailscale
	if status.Self.HostName != "" {
		return status.Self.HostName
	}

	// Fallback to DNS name if hostname is not available
	if status.Self.DNSName != "" {
		return status.Self.DNSName
	}

	return ""
}

// GetTailscaleInterface returns the Tailscale interface IP if available
func GetTailscaleInterface() string {
	// Try to get tailscale0 interface IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Name == "tailscale0" || iface.Name == "utun" {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					// Check if it's in Tailscale range
					if isTailscaleIP(ipnet.IP.String()) {
						return ipnet.IP.String()
					}
				}
			}
		}
	}

	return ""
}
