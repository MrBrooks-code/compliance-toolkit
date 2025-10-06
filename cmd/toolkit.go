package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"compliancetoolkit/pkg"
	"github.com/spf13/pflag"
)

type App struct {
	menu        *pkg.Menu
	reader      *pkg.RegistryReader
	auditLogger *pkg.AuditLogger
	config      *pkg.Config
	outputDir   string
	logsDir     string
	evidenceDir string
	reportsDir  string
	exeDir      string
}

func main() {
	// Define CLI flags using pflag for better Viper integration
	flags := pflag.NewFlagSet("compliancetoolkit", pflag.ExitOnError)

	// Report execution flags
	reportName := flags.StringP("report", "r", "", "Report to run (e.g., 'NIST_800_171_compliance.json' or 'all')")
	listReports := flags.BoolP("list", "l", false, "List available reports and exit")
	quiet := flags.BoolP("quiet", "q", false, "Suppress non-essential output (for scheduled runs)")

	// Configuration file flag
	configFile := flags.StringP("config", "c", "", "Path to config file (default: ./config.yaml)")

	// Override flags (take precedence over config file)
	outputDir := flags.String("output", "", "Output directory for reports (overrides config)")
	logsDir := flags.String("logs", "", "Logs directory (overrides config)")
	evidenceDir := flags.String("evidence", "", "Evidence logs directory (overrides config)")
	timeout := flags.Duration("timeout", 0, "Registry operation timeout (overrides config)")
	logLevel := flags.String("log-level", "", "Log level: debug, info, warn, error (overrides config)")

	// Generate default config flag
	genConfig := flags.Bool("generate-config", false, "Generate default config.yaml file and exit")

	flags.Parse(os.Args[1:])

	// Handle config generation
	if *genConfig {
		configPath := "config/config.yaml"
		if *configFile != "" {
			configPath = *configFile
		}
		if err := pkg.SaveDefaultConfig(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to generate config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated default config file: %s\n", configPath)
		fmt.Println("Edit this file to customize your configuration.")
		return
	}

	// Load configuration (YAML -> ENV -> Flags precedence)
	cfg, err := pkg.LoadConfig(*configFile, flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Determine executable directory
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to determine executable path: %v\n", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)

	// Apply CLI flag overrides to config (with validation)
	if *outputDir != "" {
		// Validate directory path
		if err := pkg.ValidateFilePath(*outputDir, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid output directory: %v\n", err)
			os.Exit(1)
		}
		cfg.Reports.OutputPath = *outputDir
	}
	if *logsDir != "" {
		if err := pkg.ValidateFilePath(*logsDir, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid logs directory: %v\n", err)
			os.Exit(1)
		}
		cfg.Logging.OutputPath = *logsDir
	}
	if *evidenceDir != "" {
		if err := pkg.ValidateFilePath(*evidenceDir, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid evidence directory: %v\n", err)
			os.Exit(1)
		}
		cfg.Reports.EvidencePath = *evidenceDir
	}
	if *timeout > 0 {
		// Validate timeout is reasonable (1s to 5 minutes)
		if *timeout < time.Second || *timeout > 5*time.Minute {
			fmt.Fprintf(os.Stderr, "Error: Timeout must be between 1s and 5m (got %v)\n", *timeout)
			os.Exit(1)
		}
		cfg.Server.ReadTimeout = *timeout
	}
	if *logLevel != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[strings.ToLower(*logLevel)] {
			fmt.Fprintf(os.Stderr, "Error: Invalid log level '%s' (must be debug, info, warn, or error)\n", *logLevel)
			os.Exit(1)
		}
		cfg.Logging.Level = *logLevel
	}

	app := &App{
		menu:        pkg.NewMenu(),
		outputDir:   cfg.Reports.OutputPath,
		logsDir:     cfg.Logging.OutputPath,
		evidenceDir: cfg.Reports.EvidencePath,
		exeDir:      exeDir,
		config:      cfg,
	}

	// Initialize
	app.init()

	// Handle CLI mode (non-interactive)
	if *listReports {
		app.listReportsCLI()
		return
	}

	if *reportName != "" {
		// Run specific report or all reports
		success := app.runReportCLI(*reportName, *quiet)
		if !success {
			os.Exit(1)
		}
		return
	}

	// Interactive mode - Main loop
	for {
		choice := app.menu.ShowMainMenu()

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
			app.menu.ShowAbout()
		case 0:
			app.exit()
			return
		default:
			app.menu.ShowError("Invalid option. Please try again.")
		}
	}
}

