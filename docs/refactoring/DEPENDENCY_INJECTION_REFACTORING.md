# Dependency Injection Refactoring Plan

## Overview

This document outlines the comprehensive refactoring plan to implement proper dependency injection (DI) throughout the ComplianceToolkit codebase. This will enable better testability, modularity, and maintainability.

## Current State Analysis

### Current Issues

1. **Direct Instantiation Throughout**
   - `App` struct directly creates `RegistryReader`, `Menu`, `EvidenceLogger`, `HTMLReport`
   - Hard-coded dependencies make unit testing difficult
   - Tight coupling between components
   - No interface abstractions

2. **Global State & Logger**
   - `slog.SetDefault(logger)` creates global state in `toolkit.go:138`
   - Makes parallel testing impossible
   - Hidden dependency on global logger in some components

3. **Concrete Types Everywhere**
   - No interfaces defined for `RegistryReader`, `EvidenceLogger`, `HTMLReport`
   - Cannot mock dependencies for testing
   - Violates Dependency Inversion Principle

4. **Mixed Concerns**
   - `App` struct handles initialization, dependency creation, and business logic
   - `executeReport()` and `executeReportQuiet()` duplicate code
   - Configuration scattered across multiple locations

### Current Dependency Graph

```
main() (toolkit.go)
 ├─> App
 │    ├─> Menu (created in main)
 │    ├─> RegistryReader (created in App.init())
 │    ├─> AppConfig (created in main)
 │    └─> executeReport()
 │         ├─> LoadConfig() (static function)
 │         ├─> HTMLReport (created in executeReport)
 │         │    └─> RegistryReader (injected via SetRegistryReader)
 │         └─> EvidenceLogger (created in executeReport)
 │              └─> RegistryReader (passed to GatherMachineInfo)
 │
 └─> Global Logger (slog.SetDefault)
```

---

## Target Architecture

### Design Principles

1. **Dependency Inversion Principle**: Depend on abstractions, not concretions
2. **Interface Segregation**: Small, focused interfaces
3. **Single Responsibility**: Each component has one reason to change
4. **Constructor Injection**: All dependencies passed via constructors
5. **No Global State**: All dependencies explicit

### New Dependency Graph

```
main() (toolkit.go)
 └─> Dependencies Container
      ├─> Logger (slog.Logger)
      ├─> Config (AppConfig)
      ├─> RegistryService (interface)
      │    └─> RegistryReader (implementation)
      ├─> ReportService (interface)
      │    └─> ReportRunner (implementation)
      ├─> EvidenceService (interface)
      │    └─> EvidenceLogger (implementation)
      ├─> UIService (interface)
      │    └─> Menu (implementation)
      └─> App (receives all dependencies)
           └─> Business logic only
```

---

## Phase 1: Define Interfaces

### Step 1.1: Create Interface Definitions

**File**: `pkg/interfaces.go` (NEW)

```go
package pkg

import (
	"context"
	"golang.org/x/sys/windows/registry"
)

// RegistryService defines operations for reading Windows Registry
type RegistryService interface {
	ReadString(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)
	ReadInteger(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error)
	ReadBinary(ctx context.Context, rootKey registry.Key, path, valueName string) ([]byte, error)
	ReadStrings(ctx context.Context, rootKey registry.Key, path, valueName string) ([]string, error)
	ReadValue(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)
	BatchRead(ctx context.Context, rootKey registry.Key, path string, values []string) (map[string]interface{}, error)
}

// ReportService defines operations for generating compliance reports
type ReportService interface {
	Generate() error
	AddResult(name, description string, value interface{}, err error)
	AddResultWithDetails(name, description, rootKey, path, valueName, expectedValue string, value interface{}, err error)
	SetMetadata(metadata ReportMetadata)
	GetOutputPath() string
}

// EvidenceService defines operations for compliance evidence logging
type EvidenceService interface {
	GatherMachineInfo(reader RegistryService) error
	LogResult(checkName, description, regPath, valueName string, actualValue interface{}, err error)
	Finalize() error
	GetSummaryText() string
	GetLogPath() string
}

// UIService defines operations for user interaction
type UIService interface {
	ShowHeader()
	ShowMainMenu() int
	ShowReportMenuDynamic(reports []ReportInfo) int
	ShowError(message string)
	ShowSuccess(message string)
	ShowInfo(message string)
	ShowProgress(message string)
	Pause()
	GetIntInput() int
	GetStringInput() string
	Confirm(message string) bool
}

// ConfigService defines operations for loading configurations
type ConfigService interface {
	LoadConfig(path string) (*Config, error)
	ParseRootKey(rootKeyStr string) (registry.Key, error)
}

// FileService defines operations for file and directory management
type FileService interface {
	FindReportsDirectory(exeDir string) string
	ResolveDirectory(dir, exeDir string) string
	ListReports(reportsDir string) ([]ReportInfo, error)
	OpenBrowser(url string) error
	OpenFile(filePath string) error
}
```

