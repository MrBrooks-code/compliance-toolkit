package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"compliancetoolkit/pkg"

	"golang.org/x/sys/windows/registry"
)

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create reader with custom options
	reader := pkg.NewRegistryReader(
		pkg.WithLogger(logger),
		pkg.WithTimeout(10*time.Second),
	)

	ctx := context.Background()

	// Example 1: Read Windows product name with context
	productName, err := reader.ReadString(
		ctx,
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"ProductName",
	)
	if err != nil {
		if pkg.IsNotExist(err) {
			log.Printf("ProductName not found in registry")
		} else {
			log.Printf("Error reading ProductName: %v", err)
		}
	} else {
		fmt.Printf("Product Name: %s\n", productName)
	}

	// Example 2: Read build number with timeout context
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	buildNumber, err := reader.ReadInteger(
		ctxWithTimeout,
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"CurrentBuild",
	)
	if err != nil {
		log.Printf("Error reading CurrentBuild: %v", err)
	} else {
		fmt.Printf("Build Number: %d\n", buildNumber)
	}

	// Example 3: Batch read multiple values efficiently
	versionData, err := reader.BatchRead(
		ctx,
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		[]string{"ProductName", "CurrentBuild", "CurrentVersion", "EditionID"},
	)
	if err != nil {
		log.Printf("Error in batch read: %v", err)
	} else {
		fmt.Printf("\nWindows Version Info (batch read):\n")
		for k, v := range versionData {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	// Example 4: Load and execute operations from config
	config, err := pkg.LoadConfig("configs/registry_operations.json")
	if err != nil {
		log.Printf("Warning: Could not load config: %v", err)
		return
	}

	fmt.Printf("\nExecuting %d operations from config:\n", len(config.Queries))
	for _, query := range config.Queries {
		if query.Operation != "read" {
			continue // Skip write operations (not implemented)
		}

		rootKey, err := pkg.ParseRootKey(query.RootKey)
		if err != nil {
			log.Printf("Invalid root key %s: %v", query.RootKey, err)
			continue
		}

		if query.ReadAll {
			// Batch read all values in the key
			data, err := reader.BatchRead(ctx, rootKey, query.Path, []string{})
			if err != nil {
				log.Printf("[%s] Error: %v", query.Name, err)
			} else {
				fmt.Printf("[%s] %s:\n", query.Name, query.Description)
				for k, v := range data {
					fmt.Printf("  %s = %v\n", k, v)
				}
			}
		} else if query.ValueName != "" {
			// Read specific value
			value, err := reader.ReadString(ctx, rootKey, query.Path, query.ValueName)
			if err != nil {
				log.Printf("[%s] Error: %v", query.Name, err)
			} else {
				fmt.Printf("[%s] %s: %s\n", query.Name, query.Description, value)
			}
		}
	}
}
