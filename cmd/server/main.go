package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
	"github.com/foxytanuki/rcode/internal/service"
)

var (
	// Version is set at build time
	Version = "0.1.1"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse command-line flags
	var (
		configFile       = flag.String("config", "", "Path to configuration file")
		host             = flag.String("host", "", "Server host to bind to")
		port             = flag.Int("port", 0, "Server port")
		logLevel         = flag.String("log-level", "", "Log level (debug, info, warn, error)")
		showVersion      = flag.Bool("version", false, "Show version information")
		installService   = flag.Bool("install-service", false, "Install rcode-server as a system service")
		uninstallService = flag.Bool("uninstall-service", false, "Uninstall rcode-server system service")
		startService     = flag.Bool("start-service", false, "Start rcode-server service")
		stopService      = flag.Bool("stop-service", false, "Stop rcode-server service")
		statusService    = flag.Bool("status-service", false, "Check status of rcode-server service")
	)
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("rcode-server version %s (built %s)\n", Version, BuildTime)
		return 0
	}

	// Handle service management commands
	if *installService || *uninstallService || *startService || *stopService || *statusService {
		return handleServiceCommands(*installService, *uninstallService, *startService, *stopService, *statusService, *configFile)
	}

	// Load configuration
	cfg, err := config.LoadServerConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		return 1
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
		return 1
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
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

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
			return 1
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
			if err := httpServer.Close(); err != nil {
				log.Error("Failed to close HTTP server", "error", err)
			}
		}
	}

	log.Info("Server stopped")
	return 0
}

// handleServiceCommands handles service management commands
func handleServiceCommands(install, uninstall, start, stop, status bool, configPath string) int {
	// Find the binary path
	binaryPath, err := findBinaryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find binary path: %v\n", err)
		return 1
	}

	// Create service manager
	sm, err := service.NewServiceManager(binaryPath, configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service manager: %v\n", err)
		return 1
	}

	// Execute the requested command
	switch {
	case install:
		if err := sm.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", err)
			return 1
		}
		return 0

	case uninstall:
		if err := sm.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to uninstall service: %v\n", err)
			return 1
		}
		return 0

	case start:
		if err := sm.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start service: %v\n", err)
			return 1
		}
		return 0

	case stop:
		if err := sm.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stop service: %v\n", err)
			return 1
		}
		return 0

	case status:
		isRunning, err := sm.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check service status: %v\n", err)
			return 1
		}

		isInstalled, err := sm.IsInstalled()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check if service is installed: %v\n", err)
			return 1
		}

		if !isInstalled {
			fmt.Println("Service is not installed.")
			fmt.Println("Run 'rcode-server -install-service' to install it.")
			return 1
		}

		if isRunning {
			fmt.Println("Service is running.")
		} else {
			fmt.Println("Service is installed but not running.")
			fmt.Println("Run 'rcode-server -start-service' to start it.")
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "No service command specified\n")
		return 1
	}
}

// findBinaryPath finds the path to the rcode-server binary
func findBinaryPath() (string, error) {
	// Try to find the binary in PATH
	path, err := exec.LookPath("rcode-server")
	if err == nil {
		return path, nil
	}

	// If not in PATH, try to get the path of the current executable
	execPath, err := os.Executable()
	if err == nil {
		// Resolve symlinks
		resolvedPath, err := filepath.EvalSymlinks(execPath)
		if err == nil {
			return resolvedPath, nil
		}
		return execPath, nil
	}

	return "", fmt.Errorf("rcode-server binary not found")
}
