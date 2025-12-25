package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
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

//nolint:gocyclo // Main function handles multiple command-line flags
func run() int {
	// Parse command-line flags
	var (
		configFile  = flag.String("config", "", "Path to configuration file")
		editor      = flag.String("editor", "", "Editor to use (overrides default)")
		host        = flag.String("host", "", "Server host (overrides config)")
		logLevel    = flag.String("log-level", "", "Log level (debug, info, warn, error)")
		showVersion = flag.Bool("version", false, "Show version information")
		listEditors = flag.Bool("list-editors", false, "List available editors")
		showConfig  = flag.Bool("show-config", false, "Show current configuration")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("rcode version %s (built %s)\n", Version, BuildTime)
		return 0
	}

	// Load configuration
	cfg, err := config.LoadClientConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		return 1
	}

	// Apply command-line overrides
	if *host != "" {
		cfg.Network.PrimaryHost = *host
	}
	if *editor != "" {
		cfg.DefaultEditor = *editor
	}
	if *logLevel != "" {
		cfg.Logging.Level = *logLevel
	}

	// Apply environment variable overrides
	config.MergeClientWithEnvironment(cfg)

	// Validate configuration
	if err := config.ValidateClientConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		return 1
	}

	// Initialize logger
	logConfig := &logger.Config{
		Level:      cfg.Logging.Level,
		Console:    cfg.Logging.Console || *verbose,
		File:       cfg.Logging.File,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
		Format:     "text",
	}

	// Use debug level if verbose flag is set
	if *verbose {
		logConfig.Level = "debug"
		logConfig.Console = true
	}

	log := logger.New(logConfig)
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

	// Show configuration if requested
	if *showConfig {
		showConfiguration(cfg)
		return 0
	}

	// Debug: Log loaded configuration values
	if *verbose {
		log.Debug("Loaded configuration",
			"ssh_host", cfg.SSHHost,
			"primary_host", cfg.Network.PrimaryHost,
			"auto_detect_tailscale", cfg.AutoDetectTailscale,
		)
	}

	// Create client
	client := NewClient(cfg, log)

	// List editors if requested
	if *listEditors {
		if err := client.ListEditors(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list editors: %v\n", err)
			return 1
		}
		return 0
	}

	// Get the path to open (default to current directory)
	path := "."
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve path: %v\n", err)
		return 1
	}

	// Extract SSH connection information
	sshInfo, err := ExtractSSHInfo()
	if err != nil {
		log.Warn("Not in SSH session", "error", err)
		// Continue anyway - might be testing locally
	}

	// If no SSH info, try to use current user and hostname
	if sshInfo.User == "" {
		sshInfo.User = os.Getenv("USER")
		if sshInfo.User == "" {
			sshInfo.User = "unknown"
		}
	}

	// Determine the appropriate host to use
	// Priority: 1. Command-line flag, 2. Config ssh_host (explicit user preference), 3. Auto-detect Tailscale, 4. SSH ClientIP (from ExtractSSHInfo), 5. Default
	// Note: sshInfo.Host is intentionally left empty by ExtractSSHInfo() so we can determine it here
	switch {
	case *host != "":
		// Use the server host as the SSH host when -host flag is provided
		// This ensures that when connecting via Tailscale (e.g., -host ws01tail),
		// the same hostname is used for both the API request and the editor SSH connection
		sshInfo.Host = *host
		log.Debug("Using host from command-line flag", "host", sshInfo.Host)
	case cfg.SSHHost != "":
		// Config ssh_host takes priority over auto-detected values
		// This allows users to explicitly specify the hostname/IP for editor connections
		sshInfo.Host = cfg.SSHHost
		log.Debug("Using ssh_host from config", "host", sshInfo.Host)
	case cfg.AutoDetectTailscale:
		// Try to auto-detect Tailscale connection if enabled
		log.Debug("Attempting Tailscale auto-detection", "clientIP", sshInfo.ClientIP, "pattern", cfg.TailscaleHostPattern)
		if tailHost, isTailscale := DetectTailscaleHost(sshInfo.ClientIP, cfg.TailscaleHostPattern); isTailscale {
			log.Info("Detected Tailscale connection", "tailHost", tailHost, "clientIP", sshInfo.ClientIP)
			sshInfo.Host = tailHost
			// Also update the primary host for the server connection
			cfg.Network.PrimaryHost = tailHost
		} else {
			log.Debug("No Tailscale connection detected")
			// Use ClientIP from SSH_CONNECTION if available
			if sshInfo.ClientIP != "" {
				sshInfo.Host = sshInfo.ClientIP
				log.Debug("Using ClientIP from SSH_CONNECTION", "host", sshInfo.Host)
			} else {
				// Fallback to hostname
				hostname, err := os.Hostname()
				if err == nil {
					sshInfo.Host = hostname
					log.Debug("Using hostname as fallback", "host", sshInfo.Host)
				} else {
					sshInfo.Host = "localhost"
					log.Debug("Using localhost as final fallback")
				}
			}
		}
	case sshInfo.ClientIP != "":
		// Use ClientIP from SSH_CONNECTION (fallback when no config specified)
		// This is the IP address where we SSHed from
		sshInfo.Host = sshInfo.ClientIP
		log.Debug("Using ClientIP from SSH_CONNECTION", "host", sshInfo.Host)
	default:
		// Final fallback
		hostname, err := os.Hostname()
		if err == nil {
			sshInfo.Host = hostname
			log.Debug("Using hostname as fallback", "host", sshInfo.Host)
		} else {
			sshInfo.Host = "localhost"
			log.Debug("Using localhost as final fallback")
		}
	}

	// Log the request details
	log.Info("Opening editor",
		"path", absPath,
		"editor", cfg.DefaultEditor,
		"user", sshInfo.User,
		"host", sshInfo.Host,
		"server", cfg.Network.PrimaryHost,
	)

	// Open the editor
	err = client.OpenEditor(absPath, *editor, &sshInfo)
	if err != nil {
		// Show manual command as fallback
		fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)

		// Generate manual command
		manualCmd := client.GetManualCommand(absPath, *editor, &sshInfo)
		if manualCmd != "" {
			fmt.Fprintf(os.Stderr, "\nYou can try running this command manually on your host machine:\n")
			fmt.Fprintf(os.Stderr, "  %s\n", manualCmd)
		}

		return 1
	}

	fmt.Printf("Successfully opened %s\n", absPath)
	return 0
}

