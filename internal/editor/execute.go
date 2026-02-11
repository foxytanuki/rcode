// Package editor handles editor command execution.
package editor

import (
	"fmt"
	"os/exec"

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
