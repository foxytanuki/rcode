// Package service provides functionality to install and manage rcode-server as a system service.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ServiceManager handles service installation and management
//
//nolint:revive // ServiceManager is a clear name and the package is not intended to be shortened
type ServiceManager struct {
	binaryPath string
	configPath string
	userHome   string
}

// NewServiceManager creates a new service manager instance
func NewServiceManager(binaryPath, configPath string) (*ServiceManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	return &ServiceManager{
		binaryPath: binaryPath,
		configPath: configPath,
		userHome:   home,
	}, nil
}

// Install installs rcode-server as a system service
func (sm *ServiceManager) Install() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.installDarwin()
	case "linux":
		return sm.installLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall removes rcode-server from system services
func (sm *ServiceManager) Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.uninstallDarwin()
	case "linux":
		return sm.uninstallLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Start starts the rcode-server service
func (sm *ServiceManager) Start() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.startDarwin()
	case "linux":
		return sm.startLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Stop stops the rcode-server service
func (sm *ServiceManager) Stop() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.stopDarwin()
	case "linux":
		return sm.stopLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Status checks the status of the rcode-server service
func (sm *ServiceManager) Status() (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		return sm.statusDarwin()
	case "linux":
		return sm.statusLinux()
	default:
		return false, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// installDarwin installs rcode-server as a launchd service on macOS
func (sm *ServiceManager) installDarwin() error {
	// Find the binary path
	binaryPath, err := sm.findBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to find binary: %w", err)
	}

	// Create LaunchAgents directory if it doesn't exist
	launchAgentsDir := filepath.Join(sm.userHome, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0750); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Generate plist content
	plistContent := sm.generateDarwinPlist(binaryPath)

	// Write plist file
	plistPath := filepath.Join(launchAgentsDir, "com.foxytanuki.rcode-server.plist")
	if err := os.WriteFile(plistPath, []byte(plistContent), 0600); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the service
	//nolint:gosec // G204: plistPath is a controlled path constructed from user home directory
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		// If already loaded, try to unload first
		//nolint:gosec // G204: plistPath is a controlled path constructed from user home directory
		_ = exec.Command("launchctl", "unload", plistPath).Run()
		//nolint:gosec // G204: plistPath is a controlled path constructed from user home directory
		cmd = exec.Command("launchctl", "load", plistPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to load service: %w", err)
		}
	}

	fmt.Printf("Service installed successfully at %s\n", plistPath)
	fmt.Println("The service will start automatically on login.")
	return nil
}

// uninstallDarwin removes rcode-server from launchd on macOS
func (sm *ServiceManager) uninstallDarwin() error {
	plistPath := filepath.Join(sm.userHome, "Library", "LaunchAgents", "com.foxytanuki.rcode-server.plist")

	// Unload the service if it's running
	//nolint:gosec // G204: plistPath is a controlled path constructed from user home directory
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	fmt.Println("Service uninstalled successfully.")
	return nil
}

// startDarwin starts the rcode-server service on macOS
func (sm *ServiceManager) startDarwin() error {
	plistPath := filepath.Join(sm.userHome, "Library", "LaunchAgents", "com.foxytanuki.rcode-server.plist")

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return fmt.Errorf("service not installed. Run 'rcode-server install-service' first")
	}

	cmd := exec.Command("launchctl", "start", "com.foxytanuki.rcode-server")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("Service started successfully.")
	return nil
}

// stopDarwin stops the rcode-server service on macOS
func (sm *ServiceManager) stopDarwin() error {
	cmd := exec.Command("launchctl", "stop", "com.foxytanuki.rcode-server")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("Service stopped successfully.")
	return nil
}

// statusDarwin checks the status of the rcode-server service on macOS
func (sm *ServiceManager) statusDarwin() (bool, error) {
	cmd := exec.Command("launchctl", "list", "com.foxytanuki.rcode-server")
	output, err := cmd.Output()
	if err != nil {
		// If the service is not loaded, launchctl list returns an error
		return false, nil
	}

	// If we get output, the service is loaded
	return len(output) > 0, nil
}

// installLinux installs rcode-server as a systemd user service on Linux
func (sm *ServiceManager) installLinux() error {
	// Find the binary path
	binaryPath, err := sm.findBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to find binary: %w", err)
	}

	// Create systemd user directory if it doesn't exist
	systemdUserDir := filepath.Join(sm.userHome, ".config", "systemd", "user")
	if err := os.MkdirAll(systemdUserDir, 0750); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	// Generate service file content
	serviceContent := sm.generateLinuxService(binaryPath)

	// Write service file
	servicePath := filepath.Join(systemdUserDir, "rcode-server.service")
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0600); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable the service
	cmd = exec.Command("systemctl", "--user", "enable", "rcode-server.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Printf("Service installed successfully at %s\n", servicePath)
	fmt.Println("The service will start automatically on login.")
	return nil
}

