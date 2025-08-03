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
	Version = "dev"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

func main() {
	os.Exit(run())
}

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
	// Check if we have a configured SSH host override
	if cfg.SSHHost != "" {
		sshInfo.Host = cfg.SSHHost
	} else if sshInfo.Host == "" {
		// Only set fallback if truly empty
		sshInfo.Host = "localhost"
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
	err = client.OpenEditor(absPath, *editor, sshInfo)
	if err != nil {
		// Show manual command as fallback
		fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)

		// Generate manual command
		manualCmd := client.GetManualCommand(absPath, *editor, sshInfo)
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