**Benefits**:
- Clear contracts for each service
- Easy to mock for testing
- Enables interface-driven development
- Facilitates parallel development

---

### Step 1.2: Update Existing Structs to Implement Interfaces

**File**: `pkg/registryreader.go`

✅ **Already implements `RegistryService` interface** (no changes needed)

**File**: `pkg/htmlreport.go`

```go
// Ensure HTMLReport implements ReportService interface
type HTMLReport struct {
	Title          string
	Timestamp      time.Time
	Results        map[string]ReportResult
	OutputPath     string
	Metadata       ReportMetadata
	tmpl           *template.Template
	registryReader RegistryService // Changed from *RegistryReader
	logger         *slog.Logger    // Added for DI
}

// NewHTMLReport creates a new HTML report with dependencies
func NewHTMLReport(title, outputDir string, logger *slog.Logger, registryReader RegistryService) *HTMLReport {
	timestamp := time.Now()
	filename := fmt.Sprintf("%s_%s.html",
		sanitizeFilename(title),
		timestamp.Format("20060102_150405"))

	return &HTMLReport{
		Title:          title,
		Timestamp:      timestamp,
		Results:        make(map[string]ReportResult),
		OutputPath:     filepath.Join(outputDir, filename),
		logger:         logger,
		registryReader: registryReader,
	}
}

// SetMetadata sets report metadata
func (r *HTMLReport) SetMetadata(metadata ReportMetadata) {
	r.Metadata = metadata
}

// GetOutputPath returns the output path
func (r *HTMLReport) GetOutputPath() string {
	return r.OutputPath
}

// Remove SetRegistryReader() - no longer needed
```

**File**: `pkg/evidence.go`

```go
type EvidenceLogger struct {
	LogPath   string
	StartTime time.Time
	Evidence  *ComplianceEvidence
	logger    *slog.Logger // Added for DI
}

// NewEvidenceLogger creates a new evidence logger with dependencies
func NewEvidenceLogger(logDir, reportType string, logger *slog.Logger) (*EvidenceLogger, error) {
	// ... existing code ...

	return &EvidenceLogger{
		LogPath:   logPath,
		StartTime: timestamp,
		Evidence:  evidence,
		logger:    logger,
	}, nil
}

// GatherMachineInfo now accepts RegistryService interface
func (e *EvidenceLogger) GatherMachineInfo(reader RegistryService) error {
	// ... existing code (no changes to logic) ...
}

// GetLogPath returns the log path
func (e *EvidenceLogger) GetLogPath() string {
	return e.LogPath
}
```

**File**: `pkg/menu.go`

✅ **Already implements `UIService` interface** (no changes needed)

---

## Phase 2: Create Dependency Container

### Step 2.1: Define Dependencies Struct

**File**: `cmd/dependencies.go` (NEW)

```go
package main

import (
	"log/slog"
	"time"

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

// AppConfig holds application configuration
type AppConfig struct {
	Timeout     time.Duration
	LogLevel    slog.Level
	OutputDir   string
	LogsDir     string
	EvidenceDir string
	ReportsDir  string
	ExeDir      string
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
```

---

