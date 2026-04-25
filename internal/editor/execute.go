// Package editor handles editor command execution.
package editor

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/foxytanuki/rcode/internal/logger"
)

// ExecuteDetached executes a command string, detaching the process for GUI editors.
func ExecuteDetached(command string, log *logger.Logger) error {
	executable, args := ParseCommand(command)
	if executable == "" {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(executable, args...) // #nosec G204
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	if err := cmd.Process.Release(); err != nil {
		log.Warn("Failed to release process", "error", err)
	}

	return nil
}

// OpenBrowser opens a URL using the OS default browser.
func OpenBrowser(url string, log *logger.Logger) error {
	if url == "" {
		return fmt.Errorf("empty url")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		if execErr, ok := err.(*exec.Error); ok {
			return fmt.Errorf("failed to open browser: %s not found", execErr.Name)
		}
		return fmt.Errorf("failed to open browser: %w", err)
	}

	if err := cmd.Process.Release(); err != nil {
		log.Warn("Failed to release browser process", "error", err)
	}

	return nil
}
