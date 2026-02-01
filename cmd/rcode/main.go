package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
	"github.com/foxytanuki/rcode/internal/network"
	"github.com/foxytanuki/rcode/internal/version"
	"github.com/spf13/cobra"
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
	Version: version.Version,
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
	rootCmd.SetVersionTemplate(fmt.Sprintf("rcode version %s\nBuilt: %s\nGit: %s\n", version.Version, version.BuildTime, version.GitHash))
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
		cfg.Hosts.Server.Primary = host
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
			"ssh_host", cfg.Hosts.SSH.Host,
			"primary_host", cfg.Hosts.Server.Primary,
			"auto_detect_tailscale", cfg.Hosts.SSH.AutoDetect.Tailscale,
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

	// Use the Resolver to determine hosts
	resolver := buildResolver(cfg, host, sshInfo.ClientIP)
	resolved := resolver.Resolve()

	// Apply resolved hosts
	sshInfo.Host = resolved.SSH
	if resolved.Server != "" {
		cfg.Hosts.Server.Primary = resolved.Server
	}
	if resolved.ServerFallback != "" {
		cfg.Hosts.Server.Fallback = resolved.ServerFallback
	}

	log.Debug("Host resolution completed",
		"ssh_host", sshInfo.Host,
		"source", resolved.Source,
		"server", cfg.Hosts.Server.Primary,
	)

	// Log the request details
	log.Info("Opening editor",
		"path", absPath,
		"editor", cfg.DefaultEditor,
		"user", sshInfo.User,
		"host", sshInfo.Host,
		"server", cfg.Hosts.Server.Primary,
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
	fmt.Printf("Hosts:\n")
	fmt.Printf("  Server:\n")
	fmt.Printf("    Primary: %s\n", cfg.Hosts.Server.Primary)
	if cfg.Hosts.Server.Fallback != "" {
		fmt.Printf("    Fallback: %s\n", cfg.Hosts.Server.Fallback)
	}
	fmt.Printf("  SSH:\n")
	if cfg.Hosts.SSH.Host != "" {
		fmt.Printf("    Host: %s\n", cfg.Hosts.SSH.Host)
	} else {
		fmt.Printf("    Host: (auto-detect)\n")
	}
	fmt.Printf("    Auto-detect Tailscale: %v\n", cfg.Hosts.SSH.AutoDetect.Tailscale)
	if cfg.Hosts.SSH.AutoDetect.TailscalePattern != "" {
		fmt.Printf("    Tailscale Pattern: %s\n", cfg.Hosts.SSH.AutoDetect.TailscalePattern)
	}
	fmt.Printf("\nNetwork:\n")
	fmt.Printf("  Timeout: %v\n", cfg.Network.Timeout)
	fmt.Printf("  Retry Attempts: %d\n", cfg.Network.RetryAttempts)
	fmt.Printf("\nDefault Editor: %s\n", cfg.DefaultEditor)
	fmt.Printf("  (Editor definitions are fetched from the server. Use 'rcode editors' to see available editors.)\n")

	if len(cfg.FallbackEditors) > 0 {
		fmt.Printf("\nFallback Editors (used when server is unreachable):\n")
		for name, cmd := range cfg.FallbackEditors {
			fmt.Printf("  %s: %s\n", name, cmd)
		}
	}

	fmt.Printf("\nLogging:\n")
	fmt.Printf("  Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  File: %s\n", cfg.Logging.File)
}

// buildResolver creates a Resolver with appropriate sources based on config and flags.
func buildResolver(cfg *config.ClientConfig, hostFlag, sshClientIP string) *network.Resolver {
	sources := []network.HostSource{}

	// 1. Command-line flag (highest priority)
	if hostFlag != "" {
		sources = append(sources, &network.CommandLineSource{Host: hostFlag})
	}

	// 2. Environment variables
	sources = append(sources, &network.EnvSource{
		ServerHostEnv: "RCODE_SERVER_HOST",
		SSHHostEnv:    "RCODE_SSH_HOST",
		LegacyHostEnv: "RCODE_HOST",
	})

	// 3. Configuration file
	sources = append(sources, &network.ConfigSource{
		ServerPrimary:  cfg.Hosts.Server.Primary,
		ServerFallback: cfg.Hosts.Server.Fallback,
		SSHHost:        cfg.Hosts.SSH.Host,
	})

	// 4. Config fallback (separate source for lower priority)
	if cfg.Hosts.Server.Fallback != "" {
		sources = append(sources, &network.ConfigFallbackSource{
			ServerFallback: cfg.Hosts.Server.Fallback,
		})
	}

	// 5. Tailscale auto-detection
	if cfg.Hosts.SSH.AutoDetect.Tailscale {
		sources = append(sources, &network.TailscaleSource{
			Enabled:     true,
			HostPattern: cfg.Hosts.SSH.AutoDetect.TailscalePattern,
			ClientIP:    sshClientIP,
		})
	}

	// 6. SSH_CONNECTION environment
	sources = append(sources, &network.SSHConnectionSource{
		ClientIP: sshClientIP,
	})

	// 7. Hostname fallback (lowest priority)
	sources = append(sources, &network.HostnameSource{})

	return network.NewResolver(sources...)
}
