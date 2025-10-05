package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"compliancetoolkit/pkg"
)

func main() {
	// Command-line flags
	configPath := flag.String("config", "", "Path to specific config file")
	reportDir := flag.String("reports", "configs/reports", "Directory containing report configs")
	outputJSON := flag.Bool("json", false, "Output results as JSON")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Set up logging
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Create reader
	reader := pkg.NewRegistryReader(
		pkg.WithLogger(logger),
		pkg.WithTimeout(10*time.Second),
	)

	ctx := context.Background()

	// Determine which configs to run
	var configFiles []string
	if *configPath != "" {
		// Run single config
		configFiles = []string{*configPath}
	} else {
		// Run all configs in directory
		matches, err := filepath.Glob(filepath.Join(*reportDir, "*.json"))
		if err != nil {
			log.Fatalf("Failed to list config files: %v", err)
		}
		configFiles = matches
	}

	if len(configFiles) == 0 {
		log.Fatal("No config files found")
	}

	// Results storage
	allResults := make(map[string]map[string]interface{})

	// Execute each config
	for _, configFile := range configFiles {
		reportName := filepath.Base(configFile)
		fmt.Printf("\n========================================\n")
		fmt.Printf("Running Report: %s\n", reportName)
		fmt.Printf("========================================\n\n")

		// Load config
		config, err := pkg.LoadConfig(configFile)
		if err != nil {
			log.Printf("ERROR: Failed to load config %s: %v\n", configFile, err)
			continue
		}

		reportResults := make(map[string]interface{})

		// Execute each query
		for _, query := range config.Queries {
			if query.Operation != "read" {
				log.Printf("SKIP [%s]: Write operations not supported\n", query.Name)
				continue
			}

			rootKey, err := pkg.ParseRootKey(query.RootKey)
			if err != nil {
				log.Printf("ERROR [%s]: Invalid root key %s\n", query.Name, query.RootKey)
				continue
			}

			if query.ReadAll {
				// Batch read all values
				data, err := reader.BatchRead(ctx, rootKey, query.Path, []string{})
				if err != nil {
					if pkg.IsNotExist(err) {
						if !*outputJSON {
							fmt.Printf("❌ [%s] Key not found: %s\n", query.Name, query.Path)
						}
						reportResults[query.Name] = map[string]interface{}{
							"error":       "not found",
							"description": query.Description,
						}
					} else {
						if !*outputJSON {
							fmt.Printf("❌ [%s] Error: %v\n", query.Name, err)
						}
						reportResults[query.Name] = map[string]interface{}{
							"error":       err.Error(),
							"description": query.Description,
						}
					}
					continue
				}

				if !*outputJSON {
					fmt.Printf("✅ [%s] %s:\n", query.Name, query.Description)
					for k, v := range data {
						fmt.Printf("   %s = %v\n", k, v)
					}
				}
				reportResults[query.Name] = map[string]interface{}{
					"description": query.Description,
					"values":      data,
				}

			} else {
				// Read specific value
				value, err := reader.ReadString(ctx, rootKey, query.Path, query.ValueName)
				if err != nil {
					if pkg.IsNotExist(err) {
						if !*outputJSON {
							fmt.Printf("❌ [%s] Not found: %s\\%s\n", query.Name, query.Path, query.ValueName)
						}
						reportResults[query.Name] = map[string]interface{}{
							"error":       "not found",
							"description": query.Description,
						}
					} else {
						if !*outputJSON {
							fmt.Printf("❌ [%s] Error: %v\n", query.Name, err)
						}
						reportResults[query.Name] = map[string]interface{}{
							"error":       err.Error(),
							"description": query.Description,
						}
					}
					continue
				}

				if !*outputJSON {
					fmt.Printf("✅ [%s] %s: %s\n", query.Name, query.Description, value)
				}
				reportResults[query.Name] = map[string]interface{}{
					"description": query.Description,
					"value":       value,
				}
			}
		}

		allResults[reportName] = reportResults
	}

	// Output JSON if requested
	if *outputJSON {
		output := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"reports":   allResults,
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("Report execution complete\n")
	fmt.Printf("========================================\n")
}
