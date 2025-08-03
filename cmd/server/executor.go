package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
)

var (
	// ErrEditorNotFound is returned when the requested editor is not configured
	ErrEditorNotFound = errors.New("editor not found")
	// ErrNoDefaultEditor is returned when no default editor is configured
	ErrNoDefaultEditor = errors.New("no default editor configured")
	// ErrCommandExecution is returned when command execution fails
	ErrCommandExecution = errors.New("command execution failed")
)

// Executor handles editor command execution
type Executor struct {
	editors       []config.EditorConfig
	log           *logger.Logger
	availability  map[string]bool
	availabilityMu sync.RWMutex
}

// NewExecutor creates a new executor
func NewExecutor(editors []config.EditorConfig, log *logger.Logger) *Executor {
	e := &Executor{
		editors:      editors,
		log:          log,
		availability: make(map[string]bool),
	}
	
	// Check editor availability on startup
	e.checkAllEditorsAvailability()
	
	return e
}

// OpenEditor opens a file/directory in the specified editor
func (e *Executor) OpenEditor(editorName, user, host, path string) (string, error) {
	// Find the editor configuration
	var editor *config.EditorConfig
	
	if editorName == "" {
		// Use default editor
		editor = e.getDefaultEditor()
		if editor == nil {
			return "", ErrNoDefaultEditor
		}
		editorName = editor.Name
	} else {
		// Find specific editor
		for i := range e.editors {
			if e.editors[i].Name == editorName {
				editor = &e.editors[i]
				break
			}
		}
		if editor == nil {
			return "", ErrEditorNotFound
		}
	}
	
	// Check if editor is available
	if !e.IsEditorAvailable(editor.Name) {
		e.log.Warn("Editor not available",
			"editor", editor.Name,
			"command", editor.Command,
		)
	}
	
	// Build command from template
	command := e.buildCommand(editor.Command, user, host, path)
	
	// Log the command
	e.log.Debug("Executing editor command",
		"editor", editor.Name,
		"command", command,
		"user", user,
		"host", host,
		"path", path,
	)
	
	// Execute the command
	if err := e.executeCommand(command); err != nil {
		return command, fmt.Errorf("%w: %v", ErrCommandExecution, err)
	}
	
	e.log.Info("Successfully opened editor",
		"editor", editor.Name,
		"path", path,
		"user", user,
		"host", host,
	)
	
	return command, nil
}

// buildCommand replaces placeholders in the command template
func (e *Executor) buildCommand(template, user, host, path string) string {
	command := template
	command = strings.ReplaceAll(command, "{user}", user)
	command = strings.ReplaceAll(command, "{host}", host)
	command = strings.ReplaceAll(command, "{path}", path)
	return command
}

// executeCommand executes the editor command
func (e *Executor) executeCommand(command string) error {
	// Parse command into executable and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	
	executable := parts[0]
	args := parts[1:]
	
	// Create command
	cmd := exec.Command(executable, args...)
	
	// For GUI applications, we want to detach from the parent process
	// so the server doesn't wait for the editor to close
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	
	// Start the command without waiting for it to complete
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Detach the process
	if err := cmd.Process.Release(); err != nil {
		// Non-critical error, log but don't fail
		e.log.Warn("Failed to release process", "error", err)
	}
	
	return nil
}

// IsEditorAvailable checks if an editor is available on the system
func (e *Executor) IsEditorAvailable(name string) bool {
	e.availabilityMu.RLock()
	available, exists := e.availability[name]
	e.availabilityMu.RUnlock()
	
	if exists {
		return available
	}
	
	// Check availability if not cached
	for _, editor := range e.editors {
		if editor.Name == name {
			available := e.checkEditorAvailability(editor)
			
			e.availabilityMu.Lock()
			e.availability[name] = available
			e.availabilityMu.Unlock()
			
			return available
		}
	}
	
	return false
}

// checkEditorAvailability checks if a specific editor is available
func (e *Executor) checkEditorAvailability(editor config.EditorConfig) bool {
	// Extract the executable name from the command template
	parts := strings.Fields(editor.Command)
	if len(parts) == 0 {
		return false
	}
	
	executable := parts[0]
	
	// Check if the executable exists in PATH
	_, err := exec.LookPath(executable)
	available := err == nil
	
	e.log.Debug("Checked editor availability",
		"editor", editor.Name,
		"executable", executable,
		"available", available,
	)
	
	return available
}

// checkAllEditorsAvailability checks availability of all configured editors
func (e *Executor) checkAllEditorsAvailability() {
	e.availabilityMu.Lock()
	defer e.availabilityMu.Unlock()
	
	for _, editor := range e.editors {
		available := e.checkEditorAvailability(editor)
		e.availability[editor.Name] = available
		
		if available {
			e.log.Info("Editor available",
				"editor", editor.Name,
				"default", editor.Default,
			)
		} else {
			e.log.Warn("Editor not available",
				"editor", editor.Name,
				"command", editor.Command,
			)
		}
	}
}

// getDefaultEditor returns the default editor configuration
func (e *Executor) getDefaultEditor() *config.EditorConfig {
	// First, look for explicitly marked default
	for i := range e.editors {
		if e.editors[i].Default {
			return &e.editors[i]
		}
	}
	
	// Then, return first available editor
	for i := range e.editors {
		if e.IsEditorAvailable(e.editors[i].Name) {
			return &e.editors[i]
		}
	}
	
	// Finally, return first editor if any exist
	if len(e.editors) > 0 {
		return &e.editors[0]
	}
	
	return nil
}

// RefreshAvailability refreshes the availability status of all editors
func (e *Executor) RefreshAvailability() {
	e.checkAllEditorsAvailability()
}

// GetEditors returns the list of configured editors
func (e *Executor) GetEditors() []config.EditorConfig {
	return e.editors
}

// ExecuteCustomCommand executes a custom command with the given parameters
// This is useful for testing or advanced use cases
func (e *Executor) ExecuteCustomCommand(commandTemplate, user, host, path string) error {
	command := e.buildCommand(commandTemplate, user, host, path)
	
	e.log.Debug("Executing custom command",
		"command", command,
		"user", user,
		"host", host,
		"path", path,
	)
	
	return e.executeCommand(command)
}

// ValidateEditorConfig validates that an editor configuration is valid
func (e *Executor) ValidateEditorConfig(editor config.EditorConfig) error {
	if editor.Name == "" {
		return errors.New("editor name cannot be empty")
	}
	
	if editor.Command == "" {
		return errors.New("editor command cannot be empty")
	}
	
	// Check that command contains required placeholder
	if !strings.Contains(editor.Command, "{path}") {
		return errors.New("editor command must contain {path} placeholder")
	}
	
	return nil
}

// init performs any initialization needed
func init() {
	// Ensure we can execute commands
	if os.Getenv("PATH") == "" {
		// Set a reasonable default PATH if not set
		os.Setenv("PATH", "/usr/local/bin:/usr/bin:/bin")
	}
}