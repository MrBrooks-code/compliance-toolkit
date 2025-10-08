package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"golang.org/x/crypto/bcrypt"
)

const version = "1.0.0"

func main() {
	// Define CLI flags
	flags := pflag.NewFlagSet("compliance-server", pflag.ExitOnError)

	configFile := flags.StringP("config", "c", "", "Path to config file")
	showVersion := flags.BoolP("version", "v", false, "Show version and exit")
	generateConfig := flags.Bool("generate-config", false, "Generate default config file and exit")
	hashAPIKey := flags.String("hash-api-key", "", "Generate bcrypt hash for an API key and exit")
	port := flags.IntP("port", "p", 0, "Server port (overrides config)")

	flags.Parse(os.Args[1:])

	// Handle version
	if *showVersion {
		fmt.Printf("Compliance Toolkit Server v%s\n", version)
		return
	}

	// Handle API key hashing
	if *hashAPIKey != "" {
		if err := hashAndPrintAPIKey(*hashAPIKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to hash API key: %v\n", err)
			os.Exit(1)
		}
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
	config, err := LoadServerConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI overrides
	if *port != 0 {
		config.Server.Port = *port
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
	slog.Info("Compliance Server starting",
		"version", version,
		"port", config.Server.Port,
		"tls_enabled", config.Server.TLS.Enabled,
	)

	// Create and start server
	server, err := NewComplianceServer(config, logger)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Start server in background
	if err := server.Start(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	slog.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown
	if err := server.Shutdown(); err != nil {
		slog.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Compliance Server stopped successfully")
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
		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Warning: Could not open log file: %v\n", err)
			output = os.Stdout
		} else {
			output = file
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

// getConfigPath returns the effective config file path
func getConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	return "server.yaml"
}

// hashAndPrintAPIKey generates a bcrypt hash for an API key
func hashAndPrintAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash API key: %w", err)
	}

	fmt.Println("API Key Hash Generation")
	fmt.Println("=======================")
	fmt.Printf("\nPlain API Key: %s\n", apiKey)
	fmt.Printf("Bcrypt Hash:   %s\n\n", string(hash))
	fmt.Println("Add this hash to your server.yaml:")
	fmt.Println("---")
	fmt.Println("auth:")
	fmt.Println("  enabled: true")
	fmt.Println("  require_key: true")
	fmt.Println("  use_hashed_keys: true")
	fmt.Println("  api_key_hashes:")
	fmt.Printf("    - \"%s\"\n", string(hash))
	fmt.Println("---")
	fmt.Println("\n⚠️  SECURITY WARNING:")
	fmt.Println("  - Store the plain API key securely for client use")
	fmt.Println("  - Only the hash should be in server.yaml")
	fmt.Println("  - The plain key cannot be recovered from the hash")

	return nil
}
