package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
)

var (
	// Version is set at build time
	Version = "dev"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

func main() {
	// Parse command-line flags
	var (
		configFile  = flag.String("config", "", "Path to configuration file")
		host        = flag.String("host", "", "Server host to bind to")
		port        = flag.Int("port", 0, "Server port")
		logLevel    = flag.String("log-level", "", "Log level (debug, info, warn, error)")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("rcode-server version %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadServerConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Apply command-line overrides
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *logLevel != "" {
		cfg.Logging.Level = *logLevel
	}

	// Validate configuration
	if err := config.ValidateServerConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:      cfg.Logging.Level,
		Console:    cfg.Logging.Console,
		File:       cfg.Logging.File,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
		Format:     "text",
	})
	defer log.Close()

	// Log startup information
	log.Info("Starting rcode-server",
		"version", Version,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"editors", len(cfg.Editors),
	)

	// Create server instance
	srv := NewServer(cfg, log)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      srv.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Server listening", "address", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		log.Info("Shutdown signal received", "signal", sig)

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		log.Info("Shutting down server gracefully...")
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error("Server shutdown error", "error", err)
			httpServer.Close()
		}
	}

	log.Info("Server stopped")
}