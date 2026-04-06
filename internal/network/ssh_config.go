package network

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ResolveSSHHostAlias returns a matching SSH config alias for the given host.
// If no exact HostName match is found, the original host is returned.
func ResolveSSHHostAlias(host string) string {
	if host == "" {
		return ""
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return host
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")
	file, err := os.Open(configPath) // #nosec G304 -- configPath is derived from the current user's home directory
	if err != nil {
		return host
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	var aliases []string
	var hostName string

	flush := func() string {
		if hostName != host {
			return ""
		}
		for _, alias := range aliases {
			if alias == "" || hasSSHWildcard(alias) {
				continue
			}
			return alias
		}
		return ""
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.ToLower(fields[0])
		value := strings.Join(fields[1:], " ")

		switch key {
		case "host":
			if alias := flush(); alias != "" {
				return alias
			}
			aliases = fields[1:]
			hostName = ""
		case "hostname":
			hostName = value
		}
	}

	if alias := flush(); alias != "" {
		return alias
	}

	return host
}

func hasSSHWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?!")
}