### Step 2.2: Create Factory for Services

**File**: `cmd/factory.go` (NEW)

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"compliancetoolkit/pkg"
)

// ServiceFactory creates service instances with proper dependencies
type ServiceFactory struct {
	deps *Dependencies
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(deps *Dependencies) *ServiceFactory {
	return &ServiceFactory{deps: deps}
}

// CreateReportService creates a new report service with all dependencies
func (f *ServiceFactory) CreateReportService(title, outputDir string) pkg.ReportService {
	return pkg.NewHTMLReport(
		title,
		outputDir,
		f.deps.Logger,
		f.deps.RegistryService,
	)
}

// CreateEvidenceService creates a new evidence service with all dependencies
func (f *ServiceFactory) CreateEvidenceService(evidenceDir, reportType string) (pkg.EvidenceService, error) {
	return pkg.NewEvidenceLogger(
		evidenceDir,
		reportType,
		f.deps.Logger,
	)
}

// CreateReportRunner creates a report runner with dependencies
func (f *ServiceFactory) CreateReportRunner() *ReportRunner {
	return NewReportRunner(f.deps)
}
```

---

## Phase 3: Refactor Application Entry Point

### Step 3.1: Update App Struct

**File**: `cmd/toolkit.go`

```go
type App struct {
	deps    *Dependencies
	factory *ServiceFactory
}

// NewApp creates a new application with dependencies
func NewApp(deps *Dependencies) (*App, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dependencies: %w", err)
	}

	return &App{
		deps:    deps,
		factory: NewServiceFactory(deps),
	}, nil
}

// Remove: menu, reader, config fields - now accessed via deps
// Remove: init() method - dependencies injected instead
```

---

### Step 3.2: Update main() Function

**File**: `cmd/toolkit.go`

```go
func main() {
	// Parse flags
	reportName := flag.String("report", "", "Report to run")
	listReports := flag.Bool("list", false, "List available reports")
	quiet := flag.Bool("quiet", false, "Suppress non-essential output")
	outputDir := flag.String("output", "output/reports", "Output directory")
	logsDir := flag.String("logs", "output/logs", "Logs directory")
	evidenceDir := flag.String("evidence", "output/evidence", "Evidence directory")
	timeout := flag.Duration("timeout", 10*time.Second, "Registry timeout")
	flag.Parse()

	// Determine executable directory
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to determine executable path: %v\n", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)

	// Create configuration
	config := &AppConfig{
		Timeout:     *timeout,
		LogLevel:    slog.LevelInfo,
		OutputDir:   *outputDir,
		LogsDir:     *logsDir,
		EvidenceDir: *evidenceDir,
		ExeDir:      exeDir,
	}

	// Setup logger
	logger, logFile, err := setupLogger(config.LogsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create log file: %v\n", err)
		logger = slog.Default()
	}
	defer logFile.Close()

	// Create directories
	os.MkdirAll(config.OutputDir, 0755)
	os.MkdirAll(config.LogsDir, 0755)
	os.MkdirAll(config.EvidenceDir, 0755)

	// Find reports directory
	fileService := pkg.NewFileService()
	config.ReportsDir = fileService.FindReportsDirectory(exeDir)

	// Create dependencies
	deps := NewDependencies(config, logger)

	// Create application
	app, err := NewApp(deps)
	if err != nil {
		logger.Error("Failed to create application", "error", err)
		os.Exit(1)
	}

	// Handle CLI mode
	if *listReports {
		app.listReportsCLI()
		return
	}

	if *reportName != "" {
		success := app.runReportCLI(*reportName, *quiet)
		if !success {
			os.Exit(1)
		}
		return
	}

	// Interactive mode
	app.runInteractive()
}

// setupLogger creates and configures the logger
func setupLogger(logsDir string) (*slog.Logger, *os.File, error) {
	logFile := filepath.Join(logsDir, fmt.Sprintf("toolkit_%s.log",
		time.Now().Format("20060102_150405")))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return logger, file, nil
}

