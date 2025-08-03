package api

import (
	"errors"
	"fmt"
)

// Common API errors
var (
	// Request validation errors
	ErrInvalidPath    = errors.New("invalid path specified")
	ErrMissingUser    = errors.New("user is required")
	ErrMissingHost    = errors.New("host is required")
	ErrInvalidEditor  = errors.New("invalid editor specified")
	ErrInvalidRequest = errors.New("invalid request format")

	// Editor errors
	ErrEditorNotFound     = errors.New("editor not found")
	ErrEditorNotAvailable = errors.New("editor not available")
	ErrNoDefaultEditor    = errors.New("no default editor configured")
	ErrEditorExecution    = errors.New("failed to execute editor command")

	// Network errors
	ErrConnectionFailed = errors.New("connection failed")
	ErrTimeout          = errors.New("request timeout")
	ErrServerDown       = errors.New("server is not responding")

	// Server errors
	ErrInternalServer = errors.New("internal server error")
	ErrNotImplemented = errors.New("not implemented")
	ErrUnauthorized   = errors.New("unauthorized request")
	ErrRateLimited    = errors.New("rate limit exceeded")
)

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Message   string `json:"error" yaml:"error"`         // Error message
	Code      string `json:"code" yaml:"code"`           // Error code for programmatic handling
	Details   string `json:"details" yaml:"details"`     // Additional error details
	Timestamp int64  `json:"timestamp" yaml:"timestamp"` // Unix timestamp
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err error, code string, details string) *ErrorResponse {
	return &ErrorResponse{
		Message:   err.Error(),
		Code:      code,
		Details:   details,
		Timestamp: timeNow().Unix(),
	}
}

// Error codes for programmatic handling
const (
	CodeInvalidRequest    = "INVALID_REQUEST"
	CodeInvalidPath       = "INVALID_PATH"
	CodeMissingUser       = "MISSING_USER"
	CodeMissingHost       = "MISSING_HOST"
	CodeInvalidEditor     = "INVALID_EDITOR"
	CodeEditorNotFound    = "EDITOR_NOT_FOUND"
	CodeEditorUnavailable = "EDITOR_UNAVAILABLE"
	CodeNoDefaultEditor   = "NO_DEFAULT_EDITOR"
	CodeEditorExecution   = "EDITOR_EXECUTION_ERROR"
	CodeConnectionFailed  = "CONNECTION_FAILED"
	CodeTimeout           = "TIMEOUT"
	CodeServerDown        = "SERVER_DOWN"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeNotImplemented    = "NOT_IMPLEMENTED"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeRateLimited       = "RATE_LIMITED"
)

// GetErrorCode returns the appropriate error code for a given error
func GetErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidPath):
		return CodeInvalidPath
	case errors.Is(err, ErrMissingUser):
		return CodeMissingUser
	case errors.Is(err, ErrMissingHost):
		return CodeMissingHost
	case errors.Is(err, ErrInvalidEditor):
		return CodeInvalidEditor
	case errors.Is(err, ErrEditorNotFound):
		return CodeEditorNotFound
	case errors.Is(err, ErrEditorNotAvailable):
		return CodeEditorUnavailable
	case errors.Is(err, ErrNoDefaultEditor):
		return CodeNoDefaultEditor
	case errors.Is(err, ErrEditorExecution):
		return CodeEditorExecution
	case errors.Is(err, ErrConnectionFailed):
		return CodeConnectionFailed
	case errors.Is(err, ErrTimeout):
		return CodeTimeout
	case errors.Is(err, ErrServerDown):
		return CodeServerDown
	case errors.Is(err, ErrInternalServer):
		return CodeInternalError
	case errors.Is(err, ErrNotImplemented):
		return CodeNotImplemented
	case errors.Is(err, ErrUnauthorized):
		return CodeUnauthorized
	case errors.Is(err, ErrRateLimited):
		return CodeRateLimited
	case errors.Is(err, ErrInvalidRequest):
		return CodeInvalidRequest
	default:
		return CodeInternalError
	}
}

// IsClientError returns true if the error is a client error (4xx)
func IsClientError(err error) bool {
	return errors.Is(err, ErrInvalidPath) ||
		errors.Is(err, ErrMissingUser) ||
		errors.Is(err, ErrMissingHost) ||
		errors.Is(err, ErrInvalidEditor) ||
		errors.Is(err, ErrInvalidRequest) ||
		errors.Is(err, ErrEditorNotFound) ||
		errors.Is(err, ErrNoDefaultEditor) ||
		errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrRateLimited)
}

// IsServerError returns true if the error is a server error (5xx)
func IsServerError(err error) bool {
	return errors.Is(err, ErrInternalServer) ||
		errors.Is(err, ErrEditorNotAvailable) ||
		errors.Is(err, ErrEditorExecution) ||
		errors.Is(err, ErrNotImplemented) ||
		errors.Is(err, ErrServerDown)
}

// IsNetworkError returns true if the error is network-related
func IsNetworkError(err error) bool {
	return errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrServerDown)
}