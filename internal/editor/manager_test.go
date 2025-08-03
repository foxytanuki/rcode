package editor

import (
	"testing"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
)

func TestNewManager(t *testing.T) {
	log := createTestLogger()

	tests := []struct {
		name    string
		configs []config.EditorConfig
		wantErr bool
	}{
		{
			name: "valid configs",
			configs: []config.EditorConfig{
				{Name: "editor1", Command: "cmd1 {path}", Default: true},
				{Name: "editor2", Command: "cmd2 {path}"},
			},
			wantErr: false,
		},
		{
			name:    "no configs",
			configs: []config.EditorConfig{},
			wantErr: true,
		},
		{
			name: "invalid config skipped",
			configs: []config.EditorConfig{
				{Name: "", Command: "cmd {path}"}, // Invalid, will be skipped
				{Name: "editor1", Command: "cmd1 {path}"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.configs, log)
			if tt.wantErr {
				if err == nil {
					t.Error("NewManager() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("NewManager() error = %v, want nil", err)
				}
				if manager == nil {
					t.Error("NewManager() returned nil manager")
				}
			}
		})
	}
}

func TestManager_GetEditor(t *testing.T) {
	manager := createTestManager()

	tests := []struct {
		name       string
		editorName string
		wantErr    bool
	}{
		{
			name:       "existing editor",
			editorName: "editor1",
			wantErr:    false,
		},
		{
			name:       "non-existing editor",
			editorName: "nonexistent",
			wantErr:    true,
		},
		{
			name:       "empty name returns default",
			editorName: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor, err := manager.GetEditor(tt.editorName)
			if tt.wantErr {
				if err == nil {
					t.Error("GetEditor() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetEditor() error = %v, want nil", err)
				}
				if editor == nil {
					t.Error("GetEditor() returned nil editor")
				}
			}
		})
	}
}

func TestManager_GetDefaultEditor(t *testing.T) {
	tests := []struct {
		name        string
		configs     []config.EditorConfig
		wantDefault string
	}{
		{
			name: "explicit default",
			configs: []config.EditorConfig{
				{Name: "editor1", Command: "cmd1 {path}"},
				{Name: "editor2", Command: "cmd2 {path}", Default: true},
				{Name: "editor3", Command: "cmd3 {path}"},
			},
			wantDefault: "editor2",
		},
		{
			name: "first editor when no default",
			configs: []config.EditorConfig{
				{Name: "editor1", Command: "cmd1 {path}"},
				{Name: "editor2", Command: "cmd2 {path}"},
			},
			wantDefault: "editor1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.configs, createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			editor, err := manager.GetDefaultEditor()
			if err != nil {
				t.Errorf("GetDefaultEditor() error = %v", err)
			}
			if editor == nil {
				t.Error("GetDefaultEditor() returned nil")
			} else if editor.Name != tt.wantDefault {
				t.Errorf("GetDefaultEditor() = %v, want %v", editor.Name, tt.wantDefault)
			}
		})
	}
}

func TestManager_SetDefault(t *testing.T) {
	manager := createTestManager()

	// Set a different editor as default
	err := manager.SetDefault("editor2")
	if err != nil {
		t.Errorf("SetDefault() error = %v", err)
	}

	// Verify the change
	if manager.GetDefaultName() != "editor2" {
		t.Errorf("GetDefaultName() = %v, want editor2", manager.GetDefaultName())
	}

	// Try to set non-existent editor as default
	err = manager.SetDefault("nonexistent")
	if err == nil {
		t.Error("SetDefault() with non-existent editor should return error")
	}
}

func TestManager_AddEditor(t *testing.T) {
	manager := createTestManager()
	initialCount := manager.Count()

	// Add a new editor
	newConfig := config.EditorConfig{
		Name:    "neweditor",
		Command: "newcmd {path}",
	}

	err := manager.AddEditor(newConfig)
	if err != nil {
		t.Errorf("AddEditor() error = %v", err)
	}

	// Verify the editor was added
	if manager.Count() != initialCount+1 {
		t.Errorf("Count() = %v, want %v", manager.Count(), initialCount+1)
	}

	// Verify we can get the new editor
	editor, err := manager.GetEditor("neweditor")
	if err != nil {
		t.Errorf("GetEditor() error = %v", err)
	}
	if editor == nil || editor.Name != "neweditor" {
		t.Error("New editor not found or incorrect")
	}
}

func TestManager_RemoveEditor(t *testing.T) {
	manager := createTestManager()
	initialCount := manager.Count()

	// Remove an existing editor
	err := manager.RemoveEditor("editor2")
	if err != nil {
		t.Errorf("RemoveEditor() error = %v", err)
	}

	// Verify the editor was removed
	if manager.Count() != initialCount-1 {
		t.Errorf("Count() = %v, want %v", manager.Count(), initialCount-1)
	}

	// Verify we can't get the removed editor
	_, err = manager.GetEditor("editor2")
	if err == nil {
		t.Error("GetEditor() should return error for removed editor")
	}

	// Try to remove non-existent editor
	err = manager.RemoveEditor("nonexistent")
	if err == nil {
		t.Error("RemoveEditor() with non-existent editor should return error")
	}
}

func TestManager_ListEditors(t *testing.T) {
	manager := createTestManager()

	editors := manager.ListEditors()
	if len(editors) == 0 {
		t.Error("ListEditors() returned empty list")
	}

	// Check that all configured editors are present
	editorMap := make(map[string]bool)
	for _, editor := range editors {
		editorMap[editor.Name] = true
	}

	expectedEditors := []string{"editor1", "editor2"}
	for _, expected := range expectedEditors {
		if !editorMap[expected] {
			t.Errorf("Editor %v not found in list", expected)
		}
	}
}

func TestManager_IsAvailable(t *testing.T) {
	manager := createTestManager()

	// Mock availability for testing
	manager.availability["editor1"] = true
	manager.availability["editor2"] = false

	tests := []struct {
		name      string
		editor    string
		available bool
	}{
		{"available editor", "editor1", true},
		{"unavailable editor", "editor2", false},
		{"unknown editor", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available := manager.IsAvailable(tt.editor)
			if available != tt.available {
				t.Errorf("IsAvailable(%v) = %v, want %v", tt.editor, available, tt.available)
			}
		})
	}
}

func TestValidateEditor(t *testing.T) {
	tests := []struct {
		name    string
		config  config.EditorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.EditorConfig{
				Name:    "test",
				Command: "editor {path}",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: config.EditorConfig{
				Command: "editor {path}",
			},
			wantErr: true,
		},
		{
			name: "missing command",
			config: config.EditorConfig{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "missing path placeholder",
			config: config.EditorConfig{
				Name:    "test",
				Command: "editor file.txt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEditor(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("ValidateEditor() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEditor() error = %v, want nil", err)
				}
			}
		})
	}
}

// Helper functions
func createTestManager() *Manager {
	configs := []config.EditorConfig{
		{Name: "editor1", Command: "cmd1 {path}", Default: true},
		{Name: "editor2", Command: "cmd2 {path}"},
	}

	manager, _ := NewManager(configs, createTestLogger())
	return manager
}

func createTestLogger() *logger.Logger {
	return logger.New(&logger.Config{
		Level:   "error",
		Console: false,
	})
}