// runInteractive runs the interactive menu
func (app *App) runInteractive() {
	for {
		choice := app.deps.UIService.ShowMainMenu()

		switch choice {
		case 1:
			app.runReports()
		case 2:
			app.viewHTMLReports()
		case 3:
			app.viewEvidenceLogs()
		case 4:
			app.viewLogFiles()
		case 5:
			app.configuration()
		case 6:
			app.deps.UIService.ShowAbout()
		case 0:
			app.exit()
			return
		default:
			app.deps.UIService.ShowError("Invalid option. Please try again.")
		}
	}
}
```

---

## Phase 4: Extract Report Runner

### Step 4.1: Create ReportRunner Service

**File**: `cmd/report_runner.go` (REFACTOR EXISTING)

```go
package main

import (
	"context"
	"fmt"
	"path/filepath"

	"compliancetoolkit/pkg"
)

// ReportRunner handles report execution with proper dependency injection
type ReportRunner struct {
	deps    *Dependencies
	factory *ServiceFactory
}

// NewReportRunner creates a new report runner
func NewReportRunner(deps *Dependencies) *ReportRunner {
	return &ReportRunner{
		deps:    deps,
		factory: NewServiceFactory(deps),
	}
}

// ExecuteReport runs a single report
func (rr *ReportRunner) ExecuteReport(configFile string, quiet bool) error {
	configPath := filepath.Join(rr.deps.Config.ReportsDir, configFile)

	// Load config
	config, err := rr.deps.ConfigService.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create report service
	reportName := config.Metadata.ReportTitle
	if reportName == "" {
		reportName = configFile
	}

	reportService := rr.factory.CreateReportService(reportName, rr.deps.Config.OutputDir)
	reportService.SetMetadata(config.Metadata)

	// Create evidence service
	reportType := filepath.Base(configFile)[:len(configFile)-5]
	evidenceService, err := rr.factory.CreateEvidenceService(rr.deps.Config.EvidenceDir, reportType)
	if err != nil {
		rr.deps.Logger.Warn("Could not create evidence log", "error", err)
	} else {
		if !quiet {
			fmt.Println("  📋  Gathering machine information for audit trail...")
		}
		if err := evidenceService.GatherMachineInfo(rr.deps.RegistryService); err != nil {
			rr.deps.Logger.Warn("Could not gather machine info", "error", err)
		}
	}

	// Execute queries
	ctx := context.Background()
	successCount := 0
	errorCount := 0

	for _, query := range config.Queries {
		if query.Operation != "read" {
			continue
		}

		rootKey, err := rr.deps.ConfigService.ParseRootKey(query.RootKey)
		if err != nil {
			if !quiet {
				fmt.Printf("  ⚠️  [%s] Invalid root key: %s\n", query.Name, query.RootKey)
			}
			reportService.AddResult(query.Name, query.Description, nil, err)
			if evidenceService != nil {
				evidenceService.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		// Execute query (batch or single)
		if query.ReadAll {
			data, err := rr.deps.RegistryService.BatchRead(ctx, rootKey, query.Path, []string{})
			if err != nil {
				handleError(query, err, quiet, reportService, evidenceService)
				errorCount++
			} else {
				handleSuccess(query, data, quiet, reportService, evidenceService)
				successCount++
			}
		} else {
			value, err := rr.deps.RegistryService.ReadValue(ctx, rootKey, query.Path, query.ValueName)
			if err != nil {
				handleDetailedError(query, err, quiet, reportService, evidenceService)
				errorCount++
			} else {
				handleDetailedSuccess(query, value, quiet, reportService, evidenceService)
				successCount++
			}
		}
	}

	// Generate report
	if err := reportService.Generate(); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Finalize evidence
	if evidenceService != nil {
		if err := evidenceService.Finalize(); err != nil {
			rr.deps.Logger.Warn("Could not finalize evidence", "error", err)
		} else if !quiet {
			fmt.Println()
			fmt.Println(evidenceService.GetSummaryText())
		}
	}

	if !quiet {
		fmt.Printf("\n  📊  Results: %d successful, %d errors\n", successCount, errorCount)
		fmt.Printf("  📄  HTML Report: %s\n", reportService.GetOutputPath())
		if evidenceService != nil {
			fmt.Printf("  📋  Evidence Log: %s\n", evidenceService.GetLogPath())
		}
	}

	// Log summary
	rr.deps.Logger.Info("Report execution completed",
		"report", reportName,
		"success_count", successCount,
		"error_count", errorCount,
		"html_report", reportService.GetOutputPath(),
	)

	return nil
}

// Helper functions
func handleError(query pkg.Query, err error, quiet bool, reportService pkg.ReportService, evidenceService pkg.EvidenceService) {
	if !quiet && !pkg.IsNotExist(err) {
		fmt.Printf("  ❌  [%s] Error: %v\n", query.Name, err)
	}
	reportService.AddResult(query.Name, query.Description, nil, err)
	if evidenceService != nil {
		evidenceService.LogResult(query.Name, query.Description, query.Path, "", nil, err)
	}
}

func handleSuccess(query pkg.Query, data map[string]interface{}, quiet bool, reportService pkg.ReportService, evidenceService pkg.EvidenceService) {
	if !quiet {
		fmt.Printf("  ✅  [%s] Read %d values\n", query.Name, len(data))
	}
	reportService.AddResult(query.Name, query.Description, data, nil)
	if evidenceService != nil {
		evidenceService.LogResult(query.Name, query.Description, query.Path, "", data, nil)
	}
}

func handleDetailedError(query pkg.Query, err error, quiet bool, reportService pkg.ReportService, evidenceService pkg.EvidenceService) {
	if !quiet && !pkg.IsNotExist(err) {
		fmt.Printf("  ❌  [%s] Error: %v\n", query.Name, err)
	}
	reportService.AddResultWithDetails(
		query.Name, query.Description,
		query.RootKey, query.Path, query.ValueName, query.ExpectedValue,
		nil, err,
	)
	if evidenceService != nil {
		evidenceService.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
	}
}

func handleDetailedSuccess(query pkg.Query, value string, quiet bool, reportService pkg.ReportService, evidenceService pkg.EvidenceService) {
	if !quiet {
		fmt.Printf("  ✅  [%s] Success\n", query.Name)
	}
	reportService.AddResultWithDetails(
		query.Name, query.Description,
		query.RootKey, query.Path, query.ValueName, query.ExpectedValue,
		value, nil,
	)
	if evidenceService != nil {
		evidenceService.LogResult(query.Name, query.Description, query.Path, query.ValueName, value, nil)
	}
}
```

---

## Phase 5: Create Missing Services

### Step 5.1: ConfigService Implementation

**File**: `pkg/config_service.go` (NEW)

```go
package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ConfigServiceImpl implements ConfigService interface
type ConfigServiceImpl struct{}

// NewConfigService creates a new config service
func NewConfigService() ConfigService {
	return &ConfigServiceImpl{}
}

// LoadConfig loads a configuration from file
func (cs *ConfigServiceImpl) LoadConfig(path string) (*Config, error) {
	return LoadConfig(path) // Delegate to existing function
}

// ParseRootKey parses a root key string
func (cs *ConfigServiceImpl) ParseRootKey(rootKeyStr string) (registry.Key, error) {
	return ParseRootKey(rootKeyStr) // Delegate to existing function
}
```

---

### Step 5.2: FileService Implementation

**File**: `pkg/file_service.go` (NEW)

```go
package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FileServiceImpl implements FileService interface
type FileServiceImpl struct{}

// NewFileService creates a new file service
func NewFileService() FileService {
	return &FileServiceImpl{}
}

// FindReportsDirectory looks for the reports directory in multiple locations
func (fs *FileServiceImpl) FindReportsDirectory(exeDir string) string {
	locations := []string{
		"configs/reports",
		filepath.Join(exeDir, "configs/reports"),
		filepath.Join(exeDir, "..", "configs/reports"),
	}

	for _, loc := range locations {
		absPath, err := filepath.Abs(loc)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	return "configs/reports"
}

// ResolveDirectory converts relative paths to absolute paths
func (fs *FileServiceImpl) ResolveDirectory(dir, exeDir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}

	if _, err := os.Stat(dir); err == nil {
		absPath, _ := filepath.Abs(dir)
		return absPath
	}

	return filepath.Join(exeDir, dir)
}

// ListReports lists all available reports
func (fs *FileServiceImpl) ListReports(reportsDir string) ([]ReportInfo, error) {
	files, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read reports directory: %w", err)
	}

	var reports []ReportInfo
	configService := NewConfigService()

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		configPath := filepath.Join(reportsDir, file.Name())
		config, err := configService.LoadConfig(configPath)
		if err != nil {
			continue
		}

		title := config.Metadata.ReportTitle
		if title == "" {
			title = file.Name()
		}

		reports = append(reports, ReportInfo{
			Title:      title,
			ConfigFile: file.Name(),
			Category:   config.Metadata.Category,
			Version:    config.Metadata.ReportVersion,
		})
	}

	return reports, nil
}