// uninstallLinux removes rcode-server from systemd on Linux
func (sm *ServiceManager) uninstallLinux() error {
	servicePath := filepath.Join(sm.userHome, ".config", "systemd", "user", "rcode-server.service")

	// Disable the service
	_ = exec.Command("systemctl", "--user", "disable", "rcode-server.service").Run()

	// Stop the service
	_ = exec.Command("systemctl", "--user", "stop", "rcode-server.service").Run()

	// Reload systemd
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	// Remove the service file
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	fmt.Println("Service uninstalled successfully.")
	return nil
}

// startLinux starts the rcode-server service on Linux
func (sm *ServiceManager) startLinux() error {
	servicePath := filepath.Join(sm.userHome, ".config", "systemd", "user", "rcode-server.service")

	// Check if service file exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return fmt.Errorf("service not installed. Run 'rcode-server install-service' first")
	}

	cmd := exec.Command("systemctl", "--user", "start", "rcode-server.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("Service started successfully.")
	return nil
}

// stopLinux stops the rcode-server service on Linux
func (sm *ServiceManager) stopLinux() error {
	cmd := exec.Command("systemctl", "--user", "stop", "rcode-server.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("Service stopped successfully.")
	return nil
}

// statusLinux checks the status of the rcode-server service on Linux
func (sm *ServiceManager) statusLinux() (bool, error) {
	cmd := exec.Command("systemctl", "--user", "is-active", "--quiet", "rcode-server.service")
	err := cmd.Run()
	if err != nil {
		// If the service is not active, is-active returns a non-zero exit code
		return false, nil
	}

	return true, nil
}

// generateDarwinPlist generates the launchd plist content for macOS
func (sm *ServiceManager) generateDarwinPlist(binaryPath string) string {
	args := []string{binaryPath}
	if sm.configPath != "" {
		args = append(args, "-config", sm.configPath)
	}

	// Build ProgramArguments array
	argsXML := ""
	for _, arg := range args {
		argsXML += fmt.Sprintf("\t\t<string>%s</string>\n", arg)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.foxytanuki.rcode-server</string>
	<key>ProgramArguments</key>
	<array>
%s	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s/.local/share/rcode/logs/service.log</string>
	<key>StandardErrorPath</key>
	<string>%s/.local/share/rcode/logs/service-error.log</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>
</dict>
</plist>`, argsXML, sm.userHome, sm.userHome)
}

// generateLinuxService generates the systemd service file content for Linux
func (sm *ServiceManager) generateLinuxService(binaryPath string) string {
	execStart := binaryPath
	if sm.configPath != "" {
		execStart = fmt.Sprintf("%s -config %s", binaryPath, sm.configPath)
	}

	return fmt.Sprintf(`[Unit]
Description=RCode Server - Remote Code Launcher
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=5
StandardOutput=append:%s/.local/share/rcode/logs/service.log
StandardError=append:%s/.local/share/rcode/logs/service-error.log
Environment="PATH=/usr/local/bin:/usr/bin:/bin"

[Install]
WantedBy=default.target
`, execStart, sm.userHome, sm.userHome)
}

// findBinaryPath finds the path to the rcode-server binary
func (sm *ServiceManager) findBinaryPath() (string, error) {
	// If binaryPath is already absolute and exists, use it
	if filepath.IsAbs(sm.binaryPath) {
		if _, err := os.Stat(sm.binaryPath); err == nil {
			return sm.binaryPath, nil
		}
	}

	// Try to find the binary in PATH
	binaryName := "rcode-server"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	path, err := exec.LookPath(binaryName)
	if err != nil {
		// If not in PATH, try common installation locations
		commonPaths := []string{
			"/usr/local/bin/rcode-server",
			"/usr/bin/rcode-server",
			filepath.Join(sm.userHome, "bin", "rcode-server"),
		}

		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}

		return "", fmt.Errorf("rcode-server binary not found in PATH or common locations")
	}

	return path, nil
}

// IsInstalled checks if the service is installed
func (sm *ServiceManager) IsInstalled() (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		plistPath := filepath.Join(sm.userHome, "Library", "LaunchAgents", "com.foxytanuki.rcode-server.plist")
		_, err := os.Stat(plistPath)
		return !os.IsNotExist(err), nil
	case "linux":
		servicePath := filepath.Join(sm.userHome, ".config", "systemd", "user", "rcode-server.service")
		_, err := os.Stat(servicePath)
		return !os.IsNotExist(err), nil
	default:
		return false, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
