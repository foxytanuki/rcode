package editor

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
)

var (
	// ErrNoEditors is returned when no editors are configured
	ErrNoEditors = errors.New("no editors configured")
	// ErrEditorNotFound is returned when a requested editor is not found
	ErrEditorNotFound = errors.New("editor not found")
	// ErrNoDefaultEditor is returned when no default editor can be determined
	ErrNoDefaultEditor = errors.New("no default editor available")
	// ErrInvalidEditor is returned when editor configuration is invalid
	ErrInvalidEditor = errors.New("invalid editor configuration")
)

// Manager manages available editors
type Manager struct {
	editors      map[string]*Editor
	defaultName  string
	log          *logger.Logger
	mu           sync.RWMutex
	availability map[string]bool
	availMu      sync.RWMutex
}

// Editor represents a single editor configuration
type Editor struct {
	Name      string
	Command   string
	Default   bool
	Available bool
	Template  *Template
}

// NewManager creates a new editor manager
func NewManager(configs []config.EditorConfig, log *logger.Logger) (*Manager, error) {
	if len(configs) == 0 {
		return nil, ErrNoEditors
	}

	m := &Manager{
		editors:      make(map[string]*Editor),
		log:          log,
		availability: make(map[string]bool),
	}

	// Initialize editors from config
	for _, cfg := range configs {
		editor, err := NewEditor(cfg)
		if err != nil {
			log.Warn("Skipping invalid editor configuration",
				"name", cfg.Name,
				"error", err,
			)
			continue
		}

		m.editors[editor.Name] = editor

		// Track default editor
		if editor.Default && m.defaultName == "" {
			m.defaultName = editor.Name
		}
	}

	// If no default was explicitly set, use the first available
	if m.defaultName == "" && len(m.editors) > 0 {
		for name := range m.editors {
			m.defaultName = name
			break
		}
	}

	// Check availability of all editors
	m.RefreshAvailability()

	return m, nil
}

// NewEditor creates a new editor from configuration
func NewEditor(cfg config.EditorConfig) (*Editor, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidEditor)
	}

	if cfg.Command == "" {
		return nil, fmt.Errorf("%w: command is required", ErrInvalidEditor)
	}

	template, err := NewTemplate(cfg.Command)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid command template: %v", ErrInvalidEditor, err)
	}

	return &Editor{
		Name:      cfg.Name,
		Command:   cfg.Command,
		Default:   cfg.Default,
		Available: cfg.Available,
		Template:  template,
	}, nil
}

// GetEditor returns an editor by name
func (m *Manager) GetEditor(name string) (*Editor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == "" {
		return m.getDefaultEditor()
	}

	editor, exists := m.editors[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrEditorNotFound, name)
	}

	return editor, nil
}

// GetDefaultEditor returns the default editor
func (m *Manager) GetDefaultEditor() (*Editor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getDefaultEditor()
}

// getDefaultEditor returns the default editor (must be called with lock held)
func (m *Manager) getDefaultEditor() (*Editor, error) {
	// First try the explicitly set default
	if m.defaultName != "" {
		if editor, exists := m.editors[m.defaultName]; exists {
			return editor, nil
		}
	}

	// Then try to find any available editor
	for name, editor := range m.editors {
		if m.IsAvailable(name) {
			return editor, nil
		}
	}

	// Finally, return any editor
	for _, editor := range m.editors {
		return editor, nil
	}

	return nil, ErrNoDefaultEditor
}

// ListEditors returns all configured editors
func (m *Manager) ListEditors() []*Editor {
	m.mu.RLock()
	defer m.mu.RUnlock()

	editors := make([]*Editor, 0, len(m.editors))
	for _, editor := range m.editors {
		// Create a copy to avoid mutation
		editorCopy := *editor
		editorCopy.Available = m.IsAvailable(editor.Name)
		editors = append(editors, &editorCopy)
	}

	return editors
}