// OpenBrowser opens a URL in the default browser
func (fs *FileServiceImpl) OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// OpenFile opens a file with the default program
func (fs *FileServiceImpl) OpenFile(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default:
		cmd = exec.Command("xdg-open", filePath)
	}

	return cmd.Start()
}
```

---

## Phase 6: Testing Strategy

### Step 6.1: Create Mock Implementations

**File**: `pkg/mocks/registry_service_mock.go` (NEW)

```go
package mocks

import (
	"context"

	"golang.org/x/sys/windows/registry"
)

// MockRegistryService is a mock implementation of RegistryService
type MockRegistryService struct {
	ReadStringFunc    func(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)
	ReadIntegerFunc   func(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error)
	ReadBinaryFunc    func(ctx context.Context, rootKey registry.Key, path, valueName string) ([]byte, error)
	ReadStringsFunc   func(ctx context.Context, rootKey registry.Key, path, valueName string) ([]string, error)
	ReadValueFunc     func(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)
	BatchReadFunc     func(ctx context.Context, rootKey registry.Key, path string, values []string) (map[string]interface{}, error)
}

func (m *MockRegistryService) ReadString(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error) {
	if m.ReadStringFunc != nil {
		return m.ReadStringFunc(ctx, rootKey, path, valueName)
	}
	return "", nil
}

