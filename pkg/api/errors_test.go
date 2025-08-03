package api

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestErrorResponse_Error(t *testing.T) {
	tests := []struct {
		name     string
		response ErrorResponse
		want     string
	}{
		{
			name: "error with details",
			response: ErrorResponse{
				Message: "test error",
				Details: "additional information",
			},
			want: "test error: additional information",
		},
		{
			name: "error without details",
			response: ErrorResponse{
				Message: "test error",
			},
			want: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.Error(); got != tt.want {
				t.Errorf("ErrorResponse.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	// Mock time for consistent testing
	originalTimeNow := timeNow
	mockTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return mockTime }
	defer func() { timeNow = originalTimeNow }()

	err := errors.New("test error")
	code := "TEST_CODE"
	details := "test details"

	resp := NewErrorResponse(err, code, details)

	if resp.Message != err.Error() {
		t.Errorf("NewErrorResponse().Message = %v, want %v", resp.Message, err.Error())
	}
	if resp.Code != code {
		t.Errorf("NewErrorResponse().Code = %v, want %v", resp.Code, code)
	}
	if resp.Details != details {
		t.Errorf("NewErrorResponse().Details = %v, want %v", resp.Details, details)
	}
	if resp.Timestamp != mockTime.Unix() {
		t.Errorf("NewErrorResponse().Timestamp = %v, want %v", resp.Timestamp, mockTime.Unix())
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"invalid path", ErrInvalidPath, CodeInvalidPath},
		{"missing user", ErrMissingUser, CodeMissingUser},
		{"missing host", ErrMissingHost, CodeMissingHost},
		{"invalid editor", ErrInvalidEditor, CodeInvalidEditor},
		{"editor not found", ErrEditorNotFound, CodeEditorNotFound},
		{"editor unavailable", ErrEditorNotAvailable, CodeEditorUnavailable},
		{"no default editor", ErrNoDefaultEditor, CodeNoDefaultEditor},
		{"editor execution", ErrEditorExecution, CodeEditorExecution},
		{"connection failed", ErrConnectionFailed, CodeConnectionFailed},
		{"timeout", ErrTimeout, CodeTimeout},
		{"server down", ErrServerDown, CodeServerDown},
		{"internal server", ErrInternalServer, CodeInternalError},
		{"not implemented", ErrNotImplemented, CodeNotImplemented},
		{"unauthorized", ErrUnauthorized, CodeUnauthorized},
		{"rate limited", ErrRateLimited, CodeRateLimited},
		{"invalid request", ErrInvalidRequest, CodeInvalidRequest},
		{"unknown error", errors.New("unknown"), CodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.want {
				t.Errorf("GetErrorCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"invalid path", ErrInvalidPath, true},
		{"missing user", ErrMissingUser, true},
		{"missing host", ErrMissingHost, true},
		{"invalid editor", ErrInvalidEditor, true},
		{"invalid request", ErrInvalidRequest, true},
		{"editor not found", ErrEditorNotFound, true},
		{"no default editor", ErrNoDefaultEditor, true},
		{"unauthorized", ErrUnauthorized, true},
		{"rate limited", ErrRateLimited, true},
		{"internal server error", ErrInternalServer, false},
		{"connection failed", ErrConnectionFailed, false},
		{"timeout", ErrTimeout, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClientError(tt.err); got != tt.want {
				t.Errorf("IsClientError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"internal server", ErrInternalServer, true},
		{"editor unavailable", ErrEditorNotAvailable, true},
		{"editor execution", ErrEditorExecution, true},
		{"not implemented", ErrNotImplemented, true},
		{"server down", ErrServerDown, true},
		{"invalid path", ErrInvalidPath, false},
		{"connection failed", ErrConnectionFailed, false},
		{"timeout", ErrTimeout, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsServerError(tt.err); got != tt.want {
				t.Errorf("IsServerError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection failed", ErrConnectionFailed, true},
		{"timeout", ErrTimeout, true},
		{"server down", ErrServerDown, true},
		{"invalid path", ErrInvalidPath, false},
		{"internal server", ErrInternalServer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.want {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	// Test that all error messages are properly defined
	errs := []error{
		ErrInvalidPath,
		ErrMissingUser,
		ErrMissingHost,
		ErrInvalidEditor,
		ErrInvalidRequest,
		ErrEditorNotFound,
		ErrEditorNotAvailable,
		ErrNoDefaultEditor,
		ErrEditorExecution,
		ErrConnectionFailed,
		ErrTimeout,
		ErrServerDown,
		ErrInternalServer,
		ErrNotImplemented,
		ErrUnauthorized,
		ErrRateLimited,
	}

	for _, err := range errs {
		if err == nil {
			t.Error("Error variable is nil")
			continue
		}
		if err.Error() == "" {
			t.Errorf("Error message is empty for %v", err)
		}
		// Check that error messages don't have trailing spaces or newlines
		msg := err.Error()
		if strings.HasSuffix(msg, " ") || strings.HasSuffix(msg, "\n") {
			t.Errorf("Error message has trailing whitespace: %q", msg)
		}
	}
}
