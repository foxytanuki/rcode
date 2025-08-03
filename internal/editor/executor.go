package editor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/foxytanuki/rcode/internal/logger"
)

var (
	// ErrExecutionFailed is returned when command execution fails
	ErrExecutionFailed = errors.New("command execution failed")
	// ErrTimeout is returned when command execution times out
	ErrTimeout = errors.New("command execution timeout")
)

// Executor handles the execution of editor commands
type Executor struct {
	manager *Manager
	log     *logger.Logger
	timeout time.Duration
}

// ExecutorOption is a functional option for configuring the executor
type ExecutorOption func(*Executor)

// WithTimeout sets the execution timeout
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = timeout
	}
}

// NewExecutor creates a new executor
func NewExecutor(manager *Manager, log *logger.Logger, opts ...ExecutorOption) *Executor {
	e := &Executor{
		manager: manager,
		log:     log,
		timeout: 30 * time.Second, // Default timeout
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute opens a file/directory in the specified editor
func (e *Executor) Execute(editorName string, vars TemplateVars) error {
	// Validate variables
	if err := ValidateVars(vars); err != nil {
		return fmt.Errorf("invalid variables: %w", err)
	}

	// Get the editor
	editor, err := e.manager.GetEditor(editorName)
	if err != nil {
		return err
	}

	// Check availability
	if !e.manager.IsAvailable(editor.Name) {
		e.log.Warn("Editor not available, attempting anyway",
			"editor", editor.Name,
		)
	}

	// Render the command template
	command, err := editor.Template.Render(vars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Log the execution
	e.log.Info("Executing editor command",
		"editor", editor.Name,
		"command", command,
		"user", vars.User,
		"host", vars.Host,
		"path", vars.Path,
	)

	// Execute the command
	if err := e.executeCommand(command); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	e.log.Info("Editor launched successfully",
		"editor", editor.Name,
		"path", vars.Path,
	)

	return nil
}

// ExecuteDefault opens a file/directory in the default editor
func (e *Executor) ExecuteDefault(vars TemplateVars) error {
	editor, err := e.manager.GetDefaultEditor()
	if err != nil {
		return err
	}

	return e.Execute(editor.Name, vars)
}

// executeCommand executes the editor command
func (e *Executor) executeCommand(command string) error {
	// Parse command
	executable, args := ParseCommand(command)
	if executable == "" {
		return errors.New("empty command")
	}

	// Create command
	cmd := exec.Command(executable, args...)

	// For GUI editors, we want to detach from the parent process
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Set environment variables if needed
	cmd.Env = os.Environ()

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start editor: %w", err)
	}

	// For GUI applications, release the process so we don't wait
	if err := cmd.Process.Release(); err != nil {
		// Non-critical error, just log it
		e.log.Debug("Failed to release process", "error", err)
	}

	return nil
}

// ExecuteWithContext executes a command with a context for cancellation
func (e *Executor) ExecuteWithContext(ctx context.Context, editorName string, vars TemplateVars) error {
	// Create a channel to signal completion
	done := make(chan error, 1)

	// Execute in a goroutine
	go func() {
		done <- e.Execute(editorName, vars)
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
	}
}

// ExecuteAndWait executes a command and waits for it to complete
func (e *Executor) ExecuteAndWait(editorName string, vars TemplateVars) error {
	// Get the editor
	editor, err := e.manager.GetEditor(editorName)
	if err != nil {
		return err
	}

	// Render the command template
	command, err := editor.Template.Render(vars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Parse command
	executable, args := ParseCommand(command)
	if executable == "" {
		return errors.New("empty command")
	}

	// Create command with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)

	// Log the execution
	e.log.Debug("Executing and waiting for editor",
		"editor", editor.Name,
		"command", command,
		"timeout", e.timeout,
	)

	// Run the command and wait for completion
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w after %v", ErrTimeout, e.timeout)
		}
		return fmt.Errorf("%w: %v (output: %s)", ErrExecutionFailed, err, string(output))
	}

	return nil
}

// TestCommand tests if a command can be executed without actually running it
func (e *Executor) TestCommand(editorName string, vars TemplateVars) error {
	// Get the editor
	editor, err := e.manager.GetEditor(editorName)
	if err != nil {
		return err
	}

	// Validate variables
	if err := ValidateVars(vars); err != nil {
		return fmt.Errorf("invalid variables: %w", err)
	}

	// Render the command template
	command, err := editor.Template.Render(vars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Parse command
	executable, _ := ParseCommand(command)
	if executable == "" {
		return errors.New("empty command")
	}

	// Check if executable exists
	if _, err := exec.LookPath(executable); err != nil {
		return fmt.Errorf("executable not found: %w", err)
	}

	return nil
}

// ListAvailableEditors returns a list of available editors
func (e *Executor) ListAvailableEditors() []string {
	editors := e.manager.ListEditors()
	available := make([]string, 0, len(editors))

	for _, editor := range editors {
		if e.manager.IsAvailable(editor.Name) {
			available = append(available, editor.Name)
		}
	}

	return available
}

// GetEditorCommand returns the rendered command for an editor
func (e *Executor) GetEditorCommand(editorName string, vars TemplateVars) (string, error) {
	// Get the editor
	editor, err := e.manager.GetEditor(editorName)
	if err != nil {
		return "", err
	}

	// Render the command template
	command, err := editor.Template.Render(vars)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return command, nil
}

// ExecuteCustom executes a custom command with the given template
func (e *Executor) ExecuteCustom(commandTemplate string, vars TemplateVars) error {
	// Create a temporary template
	template, err := NewTemplate(commandTemplate)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	// Render the command
	command, err := template.Render(vars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Log the execution
	e.log.Debug("Executing custom command",
		"command", command,
		"user", vars.User,
		"host", vars.Host,
		"path", vars.Path,
	)

	// Execute the command
	if err := e.executeCommand(command); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

// RefreshAvailability refreshes the availability status of all editors
func (e *Executor) RefreshAvailability() {
	e.manager.RefreshAvailability()
}

// SupportsRemoteEditing checks if an editor supports remote editing
func (e *Executor) SupportsRemoteEditing(editorName string) bool {
	editor, err := e.manager.GetEditor(editorName)
	if err != nil {
		return false
	}

	// Check if the command template includes host/user variables
	return editor.Template.RequiresHost() || editor.Template.RequiresUser()
}

// NormalizeEditorName normalizes an editor name for case-insensitive lookup
func NormalizeEditorName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}