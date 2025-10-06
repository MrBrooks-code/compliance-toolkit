package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
)

const version = "1.0.0"

func main() {
	// Define CLI flags
	flags := pflag.NewFlagSet("compliance-client", pflag.ExitOnError)

	configFile := flags.StringP("config", "c", "", "Path to config file")
	reportName := flags.StringP("report", "r", "", "Single report to run (overrides config)")
	serverURL := flags.String("server", "", "Server URL (overrides config)")
	apiKey := flags.String("api-key", "", "API key (overrides config)")
	standaloneMode := flags.Bool("standalone", false, "Force standalone mode (no server)")
	onceMode := flags.Bool("once", false, "Run once and exit (ignore schedule)")
	showVersion := flags.BoolP("version", "v", false, "Show version and exit")
	generateConfig := flags.Bool("generate-config", false, "Generate default config file and exit")

	flags.Parse(os.Args[1:])

	// Handle version
	if *showVersion {
		fmt.Printf("Compliance Toolkit Client v%s\n", version)
		return
	}

	// Handle config generation
	if *generateConfig {
		if err := generateDefaultConfig(*configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to generate config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated default config file: %s\n", getConfigPath(*configFile))
		return
	}

	// Load configuration
	config, err := LoadClientConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI overrides
	if *serverURL != "" {
		config.Server.URL = *serverURL
	}
	if *apiKey != "" {
		config.Server.APIKey = *apiKey
	}
	if *standaloneMode {
		config.Server.URL = "" // Force standalone
	}
	if *reportName != "" {
		config.Reports.Reports = []string{*reportName}
	}
	if *onceMode {
		config.Schedule.Enabled = false
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logger := setupLogging(config.Logging)
	slog.SetDefault(logger)

	// Log startup
	slog.Info("Compliance Client starting",
		"version", version,
		"client_id", config.Client.ID,
		"hostname", config.Client.Hostname,
		"mode", getMode(config),
	)

	// Create and run client
	client := NewComplianceClient(config, logger)

	if err := client.Run(); err != nil {
		slog.Error("Client execution failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Compliance Client finished successfully")
}

// setupLogging creates a logger based on configuration
func setupLogging(cfg LoggingSettings) *slog.Logger {
	var handler slog.Handler
	var output *os.File

	// Determine output
	switch cfg.OutputPath {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output
		dir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: Could not create log directory: %v", err)
			output = os.Stdout
		} else {
			file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				log.Printf("Warning: Could not open log file: %v", err)
				output = os.Stdout
			} else {
				output = file
			}
		}
	}

	// Determine log level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create handler based on format
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	return slog.New(handler)
}

// getMode returns a human-readable mode string
func getMode(cfg *ClientConfig) string {
	if cfg.IsStandaloneMode() {
		return "standalone"
	}
	return fmt.Sprintf("server (%s)", cfg.Server.URL)
}

// generateDefaultConfig generates a default config file
func generateDefaultConfig(configPath string) error {
	path := getConfigPath(configPath)

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate YAML content
	content := `# Compliance Toolkit Client Configuration

# Client identification
client:
  id: ""                    # Auto-generated if empty
  hostname: ""              # Auto-detected if empty
  enabled: true

# Server configuration (leave url empty for standalone mode)
server:
  url: ""                   # e.g., "https://compliance.company.com:8443"
  api_key: ""               # API key for authentication
  tls_verify: true          # Verify TLS certificates
  timeout: 30s              # Request timeout
  retry_on_startup: true    # Retry cached submissions on startup

# Report configuration
reports:
  config_path: "configs/reports"
  output_path: "output/reports"
  save_local: true          # Save HTML reports locally
  reports:
    - "NIST_800_171_compliance.json"
    # - "FIPS_140_2_compliance.json"

# Scheduling (requires Windows Service or Task Scheduler)
schedule:
  enabled: false
  cron: "0 2 * * *"         # Daily at 2 AM (cron syntax)

# Retry configuration
retry:
  max_attempts: 3
  initial_backoff: 30s
  max_backoff: 5m
  backoff_multiplier: 2.0
  retry_on_server_error: true

# Local cache for offline operation
cache:
  enabled: true
  path: "cache/submissions"
  max_size_mb: 100
  max_age: 168h             # 7 days
  auto_clean: true

# Logging configuration
logging:
  level: "info"             # debug, info, warn, error
  format: "text"            # text, json
  output_path: "stdout"     # stdout, stderr, or file path
`

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath returns the effective config file path
func getConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	return "client.yaml"
}
