package editor

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidTemplate is returned when a template is invalid
	ErrInvalidTemplate = errors.New("invalid template")
	// ErrMissingPlaceholder is returned when a required placeholder is missing
	ErrMissingPlaceholder = errors.New("missing required placeholder")
)

// Template represents a command template with placeholders
type Template struct {
	raw         string
	hasUser     bool
	hasHost     bool
	hasPath     bool
	placeholders []string
}

// TemplateVars holds the values for template substitution
type TemplateVars struct {
	User string
	Host string
	Path string
}

// NewTemplate creates a new template from a command string
func NewTemplate(command string) (*Template, error) {
	if command == "" {
		return nil, fmt.Errorf("%w: command cannot be empty", ErrInvalidTemplate)
	}

	t := &Template{
		raw:          command,
		placeholders: make([]string, 0),
	}

	// Validate for unknown and unclosed placeholders first
	if err := t.validatePlaceholders(); err != nil {
		return nil, err
	}

	// Check for placeholders
	t.hasUser = strings.Contains(command, "{user}")
	t.hasHost = strings.Contains(command, "{host}")
	t.hasPath = strings.Contains(command, "{path}")

	// Path is required
	if !t.hasPath {
		return nil, fmt.Errorf("%w: {path}", ErrMissingPlaceholder)
	}

	// Collect all placeholders
	if t.hasUser {
		t.placeholders = append(t.placeholders, "{user}")
	}
	if t.hasHost {
		t.placeholders = append(t.placeholders, "{host}")
	}
	if t.hasPath {
		t.placeholders = append(t.placeholders, "{path}")
	}

	return t, nil
}

// validatePlaceholders checks for unknown placeholders
func (t *Template) validatePlaceholders() error {
	// Valid placeholders
	valid := map[string]bool{
		"{user}": true,
		"{host}": true,
		"{path}": true,
	}

	// Check for invalid placeholders
	cmd := t.raw
	start := 0
	for {
		idx := strings.Index(cmd[start:], "{")
		if idx == -1 {
			break
		}
		idx += start

		// Look for the matching closing brace
		end := strings.Index(cmd[idx+1:], "}")
		if end == -1 {
			return fmt.Errorf("%w: unclosed placeholder at position %d", ErrInvalidTemplate, idx)
		}
		end += idx + 1 + 1 // Adjust for the slice starting at idx+1

		// Check if there's a nested brace (which would be invalid)
		innerBrace := strings.Index(cmd[idx+1:end], "{")
		if innerBrace != -1 {
			return fmt.Errorf("%w: unclosed placeholder at position %d", ErrInvalidTemplate, idx)
		}

		placeholder := cmd[idx:end]
		if !valid[placeholder] {
			return fmt.Errorf("%w: unknown placeholder %s", ErrInvalidTemplate, placeholder)
		}

		start = end
	}

	return nil
}

// Render applies the template variables to generate the final command
func (t *Template) Render(vars TemplateVars) (string, error) {
	// Validate required variables
	if t.hasPath && vars.Path == "" {
		return "", fmt.Errorf("path is required for this template")
	}
	if t.hasUser && vars.User == "" {
		return "", fmt.Errorf("user is required for this template")
	}
	if t.hasHost && vars.Host == "" {
		return "", fmt.Errorf("host is required for this template")
	}

	// Perform substitution
	result := t.raw
	result = strings.ReplaceAll(result, "{user}", vars.User)
	result = strings.ReplaceAll(result, "{host}", vars.Host)
	result = strings.ReplaceAll(result, "{path}", vars.Path)

	return result, nil
}

// RenderWithDefaults renders the template with default values for missing vars
func (t *Template) RenderWithDefaults(vars TemplateVars) string {
	result := t.raw

	// Use provided values or defaults
	user := vars.User
	if user == "" {
		user = "user"
	}

	host := vars.Host
	if host == "" {
		host = "localhost"
	}

	path := vars.Path
	if path == "" {
		path = "."
	}

	result = strings.ReplaceAll(result, "{user}", user)
	result = strings.ReplaceAll(result, "{host}", host)
	result = strings.ReplaceAll(result, "{path}", path)

	return result
}

// RequiresUser returns true if the template requires a user variable
func (t *Template) RequiresUser() bool {
	return t.hasUser
}

// RequiresHost returns true if the template requires a host variable
func (t *Template) RequiresHost() bool {
	return t.hasHost
}

// RequiresPath returns true if the template requires a path variable
func (t *Template) RequiresPath() bool {
	return t.hasPath
}

// GetPlaceholders returns the list of placeholders in the template
func (t *Template) GetPlaceholders() []string {
	return t.placeholders
}

// String returns the raw template string
func (t *Template) String() string {
	return t.raw
}

// Clone creates a copy of the template
func (t *Template) Clone() *Template {
	return &Template{
		raw:          t.raw,
		hasUser:      t.hasUser,
		hasHost:      t.hasHost,
		hasPath:      t.hasPath,
		placeholders: append([]string(nil), t.placeholders...),
	}
}

// EscapePath escapes special characters in a path for shell commands
func EscapePath(path string) string {
	// Basic shell escaping - in production, use a proper shell escaping library
	if strings.ContainsAny(path, " \t\n'\"\\$`!") {
		// Use single quotes and escape single quotes
		escaped := strings.ReplaceAll(path, "'", "'\\''")
		return "'" + escaped + "'"
	}
	return path
}

// ExpandPath expands ~ to home directory and resolves relative paths
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		// In a real implementation, get the actual home directory
		// For now, just return as-is since this is called from remote
		return path
	}
	return path
}

// ValidateVars validates template variables
func ValidateVars(vars TemplateVars) error {
	if vars.Path == "" {
		return errors.New("path cannot be empty")
	}

	// Path traversal check
	if strings.Contains(vars.Path, "../") {
		return errors.New("path traversal detected")
	}

	return nil
}

// ParseCommand parses a command string into executable and arguments
func ParseCommand(command string) (string, []string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil
	}

	executable := parts[0]
	args := parts[1:]

	return executable, args
}

// BuildCommand builds a command string from executable and arguments
func BuildCommand(executable string, args []string) string {
	if len(args) == 0 {
		return executable
	}
	return executable + " " + strings.Join(args, " ")
}