func (m *MockRegistryService) ReadInteger(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error) {
	if m.ReadIntegerFunc != nil {
		return m.ReadIntegerFunc(ctx, rootKey, path, valueName)
	}
	return 0, nil
}

// ... implement other methods ...
```

---

### Step 6.2: Unit Test Examples

**File**: `cmd/report_runner_test.go` (NEW)

```go
package main

import (
	"context"
	"testing"

	"compliancetoolkit/pkg"
	"compliancetoolkit/pkg/mocks"

	"golang.org/x/sys/windows/registry"
)

func TestReportRunner_ExecuteReport(t *testing.T) {
	// Create mock services
	mockRegistry := &mocks.MockRegistryService{
		ReadValueFunc: func(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error) {
			return "1", nil // Simulate UAC enabled
		},
	}

	mockUI := &mocks.MockUIService{}
	mockConfig := &mocks.MockConfigService{}
	mockFile := &mocks.MockFileService{}

	// Create test dependencies
	deps := &Dependencies{
		Logger:          slog.Default(),
		Config:          &AppConfig{
			OutputDir:   t.TempDir(),
			EvidenceDir: t.TempDir(),
			ReportsDir:  "testdata",
		},
		RegistryService: mockRegistry,
		UIService:       mockUI,
		ConfigService:   mockConfig,
		FileService:     mockFile,
	}

	// Create report runner
	runner := NewReportRunner(deps)

	// Execute report
	err := runner.ExecuteReport("test_config.json", true)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
```

---

## Phase 7: Migration Plan

### Execution Order

1. **Week 1: Interfaces & Mocks**
   - [ ] Create `pkg/interfaces.go`
   - [ ] Create `pkg/mocks/` directory with mock implementations
   - [ ] Update existing structs to match interfaces
   - [ ] Write interface compliance tests

2. **Week 2: Services**
   - [ ] Create `pkg/config_service.go`
   - [ ] Create `pkg/file_service.go`
   - [ ] Update `pkg/htmlreport.go` for DI
   - [ ] Update `pkg/evidence.go` for DI
   - [ ] Write unit tests for services

3. **Week 3: Dependency Container**
   - [ ] Create `cmd/dependencies.go`
   - [ ] Create `cmd/factory.go`
   - [ ] Update `cmd/toolkit.go` main()
   - [ ] Remove global logger usage

4. **Week 4: Report Runner**
   - [ ] Refactor `cmd/report_runner.go`
   - [ ] Extract helper functions
   - [ ] Write comprehensive tests
   - [ ] Remove duplicate code

5. **Week 5: Integration**
   - [ ] Update all `App` methods to use dependencies
   - [ ] Remove old initialization code
   - [ ] Integration testing
   - [ ] End-to-end testing

6. **Week 6: Polish & Documentation**
   - [ ] Update CLAUDE.md
   - [ ] Update ARCHITECTURE.md
   - [ ] Add code examples to docs
   - [ ] Performance testing
   - [ ] Final review

---

## Testing Checklist

### Unit Tests
- [ ] Test each service with mock dependencies
- [ ] Test factory creates correct instances
- [ ] Test dependency validation
- [ ] Test error handling

### Integration Tests
- [ ] Test report runner with real registry (Windows only)
- [ ] Test full report generation flow
- [ ] Test CLI mode
- [ ] Test interactive mode

### Regression Tests
- [ ] Verify all existing reports still work
- [ ] Verify HTML output unchanged
- [ ] Verify evidence logs unchanged
- [ ] Verify CLI flags work

---

## Rollback Plan

If issues arise:

1. **Git Tags**: Tag current working version before refactoring
2. **Feature Branches**: Develop on `refactor/dependency-injection` branch
3. **Incremental Merges**: Merge phases incrementally to main
4. **Rollback Commands**:
   ```bash
   # Rollback to last working version
   git checkout <tag-before-refactor>
   git checkout -b rollback-safe
   ```

---

## Benefits After Refactoring

### Testability
- ✅ Can test report generation without registry access
- ✅ Can mock UI for CLI testing
- ✅ Can test business logic in isolation
- ✅ Parallel test execution possible

### Maintainability
- ✅ Clear separation of concerns
- ✅ Easy to add new report types
- ✅ Easy to swap implementations
- ✅ Dependency changes isolated

### Performance
- ✅ Lazy initialization possible
- ✅ Dependency caching
- ✅ Resource pooling enabled
- ✅ Concurrent report generation easier

### Code Quality
- ✅ Reduced coupling
- ✅ Increased cohesion
- ✅ SOLID principles followed
- ✅ Interface-driven design

---

## Code Metrics (Before/After)

| Metric | Before | After (Target) |
|--------|--------|----------------|
| Lines of Code (cmd/) | ~1000 | ~800 |
| Cyclomatic Complexity | 45 | 25 |
| Test Coverage | 15% | 75% |
| Number of Interfaces | 0 | 6 |
| Global State Dependencies | 1 (logger) | 0 |
| Mock-able Components | 0% | 100% |

---

## Questions for Discussion

1. **Performance**: Is the overhead of interfaces acceptable? (Answer: Yes, negligible in Go)
2. **Complexity**: Does DI add too much complexity? (Answer: No, reduces coupling complexity)
3. **Learning Curve**: Will new developers understand DI? (Answer: Yes, with good docs)
4. **Migration Risk**: Can we do this incrementally? (Answer: Yes, phase-by-phase)

---

## Conclusion

This refactoring will transform the codebase from a tightly-coupled, hard-to-test application into a well-architected, testable, and maintainable system. The 6-week timeline allows for careful implementation with comprehensive testing at each phase.

**Recommendation**: Proceed with Phase 1 to validate the interface design before committing to full refactoring.
