// Package main implements the rcode server application.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/foxytanuki/rcode/internal/editor"
	"github.com/foxytanuki/rcode/internal/network"
	"github.com/foxytanuki/rcode/internal/version"
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
		Version:   version.Version,
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

	editorList := s.editor.ListEditors()
	editors := make([]api.EditorInfo, 0, len(editorList))

	for _, e := range editorList {
		info := api.EditorInfo{
			Name:      e.Name,
			Type:      string(e.Type),
			Command:   e.Command,
			URL:       e.URL,
			Available: e.Available,
			Default:   e.Default,
		}
		editors = append(editors, info)
	}

	response := api.EditorsResponse{
		Editors:       editors,
		DefaultEditor: s.editor.GetDefaultName(),
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

	// Limit request body size to prevent DoS (1MB)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

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

	// Look up editor via Manager
	e, err := s.editor.GetEditor(req.Editor)
	if err != nil {
		s.log.Error("Failed to find editor",
			"error", err,
			"editor", req.Editor,
		)

		statusCode := http.StatusInternalServerError
		if errors.Is(err, editor.ErrEditorNotFound) || errors.Is(err, editor.ErrNoDefaultEditor) {
			statusCode = http.StatusNotFound
		}

		s.respondError(w, err, statusCode, "")
		return
	}

	resolvedHost := network.ResolveSSHHostAlias(req.Host)

	// Build template variables and render template
	vars := editor.TemplateVars{
		User: req.User,
		Host: resolvedHost,
		Path: req.Path,
	}

	var command string

	if e.Type == "browser" {
		if e.URLTemplate == nil {
			s.log.Error("Missing URL template for browser editor",
				"editor", e.Name,
			)
			s.respondError(w, editor.ErrInvalidEditor, http.StatusInternalServerError, "missing browser URL template")
			return
		}

		command, err = e.URLTemplate.Render(vars)
		if err != nil {
			s.log.Error("Failed to render editor URL",
				"error", err,
				"editor", e.Name,
				"path", req.Path,
			)
			s.respondError(w, err, http.StatusInternalServerError, "")
			return
		}

		// Execute browser open
		if err := editor.OpenBrowser(command, s.log); err != nil {
			s.log.Error("Failed to open browser URL",
				"error", err,
				"editor", e.Name,
				"url", command,
			)
			s.respondError(w, err, http.StatusInternalServerError, "")
			return
		}
	} else {
		command, err = e.Template.Render(vars)
		if err != nil {
			s.log.Error("Failed to render editor command",
				"error", err,
				"editor", e.Name,
				"path", req.Path,
			)
			s.respondError(w, err, http.StatusInternalServerError, "")
			return
		}

		command = normalizeRemoteAuthority(command, req.User, req.Host, resolvedHost)

		// Execute the command
		if err := editor.ExecuteDetached(command, s.log); err != nil {
			s.log.Error("Failed to execute editor command",
				"error", err,
				"editor", e.Name,
				"command", command,
			)
			s.respondError(w, err, http.StatusInternalServerError, "")
			return
		}
	}

	editorName := req.Editor
	if editorName == "" {
		editorName = e.Name
	}

	// Success response
	response := api.OpenResponse{
		Success: true,
		Message: fmt.Sprintf("Opened %s in %s", req.Path, editorName),
		Editor:  editorName,
		Command: command,
	}
	response.SetTimestamp()

	s.respondJSON(w, http.StatusOK, response)
}

func normalizeRemoteAuthority(command, user, originalHost, resolvedHost string) string {
	if user == "" || originalHost == resolvedHost {
		return command
	}

	oldAuthority := "ssh-remote+" + user + "@" + resolvedHost
	newAuthority := "ssh-remote+" + resolvedHost
	return strings.ReplaceAll(command, oldAuthority, newAuthority)
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