// showConfiguration displays the current configuration
func showConfiguration(cfg *config.ClientConfig) {
	fmt.Println("Current Configuration:")
	fmt.Println("======================")
	fmt.Printf("Network:\n")
	fmt.Printf("  Primary Host: %s\n", cfg.Network.PrimaryHost)
	if cfg.Network.FallbackHost != "" {
		fmt.Printf("  Fallback Host: %s\n", cfg.Network.FallbackHost)
	}
	fmt.Printf("  Timeout: %v\n", cfg.Network.Timeout)
	fmt.Printf("  Retry Attempts: %d\n", cfg.Network.RetryAttempts)
	fmt.Printf("\nSSH Host: %s\n", cfg.SSHHost)
	fmt.Printf("Auto-detect Tailscale: %v\n", cfg.AutoDetectTailscale)
	if cfg.TailscaleHostPattern != "" {
		fmt.Printf("Tailscale Host Pattern: %s\n", cfg.TailscaleHostPattern)
	}
	fmt.Printf("\nDefault Editor: %s\n", cfg.DefaultEditor)

	if len(cfg.Editors) > 0 {
		fmt.Printf("\nConfigured Editors:\n")
		for _, editor := range cfg.Editors {
			fmt.Printf("  - %s: %s\n", editor.Name, editor.Command)
		}
	}

	fmt.Printf("\nLogging:\n")
	fmt.Printf("  Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  File: %s\n", cfg.Logging.File)
}
