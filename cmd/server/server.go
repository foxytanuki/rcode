package main

import (
	"net/http"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
)

// Server represents the HTTP server
type Server struct {
	config    *config.ServerConfigFile
	log       *logger.Logger
	executor  *Executor
	startTime time.Time
}

// NewServer creates a new server instance
func NewServer(cfg *config.ServerConfigFile, log *logger.Logger) *Server {
	return &Server{
		config:    cfg,
		log:       log,
		executor:  NewExecutor(cfg.Editors, log),
		startTime: time.Now(),
	}
}

// Router returns the HTTP handler with all routes configured
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Apply middleware
	handler := s.withMiddleware(mux)

	// Register routes
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/editors", s.handleEditors)
	mux.HandleFunc("/open-editor", s.handleOpenEditor)

	return handler
}

// withMiddleware applies middleware to the handler
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in reverse order (last one runs first)
	handler = s.recoveryMiddleware(handler)
	handler = s.loggingMiddleware(handler)
	handler = s.ipWhitelistMiddleware(handler)
	return handler
}