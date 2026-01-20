package main

import (
	"context"
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
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "0.2.1"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

// Command-line flags
var (
	configFile string
	host       string
	port       int
	logLevel   string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "rcode-server",
	Short: "Remote Code Server - Receive editor launch requests",
	Long: `rcode-server is an HTTP server that receives editor launch requests
from remote machines and executes editor commands locally.

By default, it starts the HTTP server listening on the configured host and port.`,
	Version: Version,
	RunE:    runServer,
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "System service management commands",
	Long:  `Commands for installing, managing, and monitoring the rcode-server as a system service.`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install rcode-server as a system service",
	Long:  `Install rcode-server as a system service (launchd on macOS, systemd on Linux).`,
	RunE:  runServiceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall rcode-server system service",
	Long:  `Uninstall the rcode-server system service.`,
	RunE:  runServiceUninstall,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start rcode-server service",
	Long:  `Start the rcode-server system service.`,
	RunE:  runServiceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop rcode-server service",
	Long:  `Stop the rcode-server system service.`,
	RunE:  runServiceStop,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check status of rcode-server service",
	Long:  `Check if the rcode-server system service is installed and running.`,
	RunE:  runServiceStatus,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "Log level (debug, info, warn, error)")

	// Server flags
	rootCmd.Flags().StringVarP(&host, "host", "H", "", "Server host to bind to")
	rootCmd.Flags().IntVarP(&port, "port", "p", 0, "Server port")

	// Legacy flag support (hidden, for backward compatibility)
	rootCmd.Flags().Bool("install-service", false, "Install rcode-server as a system service (use 'rcode-server service install' instead)")
	rootCmd.Flags().Bool("uninstall-service", false, "Uninstall rcode-server system service (use 'rcode-server service uninstall' instead)")
	rootCmd.Flags().Bool("start-service", false, "Start rcode-server service (use 'rcode-server service start' instead)")
	rootCmd.Flags().Bool("stop-service", false, "Stop rcode-server service (use 'rcode-server service stop' instead)")
	rootCmd.Flags().Bool("status-service", false, "Check status of rcode-server service (use 'rcode-server service status' instead)")
	_ = rootCmd.Flags().MarkHidden("install-service")
	_ = rootCmd.Flags().MarkHidden("uninstall-service")
	_ = rootCmd.Flags().MarkHidden("start-service")
	_ = rootCmd.Flags().MarkHidden("stop-service")
	_ = rootCmd.Flags().MarkHidden("status-service")

	// Add subcommands
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceStatusCmd)

	// Custom version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("rcode-server version %s (built %s)\n", Version, BuildTime))
}

func runServer(cmd *cobra.Command, args []string) error {
	// Handle legacy flags
	installService, _ := cmd.Flags().GetBool("install-service")
	uninstallService, _ := cmd.Flags().GetBool("uninstall-service")
	startService, _ := cmd.Flags().GetBool("start-service")
	stopService, _ := cmd.Flags().GetBool("stop-service")
	statusService, _ := cmd.Flags().GetBool("status-service")

	if installService {
		return runServiceInstall(cmd, args)
	}
	if uninstallService {
		return runServiceUninstall(cmd, args)
	}
	if startService {
		return runServiceStart(cmd, args)
	}
	if stopService {
		return runServiceStop(cmd, args)
	}
	if statusService {
		return runServiceStatus(cmd, args)
	}

	// Load configuration
	cfg, err := config.LoadServerConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply command-line overrides
	if host != "" {
		cfg.Server.Host = host
	}
	if port != 0 {
		cfg.Server.Port = port
	}
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	// Validate configuration
	if err := config.ValidateServerConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
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
			return fmt.Errorf("server error: %w", err)
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
	return nil
}

func runServiceInstall(_ *cobra.Command, _ []string) error {
	sm, err := createServiceManager()
	if err != nil {
		return err
	}

	if err := sm.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}
	return nil
}

func runServiceUninstall(_ *cobra.Command, _ []string) error {
	sm, err := createServiceManager()
	if err != nil {
		return err
	}

	if err := sm.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}
	return nil
}

func runServiceStart(_ *cobra.Command, _ []string) error {
	sm, err := createServiceManager()
	if err != nil {
		return err
	}

	if err := sm.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

func runServiceStop(_ *cobra.Command, _ []string) error {
	sm, err := createServiceManager()
	if err != nil {
		return err
	}

	if err := sm.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

func runServiceStatus(_ *cobra.Command, _ []string) error {
	sm, err := createServiceManager()
	if err != nil {
		return err
	}

	isRunning, err := sm.Status()
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}

	isInstalled, err := sm.IsInstalled()
	if err != nil {
		return fmt.Errorf("failed to check if service is installed: %w", err)
	}

	if !isInstalled {
		fmt.Println("Service is not installed.")
		fmt.Println("Run 'rcode-server service install' to install it.")
		return nil
	}

	if isRunning {
		fmt.Println("Service is running.")
	} else {
		fmt.Println("Service is installed but not running.")
		fmt.Println("Run 'rcode-server service start' to start it.")
	}
	return nil
}

func createServiceManager() (*service.ServiceManager, error) {
	// Find the binary path
	binaryPath, err := findBinaryPath()
	if err != nil {
		return nil, fmt.Errorf("failed to find binary path: %w", err)
	}

	// Create service manager
	sm, err := service.NewServiceManager(binaryPath, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create service manager: %w", err)
	}

	return sm, nil
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
