package api

import (
	"time"
)

// OpenRequest represents a request to open a file/directory in an editor
type OpenRequest struct {
	Path      string `json:"path" yaml:"path"`           // Path to open
	Editor    string `json:"editor" yaml:"editor"`       // Editor to use (optional, uses default if empty)
	User      string `json:"user" yaml:"user"`           // SSH username
	Host      string `json:"host" yaml:"host"`           // Remote hostname
	Timestamp int64  `json:"timestamp" yaml:"timestamp"` // Unix timestamp
}

// OpenResponse represents the response from an open editor request
type OpenResponse struct {
	Success   bool   `json:"success" yaml:"success"`     // Whether the operation succeeded
	Message   string `json:"message" yaml:"message"`     // Success or error message
	Editor    string `json:"editor" yaml:"editor"`       // Editor that was used
	Command   string `json:"command" yaml:"command"`     // Command that was executed
	Timestamp int64  `json:"timestamp" yaml:"timestamp"` // Unix timestamp
}

// EditorInfo represents information about an available editor
type EditorInfo struct {
	Name      string `json:"name" yaml:"name"`           // Editor name (e.g., "cursor", "vscode")
	Command   string `json:"command" yaml:"command"`     // Command template
	Available bool   `json:"available" yaml:"available"` // Whether the editor is available
	Default   bool   `json:"default" yaml:"default"`     // Whether this is the default editor
}

// EditorsResponse represents the response from the /editors endpoint
type EditorsResponse struct {
	Editors       []EditorInfo `json:"editors" yaml:"editors"`               // List of available editors
	DefaultEditor string       `json:"default_editor" yaml:"default_editor"` // Name of the default editor
	Timestamp     int64        `json:"timestamp" yaml:"timestamp"`           // Unix timestamp
}

// HealthResponse represents the response from the /health endpoint
type HealthResponse struct {
	Status    string    `json:"status" yaml:"status"`       // "healthy" or "unhealthy"
	Version   string    `json:"version" yaml:"version"`     // Server version
	Uptime    int64     `json:"uptime" yaml:"uptime"`       // Uptime in seconds
	Timestamp int64     `json:"timestamp" yaml:"timestamp"` // Unix timestamp
	StartedAt time.Time `json:"started_at" yaml:"started_at"` // Server start time
}

// Validate validates an OpenRequest
func (r *OpenRequest) Validate() error {
	if r.Path == "" {
		return ErrInvalidPath
	}
	if r.User == "" {
		return ErrMissingUser
	}
	if r.Host == "" {
		return ErrMissingHost
	}
	return nil
}

// SetTimestamp sets the current timestamp on the request
func (r *OpenRequest) SetTimestamp() {
	r.Timestamp = time.Now().Unix()
}

// SetTimestamp sets the current timestamp on the response
func (r *OpenResponse) SetTimestamp() {
	r.Timestamp = time.Now().Unix()
}

// SetTimestamp sets the current timestamp on the response
func (r *EditorsResponse) SetTimestamp() {
	r.Timestamp = time.Now().Unix()
}

// SetTimestamp sets the current timestamp on the response
func (r *HealthResponse) SetTimestamp() {
	r.Timestamp = time.Now().Unix()
}

// IsHealthy returns true if the status is healthy
func (r *HealthResponse) IsHealthy() bool {
	return r.Status == "healthy"
}