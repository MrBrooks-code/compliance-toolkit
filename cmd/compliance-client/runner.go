package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sys/windows/registry"

	"compliancetoolkit/pkg"
	"compliancetoolkit/pkg/api"
)

// ReportRunner executes compliance reports and generates submissions
type ReportRunner struct {
	config *ClientConfig
	logger *slog.Logger
	reader *pkg.RegistryReader
}

// NewReportRunner creates a new report runner
func NewReportRunner(config *ClientConfig, logger *slog.Logger) *ReportRunner {
	// Create registry reader
	reader := pkg.NewRegistryReader(
		pkg.WithLogger(logger),
		pkg.WithTimeout(5*time.Second),
	)

	return &ReportRunner{
		config: config,
		logger: logger,
		reader: reader,
	}
}

// Run executes a report and returns a ComplianceSubmission
func (r *ReportRunner) Run(reportName string) (*api.ComplianceSubmission, error) {
	startTime := time.Now()

	// Load report configuration
	reportConfig, err := r.loadReportConfig(reportName)
	if err != nil {
		return nil, fmt.Errorf("failed to load report config: %w", err)
	}

	r.logger.Info("Loaded report configuration",
		"report", reportConfig.Metadata.ReportTitle,
		"version", reportConfig.Metadata.ReportVersion,
		"queries", len(reportConfig.Queries),
	)

	// Execute all queries
	results := make([]api.QueryResult, 0, len(reportConfig.Queries))
	evidence := make([]api.EvidenceRecord, 0)

	for _, query := range reportConfig.Queries {
		result, evidenceRec := r.executeQuery(query)
		results = append(results, result)
		if evidenceRec != nil {
			evidence = append(evidence, *evidenceRec)
		}
	}

	// Calculate compliance statistics
	complianceData := r.calculateCompliance(results)

	// Collect system information
	sysInfo := r.collectSystemInfo()

	// Generate submission ID
	submissionID := uuid.New().String()

	// Create submission
	submission := &api.ComplianceSubmission{
		SubmissionID:  submissionID,
		ClientID:      r.config.Client.ID,
		Hostname:      r.config.Client.Hostname,
		Timestamp:     time.Now(),
		ReportType:    reportConfig.Metadata.ReportTitle,
		ReportVersion: reportConfig.Metadata.ReportVersion,
		Compliance:    complianceData,
		Evidence:      evidence,
		SystemInfo:    sysInfo,
	}

	// Save local HTML report if configured
	if r.config.Reports.SaveLocal {
		if err := r.saveHTMLReport(reportConfig, results); err != nil {
			r.logger.Warn("Failed to save HTML report", "error", err)
			// Don't fail - report execution succeeded
		}
	}

	duration := time.Since(startTime)
	r.logger.Info("Report execution completed",
		"submission_id", submissionID,
		"duration", duration,
	)

	return submission, nil
}

// loadReportConfig loads a report configuration file
func (r *ReportRunner) loadReportConfig(reportName string) (*pkg.RegistryConfig, error) {
	// Build path to config file
	configPath := filepath.Join(r.config.Reports.ConfigPath, reportName)

	// Load using existing pkg function
	config, err := pkg.LoadRegistryConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load report: %w", err)
	}

	return config, nil
}

