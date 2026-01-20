package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "0.1.1"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

// Command-line flags
var (
	configFile  string
	editor      string
	host        string
	logLevel    string
	verbose     bool
	listEditors bool
	showConfig  bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "rcode [path]",
	Short: "Remote Code Launcher - Open code editors from remote machines",
	Long: `rcode is a CLI tool that allows launching host machine code editors
from SSH-connected remote machines without requiring SSH server on the host.

By default, it opens the current directory or the specified path in the configured editor.`,
	Args:    cobra.MaximumNArgs(1),
	Version: Version,
	RunE:    runOpen,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Commands for managing rcode configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current rcode configuration including network settings, editors, and logging.`,
	RunE:  runConfigShow,
}

var editorsCmd = &cobra.Command{
	Use:   "editors",
	Short: "List available editors",
	Long:  `List all configured editors that can be used with rcode.`,
	RunE:  runListEditors,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Root command flags
	rootCmd.Flags().StringVarP(&editor, "editor", "e", "", "Editor to use (overrides default)")
	rootCmd.Flags().StringVarP(&host, "host", "H", "", "Server host (overrides config)")

	// Legacy flag support (hidden, for backward compatibility)
	rootCmd.Flags().BoolVar(&listEditors, "list-editors", false, "List available editors (use 'rcode editors' instead)")
	rootCmd.Flags().BoolVar(&showConfig, "show-config", false, "Show current configuration (use 'rcode config show' instead)")
	_ = rootCmd.Flags().MarkHidden("list-editors")
	_ = rootCmd.Flags().MarkHidden("show-config")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(editorsCmd)
	configCmd.AddCommand(configShowCmd)

	// Custom version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("rcode version %s (built %s)\n", Version, BuildTime))
}

//nolint:gocyclo // Main function handles multiple command-line flags
func runOpen(cmd *cobra.Command, args []string) error {
	// Handle legacy flags
	if showConfig {
		return runConfigShow(cmd, args)
	}
	if listEditors {
		return runListEditors(cmd, args)
	}

	// Load configuration
	cfg, err := config.LoadClientConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply command-line overrides
	if host != "" {
		cfg.Network.PrimaryHost = host
	}
	if editor != "" {
		cfg.DefaultEditor = editor
	}
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	// Apply environment variable overrides
	config.MergeClientWithEnvironment(cfg)

	// Validate configuration
	if err := config.ValidateClientConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize logger
	logConfig := &logger.Config{
		Level:      cfg.Logging.Level,
		Console:    cfg.Logging.Console || verbose,
		File:       cfg.Logging.File,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
		Format:     "text",
	}

	// Use debug level if verbose flag is set
	if verbose {
		logConfig.Level = "debug"
		logConfig.Console = true
	}

	log := logger.New(logConfig)
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

	// Debug: Log loaded configuration values
	if verbose {
		log.Debug("Loaded configuration",
			"ssh_host", cfg.SSHHost,
			"primary_host", cfg.Network.PrimaryHost,
			"auto_detect_tailscale", cfg.AutoDetectTailscale,
		)
	}

	// Create client
	client := NewClient(cfg, log)

	// Get the path to open (default to current directory)
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
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
	case host != "":
		// Use the server host as the SSH host when -host flag is provided
		// This ensures that when connecting via Tailscale (e.g., -host ws01tail),
		// the same hostname is used for both the API request and the editor SSH connection
		sshInfo.Host = host
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
	err = client.OpenEditor(absPath, editor, &sshInfo)
	if err != nil {
		// Show manual command as fallback
		fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)

		// Generate manual command
		manualCmd := client.GetManualCommand(absPath, editor, &sshInfo)
		if manualCmd != "" {
			fmt.Fprintf(os.Stderr, "\nYou can try running this command manually on your host machine:\n")
			fmt.Fprintf(os.Stderr, "  %s\n", manualCmd)
		}

		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("Successfully opened %s\n", absPath)
	return nil
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.LoadClientConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	showConfiguration(cfg)
	return nil
}

func runListEditors(_ *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.LoadClientConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger (minimal for this command)
	log := logger.New(&logger.Config{
		Level:   "error",
		Console: true,
		Format:  "text",
	})
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

	// Create client and list editors
	client := NewClient(cfg, log)
	if err := client.ListEditors(); err != nil {
		return fmt.Errorf("failed to list editors: %w", err)
	}
	return nil
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
	fmt.Printf("  (Editor definitions are fetched from the server. Use --list-editors to see available editors.)\n")

	fmt.Printf("\nLogging:\n")
	fmt.Printf("  Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  File: %s\n", cfg.Logging.File)
}
