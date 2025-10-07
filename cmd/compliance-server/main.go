package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
)

const version = "1.0.0"

func main() {
	// Define CLI flags
	flags := pflag.NewFlagSet("compliance-server", pflag.ExitOnError)

	configFile := flags.StringP("config", "c", "", "Path to config file")
	showVersion := flags.BoolP("version", "v", false, "Show version and exit")
	generateConfig := flags.Bool("generate-config", false, "Generate default config file and exit")
	port := flags.IntP("port", "p", 0, "Server port (overrides config)")

	flags.Parse(os.Args[1:])

	// Handle version
	if *showVersion {
		fmt.Printf("Compliance Toolkit Server v%s\n", version)
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