// executeQuery executes a single registry query
func (r *ReportRunner) executeQuery(query pkg.RegistryQuery) (api.QueryResult, *api.EvidenceRecord) {
	ctx := context.Background()
	queryStart := time.Now()

	result := api.QueryResult{
		Name:        query.Name,
		Description: query.Description,
		Expected:    query.ExpectedValue,
		RootKey:     query.RootKey,
		Path:        query.Path,
		ValueName:   query.ValueName,
	}

	// Parse root key
	rootKey, err := pkg.ParseRootKey(query.RootKey)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Invalid root key: %v", err)
		result.Actual = "error"
		return result, nil
	}

	// Execute registry read
	value, err := r.reader.ReadValue(ctx, rootKey, query.Path, query.ValueName)

	// Create evidence record
	evidence := &api.EvidenceRecord{
		QueryName: query.Name,
		Timestamp: time.Now(),
		Action:    "registry_read",
		Details: map[string]interface{}{
			"root_key":   query.RootKey,
			"path":       query.Path,
			"value_name": query.ValueName,
			"duration":   time.Since(queryStart).Milliseconds(),
		},
	}

	if err != nil {
		// Check if it's a "not found" error
		if pkg.IsNotExist(err) {
			result.Status = "fail"
			result.Actual = "not found"
			result.Message = "Registry key or value not found"
			evidence.Result = "not_found"
		} else {
			result.Status = "error"
			result.Actual = "error"
			result.Message = err.Error()
			evidence.Result = "error"
		}
		evidence.Details["error"] = err.Error()
		return result, evidence
	}

	// Success - compare with expected value
	result.Actual = value
	evidence.Result = "success"
	evidence.Details["actual_value"] = value

	// Simple comparison (case-insensitive)
	if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(query.ExpectedValue)) {
		result.Status = "pass"
	} else {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Expected '%s', got '%s'", query.ExpectedValue, value)
	}

	return result, evidence
}

// calculateCompliance calculates overall compliance statistics
func (r *ReportRunner) calculateCompliance(results []api.QueryResult) api.ComplianceData {
	data := api.ComplianceData{
		TotalChecks: len(results),
		Queries:     results,
	}

	for _, result := range results {
		switch result.Status {
		case "pass":
			data.PassedChecks++
		case "fail":
			data.FailedChecks++
		case "warning":
			data.WarningChecks++
		case "error":
			data.ErrorChecks++
		}
	}

	data.OverallStatus = data.CalculateOverallStatus()

	return data
}

// collectSystemInfo collects system information
func (r *ReportRunner) collectSystemInfo() api.SystemInfo {
	info := api.SystemInfo{
		OSVersion:    "Windows",
		Architecture: runtime.GOARCH,
	}

	// Try to get detailed OS version
	if osVersion := r.getWindowsVersion(); osVersion != "" {
		info.OSVersion = osVersion
	}

	// Try to get build number
	if buildNumber := r.getBuildNumber(); buildNumber != "" {
		info.BuildNumber = buildNumber
	}

	// Try to get domain
	if domain := r.getDomain(); domain != "" {
		info.Domain = domain
	}

	return info
}

// getWindowsVersion attempts to get Windows version from registry
func (r *ReportRunner) getWindowsVersion() string {
	ctx := context.Background()

	// Try to read from registry
	productName, err := r.reader.ReadValue(ctx, registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName")
	if err == nil && productName != "" {
		return productName
	}

	return "Windows (version unknown)"
}

// getBuildNumber attempts to get Windows build number
func (r *ReportRunner) getBuildNumber() string {
	ctx := context.Background()

	buildNumber, err := r.reader.ReadValue(ctx, registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentBuildNumber")
	if err == nil && buildNumber != "" {
		return buildNumber
	}

	return ""
}

// getDomain attempts to get computer domain
func (r *ReportRunner) getDomain() string {
	ctx := context.Background()

	domain, err := r.reader.ReadValue(ctx, registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`, "Domain")
	if err == nil && domain != "" {
		return domain
	}

	return ""
}

// saveHTMLReport generates and saves an HTML report locally
func (r *ReportRunner) saveHTMLReport(reportConfig *pkg.RegistryConfig, results []api.QueryResult) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.config.Reports.OutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create HTML report using existing pattern
	htmlReport := pkg.NewHTMLReport(
		reportConfig.Metadata.ReportTitle,
		r.config.Reports.OutputPath,
		r.logger,
		r.reader,
	)

	// Set metadata
	htmlReport.SetMetadata(reportConfig.Metadata)

	// Add all results to HTML report
	for _, result := range results {
		var value interface{} = result.Actual
		var err error
		if result.Status == "error" || result.Status == "fail" {
			err = fmt.Errorf("%s", result.Message)
		}

		htmlReport.AddResultWithDetails(
			result.Name,
			result.Description,
			result.RootKey,
			result.Path,
			result.ValueName,
			result.Expected,
			value,
			err,
		)
	}

	// Generate HTML report
	if err := htmlReport.Generate(); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	r.logger.Info("HTML report saved", "path", htmlReport.OutputPath)
	return nil
}
