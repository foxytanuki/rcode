package main

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		s.log.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// recoveryMiddleware recovers from panics
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.log.Error("Panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ipWhitelistMiddleware restricts access based on IP whitelist
func (s *Server) ipWhitelistMiddleware(next http.Handler) http.Handler {
	// If no whitelist configured, allow all
	if len(s.config.Server.AllowedIPs) == 0 {
		return next
	}

	// Parse allowed IPs and CIDRs
	var allowedNets []*net.IPNet
	var allowedIPs []net.IP

	for _, allowed := range s.config.Server.AllowedIPs {
		if strings.Contains(allowed, "/") {
			// CIDR notation
			_, ipNet, err := net.ParseCIDR(allowed)
			if err == nil {
				allowedNets = append(allowedNets, ipNet)
			}
		} else {
			// Single IP
			if ip := net.ParseIP(allowed); ip != nil {
				allowedIPs = append(allowedIPs, ip)
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		clientIP := getClientIP(r)
		ip := net.ParseIP(clientIP)

		if ip == nil {
			s.log.Warn("Could not parse client IP",
				"remote_addr", r.RemoteAddr,
				"client_ip", clientIP,
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check if IP is allowed
		allowed := false

		// Check single IPs
		for _, allowedIP := range allowedIPs {
			if ip.Equal(allowedIP) {
				allowed = true
				break
			}
		}

		// Check CIDR ranges
		if !allowed {
			for _, ipNet := range allowedNets {
				if ipNet.Contains(ip) {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			s.log.Warn("Access denied by IP whitelist",
				"client_ip", clientIP,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}
