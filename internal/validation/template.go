// Package validation provides shared validation logic for rcode.
package validation

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidTemplate is returned when a template is invalid.
	ErrInvalidTemplate = errors.New("invalid template")
	// ErrMissingPlaceholder is returned when a required placeholder is missing.
	ErrMissingPlaceholder = errors.New("missing required placeholder")
)

// ValidPlaceholders defines the set of allowed placeholders.
var ValidPlaceholders = map[string]bool{
	"{user}": true,
	"{host}": true,
	"{path}": true,
}

// ValidateCommandTemplate validates an editor command template for correct placeholders.
func ValidateCommandTemplate(command string) error {
	if command == "" {
		return fmt.Errorf("%w: command cannot be empty", ErrInvalidTemplate)
	}

	// Scan for placeholder-like patterns and validate them first
	// (catches unclosed/unknown before checking required placeholders)
	start := 0
	for {
		idx := strings.Index(command[start:], "{")
		if idx == -1 {
			break
		}
		idx += start

		end := strings.Index(command[idx+1:], "}")
		if end == -1 {
			return fmt.Errorf("%w: unclosed placeholder at position %d", ErrInvalidTemplate, idx)
		}
		end += idx + 1 + 1

		// Check for nested braces
		if innerBrace := strings.Index(command[idx+1:end], "{"); innerBrace != -1 {
			return fmt.Errorf("%w: unclosed placeholder at position %d", ErrInvalidTemplate, idx)
		}

		placeholder := command[idx:end]
		if !ValidPlaceholders[placeholder] {
			return fmt.Errorf("%w: unknown placeholder %s", ErrInvalidTemplate, placeholder)
		}

		start = end
	}

	// Check for required {path} placeholder
	if !strings.Contains(command, "{path}") {
		return fmt.Errorf("%w: {path}", ErrMissingPlaceholder)
	}

	return nil
}
