package main

import (
	"fmt"
	"log/slog"

	"compliancetoolkit/pkg"
)

// Dependencies holds all application dependencies
type Dependencies struct {
	Logger          *slog.Logger
	Config          *AppConfig
	RegistryService pkg.RegistryService
	UIService       pkg.UIService
	ConfigService   pkg.ConfigService
	FileService     pkg.FileService
}

// NewDependencies creates and wires all application dependencies
func NewDependencies(config *AppConfig, logger *slog.Logger) *Dependencies {
	return &Dependencies{
		Logger:          logger,
		Config:          config,
		RegistryService: pkg.NewRegistryReader(
			pkg.WithLogger(logger),
			pkg.WithTimeout(config.Timeout),
		),
		UIService:     pkg.NewMenu(),
		ConfigService: pkg.NewConfigService(),
		FileService:   pkg.NewFileService(),
	}
}

// Validate ensures all dependencies are properly initialized
func (d *Dependencies) Validate() error {
	if d.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if d.Config == nil {
		return fmt.Errorf("config is required")
	}
	if d.RegistryService == nil {
		return fmt.Errorf("registry service is required")
	}
	if d.UIService == nil {
		return fmt.Errorf("UI service is required")
	}
	if d.ConfigService == nil {
		return fmt.Errorf("config service is required")
	}
	if d.FileService == nil {
		return fmt.Errorf("file service is required")
	}
	return nil
}

// Clone creates a copy of dependencies with different config (useful for testing)
func (d *Dependencies) Clone(config *AppConfig) *Dependencies {
	return &Dependencies{
		Logger:          d.Logger,
		Config:          config,
		RegistryService: d.RegistryService,
		UIService:       d.UIService,
		ConfigService:   d.ConfigService,
		FileService:     d.FileService,
	}
}
