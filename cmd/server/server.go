package main

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/editor"
	"github.com/foxytanuki/rcode/internal/logger"
)

// Server represents the HTTP server
type Server struct {
	config      *config.ServerConfigFile
	log         *logger.Logger
	editor      *editor.Manager
	startTime   time.Time
	allowedIPs  []net.IP
	allowedNets []*net.IPNet
}

// NewServer creates a new server instance
func NewServer(cfg *config.ServerConfigFile, log *logger.Logger) (*Server, error) {
	mgr, err := editor.NewManager(cfg.Editors, log)
	if err != nil {
		return nil, err
	}

	// Parse IP whitelist once at startup
	var allowedIPs []net.IP
	var allowedNets []*net.IPNet
	for _, allowed := range cfg.Server.AllowedIPs {
		if strings.Contains(allowed, "/") {
			if _, ipNet, err := net.ParseCIDR(allowed); err == nil {
				allowedNets = append(allowedNets, ipNet)
			}
		} else {
			if ip := net.ParseIP(allowed); ip != nil {
				allowedIPs = append(allowedIPs, ip)
			}
		}
	}

	return &Server{
		config:      cfg,
		log:         log,
		editor:      mgr,
		startTime:   time.Now(),
		allowedIPs:  allowedIPs,
		allowedNets: allowedNets,
	}, nil
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