func (app *App) init() {
	// Find reports directory (try multiple locations)
	app.reportsDir = app.findReportsDirectory()

	// Create output directories (use absolute paths if needed)
	app.outputDir = app.resolveDirectory(app.outputDir)
	app.logsDir = app.resolveDirectory(app.logsDir)
	app.evidenceDir = app.resolveDirectory(app.evidenceDir)

	os.MkdirAll(app.outputDir, 0755)
	os.MkdirAll(app.logsDir, 0755)
	os.MkdirAll(app.evidenceDir, 0755)

	// Create audit log directory if audit mode is enabled
	if app.config.Security.AuditMode {
		auditDir := app.resolveDirectory(app.config.Security.AuditLogPath)
		os.MkdirAll(auditDir, 0755)
	}

	// Set up logging based on config
	var logger *slog.Logger

	// Determine log output
	if app.config.Logging.EnableFileLogging {
		logFile := filepath.Join(app.logsDir, fmt.Sprintf("toolkit_%s.log",
			time.Now().Format("20060102_150405")))

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("Warning: Could not create log file: %v", err)
			logger = createLogger(os.Stdout, app.config.Logging)
		} else {
			logger = createLogger(file, app.config.Logging)
		}
	} else {
		// Log to stdout/stderr
		if strings.ToLower(app.config.Logging.OutputPath) == "stderr" {
			logger = createLogger(os.Stderr, app.config.Logging)
		} else {
			logger = createLogger(os.Stdout, app.config.Logging)
		}
	}

	slog.SetDefault(logger)

	// Initialize audit logger if enabled
	var auditLogger *pkg.AuditLogger
	if app.config.Security.AuditMode {
		// Create audit log file path with timestamp
		auditLogPath := filepath.Join(app.config.Security.AuditLogPath,
			fmt.Sprintf("audit_%s.log", time.Now().Format("20060102_150405")))

		var err error
		auditLogger, err = pkg.NewAuditLoggerWithFile(auditLogPath, true)
		if err != nil {
			log.Printf("Warning: Failed to create audit logger: %v", err)
			// Continue without audit logging
			auditLogger = nil
		} else {
			slog.Info("Audit logging enabled", "path", auditLogPath)
		}
	}
	app.auditLogger = auditLogger

	// Initialize registry reader with config values
	readerOpts := []pkg.RegistryReaderOption{
		pkg.WithLogger(logger),
		pkg.WithTimeout(app.config.Server.ReadTimeout),
	}
	if auditLogger != nil {
		readerOpts = append(readerOpts, pkg.WithAuditLogger(auditLogger))
	}
	app.reader = pkg.NewRegistryReader(readerOpts...)
}

// createLogger creates a structured logger based on config
func createLogger(output *os.File, cfg pkg.LoggingConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}

	if cfg.IsJSONFormat() {
		return slog.New(slog.NewJSONHandler(output, opts))
	}
	return slog.New(slog.NewTextHandler(output, opts))
}

