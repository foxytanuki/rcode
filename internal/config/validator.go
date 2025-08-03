package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s - %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// ValidateServerConfig validates server configuration
func ValidateServerConfig(config *ServerConfigFile) error {
	var errors ValidationErrors

	// Validate server settings
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		errors = append(errors, ValidationError{
			Field:   "server.port",
			Message: fmt.Sprintf("invalid port number: %d", config.Server.Port),
		})
	}

	// Validate IP whitelist if specified
	for i, ip := range config.Server.AllowedIPs {
		if _, _, err := net.ParseCIDR(ip); err != nil {
			if net.ParseIP(ip) == nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("server.allowed_ips[%d]", i),
					Message: fmt.Sprintf("invalid IP or CIDR: %s", ip),
				})
			}
		}
	}

	// Validate timeouts
	if config.Server.ReadTimeout < 0 {
		errors = append(errors, ValidationError{
			Field:   "server.read_timeout",
			Message: "timeout cannot be negative",
		})
	}
	if config.Server.WriteTimeout < 0 {
		errors = append(errors, ValidationError{
			Field:   "server.write_timeout",
			Message: "timeout cannot be negative",
		})
	}
	if config.Server.IdleTimeout < 0 {
		errors = append(errors, ValidationError{
			Field:   "server.idle_timeout",
			Message: "timeout cannot be negative",
		})
	}

	// Validate editors
	if len(config.Editors) == 0 {
		errors = append(errors, ValidationError{
			Field:   "editors",
			Message: "at least one editor must be configured",
		})
	}

	editorNames := make(map[string]bool)
	defaultCount := 0
	for i, editor := range config.Editors {
		if editor.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("editors[%d].name", i),
				Message: "editor name cannot be empty",
			})
		}
		if editor.Command == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("editors[%d].command", i),
				Message: "editor command cannot be empty",
			})
		}
		
		// Check for duplicate names
		if editorNames[editor.Name] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("editors[%d].name", i),
				Message: fmt.Sprintf("duplicate editor name: %s", editor.Name),
			})
		}
		editorNames[editor.Name] = true
		
		// Count default editors
		if editor.Default {
			defaultCount++
		}
		
		// Validate command template
		if err := validateCommandTemplate(editor.Command); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("editors[%d].command", i),
				Message: err.Error(),
			})
		}
	}

	if defaultCount > 1 {
		errors = append(errors, ValidationError{
			Field:   "editors",
			Message: "only one editor can be marked as default",
		})
	}

	// Validate logging
	if err := validateLogConfig(&config.Logging); err != nil {
		errors = append(errors, err...)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// ValidateClientConfig validates client configuration
func ValidateClientConfig(config *ClientConfig) error {
	var errors ValidationErrors

	// Validate network settings
	if config.Network.PrimaryHost == "" {
		errors = append(errors, ValidationError{
			Field:   "network.primary_host",
			Message: "primary host cannot be empty",
		})
	}
	
	if config.Network.Timeout < 0 {
		errors = append(errors, ValidationError{
			Field:   "network.timeout",
			Message: "timeout cannot be negative",
		})
	}
	
	if config.Network.RetryAttempts < 0 {
		errors = append(errors, ValidationError{
			Field:   "network.retry_attempts",
			Message: "retry attempts cannot be negative",
		})
	}
	
	if config.Network.RetryDelay < 0 {
		errors = append(errors, ValidationError{
			Field:   "network.retry_delay",
			Message: "retry delay cannot be negative",
		})
	}

	// Validate default editor if specified
	if config.DefaultEditor != "" && len(config.Editors) > 0 {
		found := false
		for _, editor := range config.Editors {
			if editor.Name == config.DefaultEditor {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, ValidationError{
				Field:   "default_editor",
				Message: fmt.Sprintf("default editor '%s' not found in editors list", config.DefaultEditor),
			})
		}
	}

	// Validate logging
	if err := validateLogConfig(&config.Logging); err != nil {
		errors = append(errors, err...)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// validateLogConfig validates logging configuration
func validateLogConfig(config *LogConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	
	if !validLevels[strings.ToLower(config.Level)] {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Message: fmt.Sprintf("invalid log level: %s (must be debug, info, warn, or error)", config.Level),
		})
	}

	// Validate log file path
	if config.File != "" {
		dir := filepath.Dir(config.File)
		if dir != "" && dir != "." {
			// Check if parent directory is writable
			if err := checkDirectoryWritable(dir); err != nil {
				errors = append(errors, ValidationError{
					Field:   "logging.file",
					Message: fmt.Sprintf("log directory not writable: %s", err),
				})
			}
		}
	}

	// Validate log rotation settings
	if config.MaxSize < 0 {
		errors = append(errors, ValidationError{
			Field:   "logging.max_size",
			Message: "max size cannot be negative",
		})
	}
	
	if config.MaxBackups < 0 {
		errors = append(errors, ValidationError{
			Field:   "logging.max_backups",
			Message: "max backups cannot be negative",
		})
	}
	
	if config.MaxAge < 0 {
		errors = append(errors, ValidationError{
			Field:   "logging.max_age",
			Message: "max age cannot be negative",
		})
	}

	return errors
}

// validateCommandTemplate validates an editor command template
func validateCommandTemplate(command string) error {
	// Check for required placeholders
	requiredPlaceholders := []string{"{path}"}
	for _, placeholder := range requiredPlaceholders {
		if !strings.Contains(command, placeholder) {
			return fmt.Errorf("missing required placeholder: %s", placeholder)
		}
	}

	// Check for invalid placeholders
	validPlaceholders := map[string]bool{
		"{user}": true,
		"{host}": true,
		"{path}": true,
	}
	
	// Simple check for placeholder-like patterns
	for i := 0; i < len(command); i++ {
		if command[i] == '{' {
			end := strings.Index(command[i:], "}")
			if end > 0 {
				placeholder := command[i : i+end+1]
				if !validPlaceholders[placeholder] {
					return fmt.Errorf("invalid placeholder: %s", placeholder)
				}
			}
		}
	}

	return nil
}

// checkDirectoryWritable checks if a directory is writable
func checkDirectoryWritable(dir string) error {
	// If directory doesn't exist, check parent
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("cannot determine parent directory")
		}
		return checkDirectoryWritable(parent)
	}

	// Try to create a temporary file to test writability
	testFile := filepath.Join(dir, ".rcode_write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	}
	file.Close()
	os.Remove(testFile)
	
	return nil
}

// ValidateConfig validates a complete configuration
func ValidateConfig(config *Config) error {
	var errors ValidationErrors

	// Validate editors
	if len(config.Editors) == 0 {
		errors = append(errors, ValidationError{
			Field:   "editors",
			Message: "at least one editor must be configured",
		})
	}

	// Validate that default editor exists if specified
	if config.DefaultEditor != "" {
		found := false
		for _, editor := range config.Editors {
			if editor.Name == config.DefaultEditor {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, ValidationError{
				Field:   "default_editor",
				Message: fmt.Sprintf("default editor '%s' not found", config.DefaultEditor),
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}