// IsAvailable checks if an editor is available on the system
func (m *Manager) IsAvailable(name string) bool {
	m.availMu.RLock()
	available, exists := m.availability[name]
	m.availMu.RUnlock()

	if exists {
		return available
	}

	// Check availability if not cached
	m.mu.RLock()
	editor, editorExists := m.editors[name]
	m.mu.RUnlock()

	if !editorExists {
		return false
	}

	available = m.checkAvailability(editor)

	m.availMu.Lock()
	m.availability[name] = available
	m.availMu.Unlock()

	return available
}

// RefreshAvailability refreshes the availability status of all editors
func (m *Manager) RefreshAvailability() {
	m.mu.RLock()
	editors := make([]*Editor, 0, len(m.editors))
	for _, editor := range m.editors {
		editors = append(editors, editor)
	}
	m.mu.RUnlock()

	m.availMu.Lock()
	defer m.availMu.Unlock()

	for _, editor := range editors {
		available := m.checkAvailability(editor)
		m.availability[editor.Name] = available

		if available {
			m.log.Debug("Editor available",
				"name", editor.Name,
				"default", editor.Default,
			)
		} else {
			m.log.Debug("Editor not available",
				"name", editor.Name,
			)
		}
	}
}

// checkAvailability checks if an editor is available
func (m *Manager) checkAvailability(editor *Editor) bool {
	// Extract the executable from the command
	executable := m.extractExecutable(editor.Command)
	if executable == "" {
		return false
	}

	// Check if the executable exists in PATH
	_, err := exec.LookPath(executable)
	return err == nil
}

// extractExecutable extracts the executable name from a command string
func (m *Manager) extractExecutable(command string) string {
	// Find the first part before any flags or arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	// The executable is the first part
	executable := parts[0]

	// Remove any template variables from the executable name
	if strings.Contains(executable, "{") {
		return ""
	}

	return executable
}

// SetDefault sets the default editor
func (m *Manager) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.editors[name]; !exists {
		return fmt.Errorf("%w: %s", ErrEditorNotFound, name)
	}

	// Clear previous default
	for _, editor := range m.editors {
		editor.Default = false
	}

	// Set new default
	m.editors[name].Default = true
	m.defaultName = name

	m.log.Info("Default editor changed", "editor", name)

	return nil
}

// AddEditor adds a new editor to the manager
func (m *Manager) AddEditor(cfg config.EditorConfig) error {
	editor, err := NewEditor(cfg)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.editors[editor.Name] = editor

	// Check availability
	available := m.checkAvailability(editor)
	m.availMu.Lock()
	m.availability[editor.Name] = available
	m.availMu.Unlock()

	m.log.Info("Editor added",
		"name", editor.Name,
		"available", available,
	)

	return nil
}

// RemoveEditor removes an editor from the manager
func (m *Manager) RemoveEditor(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.editors[name]; !exists {
		return fmt.Errorf("%w: %s", ErrEditorNotFound, name)
	}

	delete(m.editors, name)

	m.availMu.Lock()
	delete(m.availability, name)
	m.availMu.Unlock()

	// Update default if necessary
	if m.defaultName == name {
		m.defaultName = ""
		for editorName := range m.editors {
			m.defaultName = editorName
			break
		}
	}

	m.log.Info("Editor removed", "name", name)

	return nil
}

// ValidateEditor validates an editor configuration
func ValidateEditor(cfg config.EditorConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidEditor)
	}

	if cfg.Command == "" {
		return fmt.Errorf("%w: command is required", ErrInvalidEditor)
	}

	// Validate command template
	if _, err := NewTemplate(cfg.Command); err != nil {
		return fmt.Errorf("%w: invalid command template: %v", ErrInvalidEditor, err)
	}

	return nil
}

// Count returns the number of configured editors
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.editors)
}

// GetDefaultName returns the name of the default editor
func (m *Manager) GetDefaultName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultName
}