// findReportsDirectory looks for the reports directory in multiple locations
func (app *App) findReportsDirectory() string {
	// Try these locations in order:
	locations := []string{
		"configs/reports",                              // 1. Current working directory
		filepath.Join(app.exeDir, "configs/reports"),   // 2. Next to executable
		filepath.Join(app.exeDir, "..", "configs/reports"), // 3. One level up from exe
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

	// Default fallback
	return "configs/reports"
}

// resolveDirectory converts relative paths to absolute paths based on exe location
func (app *App) resolveDirectory(dir string) string {
	// If already absolute, return as-is
	if filepath.IsAbs(dir) {
		return dir
	}

	// Try current working directory first
	if _, err := os.Stat(dir); err == nil {
		absPath, _ := filepath.Abs(dir)
		return absPath
	}

	// Otherwise use path relative to executable
	return filepath.Join(app.exeDir, dir)
}

func (app *App) runReports() {
	// Dynamically load available reports from configs directory
	reports, err := app.loadAvailableReports()
	if err != nil {
		app.menu.ShowError(fmt.Sprintf("Failed to load reports: %v", err))
		app.menu.Pause()
		return
	}

	if len(reports) == 0 {
		app.menu.ShowError("No report configurations found in configs/reports/")
		app.menu.Pause()
		return
	}

	// Convert to menu.ReportInfo
	menuReports := make([]pkg.ReportInfo, len(reports))
	for i, r := range reports {
		menuReports[i] = pkg.ReportInfo{
			Title:      r.Title,
			ConfigFile: r.ConfigFile,
			Category:   r.Category,
			Version:    r.Version,
		}
	}

	choice := app.menu.ShowReportMenuDynamic(menuReports)

	if choice == 0 {
		return
	}

	if choice == len(reports)+1 {
		// Run all reports
		app.menu.ShowHeader()
		app.menu.ShowProgress("Running all reports")
		fmt.Println()

		for _, report := range reports {
			fmt.Printf("  ▶️  Running %s...\n", report.Title)
			app.executeReport(report.ConfigFile)
		}

		app.menu.ShowSuccess("All reports completed!")
		app.menu.ShowInfo(fmt.Sprintf("Reports saved to: %s", app.outputDir))
		app.menu.Pause()
	} else if choice >= 1 && choice <= len(reports) {
		// Run single report
		report := reports[choice-1]
		app.menu.ShowHeader()
		app.menu.ShowProgress(fmt.Sprintf("Running %s", report.Title))
		fmt.Println()

		if app.executeReport(report.ConfigFile) {
			app.menu.ShowSuccess("Report completed successfully!")
			app.menu.ShowInfo(fmt.Sprintf("Report saved to: %s", app.outputDir))
		} else {
			app.menu.ShowError("Report execution failed. Check logs for details.")
		}
		app.menu.Pause()
	} else {
		app.menu.ShowError("Invalid report selection")
		app.menu.Pause()
	}
}

type ReportInfo struct {
	Title      string
	ConfigFile string
	Category   string
	Version    string
}

func (app *App) loadAvailableReports() ([]ReportInfo, error) {
	files, err := os.ReadDir(app.reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read reports directory '%s': %w (try running from the directory containing the executable)", app.reportsDir, err)
	}

	var reports []ReportInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Load the config to get metadata
		configPath := filepath.Join(app.reportsDir, file.Name())
		config, err := pkg.LoadRegistryConfig(configPath)
		if err != nil {
			slog.Warn("Failed to load report config", "file", file.Name(), "error", err)
			continue
		}

		// Use metadata title, fallback to filename
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

func (app *App) executeReport(configFile string) bool {
	configPath := filepath.Join(app.reportsDir, configFile)

	// Validate config file path
	if err := pkg.ValidateFilePath(configFile, []string{".json"}); err != nil {
		fmt.Printf("  ❌  Invalid config file path: %v\n", err)
		return false
	}

	// Load config
	config, err := pkg.LoadRegistryConfig(configPath)
	if err != nil {
		fmt.Printf("  ❌  Failed to load config: %v\n", err)
		return false
	}

	// Validate config structure and all queries
	if err := pkg.ValidateConfig(config); err != nil {
		fmt.Printf("  ❌  Config validation failed: %v\n", err)
		return false
	}

	// Create HTML report using metadata
	reportName := config.Metadata.ReportTitle
	if reportName == "" {
		// Fallback to first query description for backward compatibility
		if len(config.Queries) > 0 && config.Queries[0].Description != "" {
			reportName = config.Queries[0].Description
		} else {
			reportName = configFile
		}
	}

	htmlReport := pkg.NewHTMLReport(reportName, app.outputDir, slog.Default(), app.reader)

	// Add metadata to HTML report
	htmlReport.SetMetadata(config.Metadata)

	// Create evidence logger for compliance audit trail
	reportType := filepath.Base(configFile)
	reportType = reportType[:len(reportType)-5] // Remove .json extension
	evidenceLogger, err := pkg.NewEvidenceLogger(app.evidenceDir, reportType, slog.Default())
	if err != nil {
		fmt.Printf("  ⚠️  Warning: Could not create evidence log: %v\n", err)
	} else {
		// Gather machine information for evidence
		fmt.Println("  📋  Gathering machine information for audit trail...")
		if err := evidenceLogger.GatherMachineInfo(app.reader); err != nil {
			fmt.Printf("  ⚠️  Warning: Could not gather machine info: %v\n", err)
		}
	}

	ctx := context.Background()
	successCount := 0
	errorCount := 0

	// Execute queries
	for _, query := range config.Queries {
		if query.Operation != "read" {
			continue
		}

		// Additional runtime validation with security policy enforcement
		if err := pkg.ValidateAgainstDenyList(query.Path, app.config.Security.DenyRegistryPaths); err != nil {
			fmt.Printf("  🔒  [%s] Blocked by security policy: %s\n", query.Name, query.Path)
			htmlReport.AddResult(query.Name, query.Description, nil, err)
			if evidenceLogger != nil {
				evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		if err := pkg.ValidateAgainstAllowList(query.RootKey, app.config.Security.AllowedRegistryRoots); err != nil {
			fmt.Printf("  🔒  [%s] Root key not allowed: %s\n", query.Name, query.RootKey)
			htmlReport.AddResult(query.Name, query.Description, nil, err)
			if evidenceLogger != nil {
				evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		rootKey, err := pkg.ParseRootKey(query.RootKey)
		if err != nil {
			fmt.Printf("  ⚠️  [%s] Invalid root key: %s\n", query.Name, query.RootKey)
			htmlReport.AddResult(query.Name, query.Description, nil, err)
			if evidenceLogger != nil {
				evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		if query.ReadAll {
			// Batch read
			data, err := app.reader.BatchRead(ctx, rootKey, query.Path, []string{})
			if err != nil {
				if pkg.IsNotExist(err) {
					fmt.Printf("  ⚠️  [%s] Not found\n", query.Name)
				} else {
					fmt.Printf("  ❌  [%s] Error: %v\n", query.Name, err)
				}
				htmlReport.AddResult(query.Name, query.Description, nil, err)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, "", nil, err)
				}
				errorCount++
			} else {
				fmt.Printf("  ✅  [%s] Read %d values\n", query.Name, len(data))
				htmlReport.AddResult(query.Name, query.Description, data, nil)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, "", data, nil)
				}
				successCount++
			}
		} else {
			// Single value read (auto-detect type: string, integer, or binary)
			value, err := app.reader.ReadValue(ctx, rootKey, query.Path, query.ValueName)
			if err != nil {
				if pkg.IsNotExist(err) {
					fmt.Printf("  ⚠️  [%s] Not found\n", query.Name)
				} else {
					fmt.Printf("  ❌  [%s] Error: %v\n", query.Name, err)
				}
				htmlReport.AddResultWithDetails(
					query.Name,
					query.Description,
					query.RootKey,
					query.Path,
					query.ValueName,
					query.ExpectedValue,
					nil,
					err,
				)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
				}
				errorCount++
			} else {
				fmt.Printf("  ✅  [%s] Success\n", query.Name)
				htmlReport.AddResultWithDetails(
					query.Name,
					query.Description,
					query.RootKey,
					query.Path,
					query.ValueName,
					query.ExpectedValue,
					value,
					nil,
				)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, value, nil)
				}
				successCount++
			}
		}
	}

	// Generate HTML report
	if err := htmlReport.Generate(); err != nil {
		fmt.Printf("  ❌  Failed to generate HTML report: %v\n", err)
		return false
	}

	// Finalize evidence log
	if evidenceLogger != nil {
		fmt.Println("  📝  Finalizing compliance evidence log...")
		if err := evidenceLogger.Finalize(); err != nil {
			fmt.Printf("  ⚠️  Warning: Could not finalize evidence log: %v\n", err)
		} else {
			fmt.Println()
			fmt.Println(evidenceLogger.GetSummaryText())
		}
	}

	fmt.Println()
	fmt.Printf("  📊  Results: %d successful, %d errors\n", successCount, errorCount)
	fmt.Printf("  📄  HTML Report: %s\n", htmlReport.OutputPath)
	if evidenceLogger != nil {
		fmt.Printf("  📋  Evidence Log: %s\n", evidenceLogger.LogPath)
	}

	return true
}

func (app *App) viewHTMLReports() {
	// Find HTML reports
	reports, err := filepath.Glob(filepath.Join(app.outputDir, "*.html"))
	if err != nil {
		app.menu.ShowError(fmt.Sprintf("Failed to list reports: %v", err))
		return
	}

	if len(reports) == 0 {
		app.menu.ShowHeader()
		fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
		fmt.Println("│                         NO REPORTS FOUND                             │")
		fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
		fmt.Println("│                                                                      │")
		fmt.Println("│  No HTML reports have been generated yet.                            │")
		fmt.Println("│                                                                      │")
		fmt.Println("│  Please run reports from the 'Run Reports' menu first.               │")
		fmt.Println("│                                                                      │")
		fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		app.menu.Pause()
		return
	}

	// Show report selection
	app.menu.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                          HTML REPORTS                                │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                      │")

	for i, report := range reports {
		basename := filepath.Base(report)
		fmt.Printf("│  [%d]  %-60s │\n", i+1, basename)
	}

	fmt.Println("│                                                                      │")
	fmt.Println("│  [0]  Back to Main Menu                                              │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Print("  ➤  Select report to open: ")

	choice := app.menu.GetIntInput()

	if choice == 0 {
		return
	}

	if choice > 0 && choice <= len(reports) {
		reportPath := reports[choice-1]
		app.menu.ShowProgress(fmt.Sprintf("Opening %s", filepath.Base(reportPath)))

		// Open in default browser
		if err := app.openBrowser(reportPath); err != nil {
			app.menu.ShowError(fmt.Sprintf("Failed to open report: %v", err))
		} else {
			app.menu.ShowSuccess("Report opened in browser")
		}
		app.menu.Pause()
	} else {
		app.menu.ShowError("Invalid selection")
	}
}

func (app *App) viewEvidenceLogs() {
	evidenceLogs, err := filepath.Glob(filepath.Join(app.evidenceDir, "*.json"))
	if err != nil {
		app.menu.ShowError(fmt.Sprintf("Failed to list evidence logs: %v", err))
		app.menu.Pause()
		return
	}

	app.menu.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                        EVIDENCE LOGS                                 │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                      │")

	if len(evidenceLogs) == 0 {
		fmt.Println("│  No evidence logs found.                                             │")
		fmt.Println("│  Evidence logs are created when you run compliance reports.         │")
		fmt.Println("│                                                                      │")
		fmt.Printf("│  Evidence Directory: %-47s │\n", app.evidenceDir)
		fmt.Println("│                                                                      │")
		fmt.Println("│  [0]  Back to Main Menu                                              │")
		fmt.Println("│                                                                      │")
		fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		app.menu.Pause()
		return
	}

	for i, logFile := range evidenceLogs {
		basename := filepath.Base(logFile)
		if len(basename) > 60 {
			basename = basename[:57] + "..."
		}
		fmt.Printf("│  [%d]  %-60s │\n", i+1, basename)
	}

	fmt.Println("│                                                                      │")
	fmt.Printf("│  Evidence Directory: %-47s │\n", app.evidenceDir)
	fmt.Println("│                                                                      │")
	fmt.Println("│  [0]  Back to Main Menu                                              │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Print("  ➤  Select evidence log to open: ")

	choice := app.menu.GetIntInput()

	if choice == 0 {
		return
	}

	if choice > 0 && choice <= len(evidenceLogs) {
		logPath := evidenceLogs[choice-1]
		app.menu.ShowProgress(fmt.Sprintf("Opening %s", filepath.Base(logPath)))

		// Open in default text editor
		if err := app.openFile(logPath); err != nil {
			app.menu.ShowError(fmt.Sprintf("Failed to open evidence log: %v", err))
		} else {
			app.menu.ShowSuccess("Evidence log opened")
		}
		app.menu.Pause()
	} else {
		app.menu.ShowError("Invalid selection")
		app.menu.Pause()
	}
}

func (app *App) viewLogFiles() {
	logs, err := filepath.Glob(filepath.Join(app.logsDir, "*.log"))
	if err != nil {
		app.menu.ShowError(fmt.Sprintf("Failed to list logs: %v", err))
		app.menu.Pause()
		return
	}

	app.menu.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                           LOG FILES                                  │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                      │")

	if len(logs) == 0 {
		fmt.Println("│  No log files found.                                                 │")
		fmt.Println("│                                                                      │")
		fmt.Printf("│  Log Directory: %-51s │\n", app.logsDir)
		fmt.Println("│                                                                      │")
		fmt.Println("│  [0]  Back to Main Menu                                              │")
		fmt.Println("│                                                                      │")
		fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		app.menu.Pause()
		return
	}

	for i, logFile := range logs {
		basename := filepath.Base(logFile)
		if len(basename) > 60 {
			basename = basename[:57] + "..."
		}
		fmt.Printf("│  [%d]  %-60s │\n", i+1, basename)
	}

	fmt.Println("│                                                                      │")
	fmt.Printf("│  Log Directory: %-51s │\n", app.logsDir)
	fmt.Println("│                                                                      │")
	fmt.Println("│  [0]  Back to Main Menu                                              │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Print("  ➤  Select log file to open: ")

	choice := app.menu.GetIntInput()

	if choice == 0 {
		return
	}

	if choice > 0 && choice <= len(logs) {
		logPath := logs[choice-1]
		app.menu.ShowProgress(fmt.Sprintf("Opening %s", filepath.Base(logPath)))

		// Open in default text editor
		if err := app.openFile(logPath); err != nil {
			app.menu.ShowError(fmt.Sprintf("Failed to open log file: %v", err))
		} else {
			app.menu.ShowSuccess("Log file opened")
		}
		app.menu.Pause()
	} else {
		app.menu.ShowError("Invalid selection")
		app.menu.Pause()
	}
}

func (app *App) configuration() {
	app.menu.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                         CONFIGURATION                                │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                      │")
	fmt.Printf("│  Current Settings:                                                   │\n")
	fmt.Println("│                                                                      │")

	// Server/Runtime
	fmt.Println("│  ▶ Server/Runtime:                                                   │")
	fmt.Printf("│    Read Timeout:      %-46s │\n", app.config.Server.ReadTimeout)
	fmt.Printf("│    Max Concurrent:    %-46d │\n", app.config.Server.MaxConcurrentReads)
	fmt.Println("│                                                                      │")

	// Logging
	fmt.Println("│  ▶ Logging:                                                          │")
	fmt.Printf("│    Level:             %-46s │\n", app.config.Logging.Level)
	fmt.Printf("│    Format:            %-46s │\n", app.config.Logging.Format)
	fmt.Printf("│    File Logging:      %-46v │\n", app.config.Logging.EnableFileLogging)
	fmt.Println("│                                                                      │")

	// Reports
	fmt.Println("│  ▶ Reports:                                                          │")
	fmt.Printf("│    Config Path:       %-46s │\n", truncate(app.config.Reports.ConfigPath, 46))
	fmt.Printf("│    Output Path:       %-46s │\n", truncate(app.config.Reports.OutputPath, 46))
	fmt.Printf("│    Evidence Path:     %-46s │\n", truncate(app.config.Reports.EvidencePath, 46))
	fmt.Printf("│    Evidence Enabled:  %-46v │\n", app.config.Reports.EnableEvidence)
	fmt.Printf("│    Dark Mode:         %-46v │\n", app.config.Reports.EnableDarkMode)
	fmt.Println("│                                                                      │")

	// Security
	fmt.Println("│  ▶ Security:                                                         │")
	fmt.Printf("│    Read-Only:         %-46v │\n", app.config.Security.ReadOnly)
	fmt.Printf("│    Audit Mode:        %-46v │\n", app.config.Security.AuditMode)
	fmt.Printf("│    Allowed Roots:     %-46d │\n", len(app.config.Security.AllowedRegistryRoots))
	fmt.Println("│                                                                      │")

	fmt.Println("│  Configuration can be managed via:                                   │")
	fmt.Println("│    • config/config.yaml (YAML file)                                  │")
	fmt.Println("│    • COMPLIANCE_TOOLKIT_* environment variables                      │")
	fmt.Println("│    • Command-line flags (--help for list)                            │")
	fmt.Println("│                                                                      │")
	fmt.Println("│  Generate default config: ComplianceToolkit.exe --generate-config    │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	app.menu.Pause()
}

// truncate string to specified length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (app *App) exit() {
	app.menu.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                                                                      │")
	fmt.Println("│                   Thank you for using                                │")
	fmt.Println("│                   COMPLIANCE TOOLKIT                                 │")
	fmt.Println("│                                                                      │")
	fmt.Println("│              Stay secure, stay compliant!                            │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func (app *App) openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use "" as first arg to prevent issues with paths containing spaces
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

func (app *App) openFile(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use explorer to open file with default program
		cmd = exec.Command("explorer", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default: // linux
		cmd = exec.Command("xdg-open", filePath)
	}

	return cmd.Start()
}

// CLI-specific functions for non-interactive mode

func (app *App) listReportsCLI() {
	reports, err := app.loadAvailableReports()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load reports: %v\n", err)
		os.Exit(1)
	}

	if len(reports) == 0 {
		fmt.Println("No reports found in configs/reports/")
		return
	}

	fmt.Println("Available Reports:")
	fmt.Println("==================")
	for _, report := range reports {
		fmt.Printf("  - %s\n", report.ConfigFile)
		fmt.Printf("    Title:    %s\n", report.Title)
		fmt.Printf("    Category: %s\n", report.Category)
		fmt.Printf("    Version:  %s\n", report.Version)
		fmt.Println()
	}
	fmt.Println("To run a specific report:")
	fmt.Printf("  ComplianceToolkit.exe -report=<report-name.json>\n\n")
	fmt.Println("To run all reports:")
	fmt.Printf("  ComplianceToolkit.exe -report=all\n")
}

func (app *App) runReportCLI(reportName string, quiet bool) bool {
	reports, err := app.loadAvailableReports()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load reports: %v\n", err)
		return false
	}

	if len(reports) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No reports found in configs/reports/\n")
		return false
	}

	// Handle "all" reports
	if strings.ToLower(reportName) == "all" {
		if !quiet {
			fmt.Println("Running all reports...")
			fmt.Println("======================")
		}

		allSuccess := true
		for _, report := range reports {
			if !quiet {
				fmt.Printf("\n▶ Running: %s\n", report.Title)
			}
			success := app.executeReportQuiet(report.ConfigFile, quiet)
			if !success {
				allSuccess = false
				if !quiet {
					fmt.Printf("  ❌ FAILED\n")
				}
			} else {
				if !quiet {
					fmt.Printf("  ✅ SUCCESS\n")
				}
			}
		}

		if !quiet {
			fmt.Println("\n======================")
			if allSuccess {
				fmt.Println("All reports completed successfully!")
			} else {
				fmt.Println("Some reports failed. Check logs for details.")
			}
			fmt.Printf("Reports saved to: %s\n", app.outputDir)
			fmt.Printf("Evidence saved to: %s\n", app.evidenceDir)
		}

		return allSuccess
	}

	// Run specific report
	var selectedReport *ReportInfo
	for i := range reports {
		if reports[i].ConfigFile == reportName {
			selectedReport = &reports[i]
			break
		}
	}

	if selectedReport == nil {
		fmt.Fprintf(os.Stderr, "Error: Report '%s' not found\n", reportName)
		fmt.Fprintf(os.Stderr, "Use -list to see available reports\n")
		return false
	}

	if !quiet {
		fmt.Printf("Running: %s\n", selectedReport.Title)
		fmt.Println("======================")
	}

	success := app.executeReportQuiet(selectedReport.ConfigFile, quiet)

	if !quiet {
		fmt.Println("======================")
		if success {
			fmt.Println("Report completed successfully!")
		} else {
			fmt.Println("Report execution failed. Check logs for details.")
		}
		fmt.Printf("Report saved to: %s\n", app.outputDir)
		fmt.Printf("Evidence saved to: %s\n", app.evidenceDir)
	}

	return success
}

func (app *App) executeReportQuiet(configFile string, quiet bool) bool {
	configPath := filepath.Join(app.reportsDir, configFile)

	// Validate config file path
	if err := pkg.ValidateFilePath(configFile, []string{".json"}); err != nil {
		if !quiet {
			fmt.Printf("Invalid config file path: %v\n", err)
		}
		slog.Error("Invalid config file path", "file", configFile, "error", err)
		return false
	}

	// Load config
	config, err := pkg.LoadRegistryConfig(configPath)
	if err != nil {
		if !quiet {
			fmt.Printf("Failed to load config: %v\n", err)
		}
		slog.Error("Failed to load config", "file", configFile, "error", err)
		return false
	}

	// Validate config structure
	if err := pkg.ValidateConfig(config); err != nil {
		if !quiet {
			fmt.Printf("Config validation failed: %v\n", err)
		}
		slog.Error("Config validation failed", "file", configFile, "error", err)
		return false
	}

	// Create HTML report using metadata
	reportName := config.Metadata.ReportTitle
	if reportName == "" {
		if len(config.Queries) > 0 && config.Queries[0].Description != "" {
			reportName = config.Queries[0].Description
		} else {
			reportName = configFile
		}
	}

	htmlReport := pkg.NewHTMLReport(reportName, app.outputDir, slog.Default(), app.reader)
	htmlReport.SetMetadata(config.Metadata)

	// Create evidence logger
	reportType := filepath.Base(configFile)
	reportType = reportType[:len(reportType)-5] // Remove .json extension
	evidenceLogger, err := pkg.NewEvidenceLogger(app.evidenceDir, reportType, slog.Default())
	if err != nil {
		if !quiet {
			fmt.Printf("Warning: Could not create evidence log: %v\n", err)
		}
		slog.Warn("Could not create evidence log", "error", err)
	} else {
		if err := evidenceLogger.GatherMachineInfo(app.reader); err != nil {
			if !quiet {
				fmt.Printf("Warning: Could not gather machine info: %v\n", err)
			}
			slog.Warn("Could not gather machine info", "error", err)
		}
	}

	ctx := context.Background()
	successCount := 0
	errorCount := 0

	// Execute queries
	for _, query := range config.Queries {
		if query.Operation != "read" {
			continue
		}

		// Security policy enforcement
		if err := pkg.ValidateAgainstDenyList(query.Path, app.config.Security.DenyRegistryPaths); err != nil {
			if !quiet {
				fmt.Printf("  Blocked by security policy [%s]: %s\n", query.Name, query.Path)
			}
			slog.Warn("Query blocked by security policy", "query", query.Name, "path", query.Path)
			htmlReport.AddResult(query.Name, query.Description, nil, err)
			if evidenceLogger != nil {
				evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		if err := pkg.ValidateAgainstAllowList(query.RootKey, app.config.Security.AllowedRegistryRoots); err != nil {
			if !quiet {
				fmt.Printf("  Root key not allowed [%s]: %s\n", query.Name, query.RootKey)
			}
			slog.Warn("Root key not allowed", "query", query.Name, "root_key", query.RootKey)
			htmlReport.AddResult(query.Name, query.Description, nil, err)
			if evidenceLogger != nil {
				evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		rootKey, err := pkg.ParseRootKey(query.RootKey)
		if err != nil {
			if !quiet {
				fmt.Printf("  Invalid root key [%s]: %s\n", query.Name, query.RootKey)
			}
			htmlReport.AddResult(query.Name, query.Description, nil, err)
			if evidenceLogger != nil {
				evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
			}
			errorCount++
			continue
		}

		if query.ReadAll {
			// Batch read
			data, err := app.reader.BatchRead(ctx, rootKey, query.Path, []string{})
			if err != nil {
				if !quiet && !pkg.IsNotExist(err) {
					fmt.Printf("  Error [%s]: %v\n", query.Name, err)
				}
				htmlReport.AddResult(query.Name, query.Description, nil, err)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, "", nil, err)
				}
				errorCount++
			} else {
				htmlReport.AddResult(query.Name, query.Description, data, nil)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, "", data, nil)
				}
				successCount++
			}
		} else {
			// Single value read
			value, err := app.reader.ReadValue(ctx, rootKey, query.Path, query.ValueName)
			if err != nil {
				if !quiet && !pkg.IsNotExist(err) {
					fmt.Printf("  Error [%s]: %v\n", query.Name, err)
				}
				htmlReport.AddResultWithDetails(
					query.Name,
					query.Description,
					query.RootKey,
					query.Path,
					query.ValueName,
					query.ExpectedValue,
					nil,
					err,
				)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, nil, err)
				}
				errorCount++
			} else {
				htmlReport.AddResultWithDetails(
					query.Name,
					query.Description,
					query.RootKey,
					query.Path,
					query.ValueName,
					query.ExpectedValue,
					value,
					nil,
				)
				if evidenceLogger != nil {
					evidenceLogger.LogResult(query.Name, query.Description, query.Path, query.ValueName, value, nil)
				}
				successCount++
			}
		}
	}

	// Generate HTML report
	if err := htmlReport.Generate(); err != nil {
		if !quiet {
			fmt.Printf("Failed to generate HTML report: %v\n", err)
		}
		slog.Error("Failed to generate HTML report", "error", err)
		return false
	}

	// Finalize evidence log
	if evidenceLogger != nil {
		if err := evidenceLogger.Finalize(); err != nil {
			if !quiet {
				fmt.Printf("Warning: Could not finalize evidence log: %v\n", err)
			}
			slog.Warn("Could not finalize evidence log", "error", err)
		}
	}

	if !quiet {
		fmt.Printf("  Results: %d successful, %d errors\n", successCount, errorCount)
		fmt.Printf("  HTML Report: %s\n", htmlReport.OutputPath)
		if evidenceLogger != nil {
			fmt.Printf("  Evidence Log: %s\n", evidenceLogger.LogPath)
		}
	}

	// Log summary
	slog.Info("Report execution completed",
		"report", reportName,
		"success_count", successCount,
		"error_count", errorCount,
		"html_report", htmlReport.OutputPath,
	)

	return true
}
