package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/foxytanuki/rcode/pkg/api"
)

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, api.ErrNotImplemented, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uptime := time.Since(s.startTime).Seconds()
	response := api.HealthResponse{
		Status:    "healthy",
		Version:   Version,
		Uptime:    int64(uptime),
		StartedAt: s.startTime,
	}
	response.SetTimestamp()

	s.respondJSON(w, http.StatusOK, response)
}

// handleEditors handles GET /editors
func (s *Server) handleEditors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, api.ErrNotImplemented, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	editors := make([]api.EditorInfo, 0, len(s.config.Editors))
	defaultEditor := ""

	for _, editor := range s.config.Editors {
		info := api.EditorInfo{
			Name:      editor.Name,
			Command:   editor.Command,
			Available: s.executor.IsEditorAvailable(editor.Name),
			Default:   editor.Default,
		}
		editors = append(editors, info)
		
		if editor.Default {
			defaultEditor = editor.Name
		}
	}

	response := api.EditorsResponse{
		Editors:       editors,
		DefaultEditor: defaultEditor,
	}
	response.SetTimestamp()

	s.respondJSON(w, http.StatusOK, response)
}

// handleOpenEditor handles POST /open-editor
func (s *Server) handleOpenEditor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, api.ErrNotImplemented, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req api.OpenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, api.ErrInvalidRequest, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		s.respondError(w, err, http.StatusBadRequest, "")
		return
	}

	// Log the request
	s.log.Info("Open editor request",
		"path", req.Path,
		"editor", req.Editor,
		"user", req.User,
		"host", req.Host,
		"remote_addr", r.RemoteAddr,
	)

	// Execute editor command
	command, err := s.executor.OpenEditor(req.Editor, req.User, req.Host, req.Path)
	if err != nil {
		s.log.Error("Failed to open editor",
			"error", err,
			"editor", req.Editor,
			"path", req.Path,
		)
		
		// Determine appropriate error code
		statusCode := http.StatusInternalServerError
		if err == ErrEditorNotFound {
			statusCode = http.StatusNotFound
		}
		
		s.respondError(w, err, statusCode, "")
		return
	}

	// Success response
	response := api.OpenResponse{
		Success: true,
		Message: fmt.Sprintf("Opened %s in %s", req.Path, req.Editor),
		Editor:  req.Editor,
		Command: command,
	}
	response.SetTimestamp()

	s.respondJSON(w, http.StatusOK, response)
}

// respondJSON sends a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.log.Error("Failed to encode JSON response", "error", err)
	}
}

// respondError sends an error response
func (s *Server) respondError(w http.ResponseWriter, err error, status int, details string) {
	response := api.NewErrorResponse(err, api.GetErrorCode(err), details)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		s.log.Error("Failed to encode error response", "error", encodeErr)
	}
}