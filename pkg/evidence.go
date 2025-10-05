package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/registry"
)

// EvidenceLogger creates comprehensive audit logs for compliance evidence
type EvidenceLogger struct {
	LogPath   string
	StartTime time.Time
	Evidence  *ComplianceEvidence
}

// ComplianceEvidence contains all audit trail information
type ComplianceEvidence struct {
	ScanMetadata  ScanMetadata              `json:"scan_metadata"`
	MachineInfo   MachineInfo               `json:"machine_information"`
	ScanResults   map[string]ScanResult     `json:"scan_results"`
	Summary       ScanSummary               `json:"summary"`
}

// ScanMetadata contains scan execution details
type ScanMetadata struct {
	ToolVersion   string    `json:"tool_version"`
	ScanID        string    `json:"scan_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Duration      string    `json:"duration"`
	Operator      string    `json:"operator"`
	ReportType    string    `json:"report_type"`
}

// MachineInfo contains system identification
type MachineInfo struct {
	Hostname          string    `json:"hostname"`
	OSProductName     string    `json:"os_product_name"`
	OSEdition         string    `json:"os_edition"`
	OSBuildNumber     string    `json:"os_build_number"`
	OSVersion         string    `json:"os_version"`
	RegisteredOwner   string    `json:"registered_owner"`
	RegisteredOrg     string    `json:"registered_organization"`
	InstallDate       string    `json:"install_date"`
	SystemRoot        string    `json:"system_root"`
	Architecture      string    `json:"processor_architecture"`
	ScanTimestamp     time.Time `json:"scan_timestamp"`
}

// ScanResult represents a single check result
type ScanResult struct {
	CheckName       string      `json:"check_name"`
	Description     string      `json:"description"`
	RegistryPath    string      `json:"registry_path"`
	ValueName       string      `json:"value_name"`
	ExpectedValue   string      `json:"expected_value,omitempty"`
	ActualValue     interface{} `json:"actual_value"`
	Status          string      `json:"status"` // PASS, FAIL, NOT_FOUND, ERROR
	Timestamp       time.Time   `json:"timestamp"`
	ErrorMessage    string      `json:"error_message,omitempty"`
	ComplianceNote  string      `json:"compliance_note,omitempty"`
}

// ScanSummary provides scan statistics
type ScanSummary struct {
	TotalChecks    int       `json:"total_checks"`
	Passed         int       `json:"passed"`
	Failed         int       `json:"failed"`
	NotFound       int       `json:"not_found"`
	Errors         int       `json:"errors"`
	ComplianceRate float64   `json:"compliance_rate_percent"`
	Timestamp      time.Time `json:"timestamp"`
}

// NewEvidenceLogger creates a new evidence logger
func NewEvidenceLogger(logDir, reportType string) (*EvidenceLogger, error) {
	timestamp := time.Now()
	scanID := fmt.Sprintf("SCAN_%s", timestamp.Format("20060102_150405"))

	logFilename := fmt.Sprintf("%s_evidence_%s.json", reportType, timestamp.Format("20060102_150405"))
	logPath := fmt.Sprintf("%s/%s", logDir, logFilename)

	// Get operator (Windows username)
	operator := os.Getenv("USERNAME")
	if operator == "" {
		operator = "UNKNOWN"
	}

	evidence := &ComplianceEvidence{
		ScanMetadata: ScanMetadata{
			ToolVersion: "Compliance Toolkit v1.0.0",
			ScanID:      scanID,
			StartTime:   timestamp,
			Operator:    operator,
			ReportType:  reportType,
		},
		ScanResults: make(map[string]ScanResult),
	}

	return &EvidenceLogger{
		LogPath:   logPath,
		StartTime: timestamp,
		Evidence:  evidence,
	}, nil
}

// GatherMachineInfo collects system information for evidence
func (e *EvidenceLogger) GatherMachineInfo(reader *RegistryReader) error {
	ctx := context.Background()
	info := MachineInfo{
		ScanTimestamp: time.Now(),
	}

	// Helper function to safely read registry values
	readValue := func(rootKey registry.Key, path, valueName string) string {
		value, err := reader.ReadString(ctx, rootKey, path, valueName)
		if err != nil {
			return "UNKNOWN"
		}
		return value
	}

	// Gather system information
	info.Hostname = readValue(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName`,
		"ComputerName")

	info.OSProductName = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"ProductName")

	info.OSEdition = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"EditionID")

	info.OSBuildNumber = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"CurrentBuild")

	info.OSVersion = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"CurrentVersion")

	info.RegisteredOwner = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"RegisteredOwner")

	info.RegisteredOrg = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"RegisteredOrganization")

	info.SystemRoot = readValue(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"SystemRoot")

	info.Architecture = readValue(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
		"PROCESSOR_ARCHITECTURE")

	e.Evidence.MachineInfo = info
	return nil
}

// LogResult adds a scan result to the evidence
func (e *EvidenceLogger) LogResult(checkName, description, regPath, valueName string,
	actualValue interface{}, err error) {

	result := ScanResult{
		CheckName:    checkName,
		Description:  description,
		RegistryPath: regPath,
		ValueName:    valueName,
		ActualValue:  actualValue,
		Timestamp:    time.Now(),
	}

	if err != nil {
		if IsNotExist(err) {
			result.Status = "NOT_FOUND"
			result.ErrorMessage = "Registry key or value does not exist"
		} else {
			result.Status = "ERROR"
			result.ErrorMessage = err.Error()
		}
	} else {
		result.Status = "PASS"
	}

	e.Evidence.ScanResults[checkName] = result
}

// Finalize completes the evidence log and writes to file
func (e *EvidenceLogger) Finalize() error {
	endTime := time.Now()
	duration := endTime.Sub(e.StartTime)

	// Update metadata
	e.Evidence.ScanMetadata.EndTime = endTime
	e.Evidence.ScanMetadata.Duration = duration.String()

	// Calculate summary
	summary := ScanSummary{
		TotalChecks: len(e.Evidence.ScanResults),
		Timestamp:   endTime,
	}

	for _, result := range e.Evidence.ScanResults {
		switch result.Status {
		case "PASS":
			summary.Passed++
		case "FAIL":
			summary.Failed++
		case "NOT_FOUND":
			summary.NotFound++
		case "ERROR":
			summary.Errors++
		}
	}

	// Calculate compliance rate (PASS / (PASS + FAIL + ERROR))
	totalValid := summary.Passed + summary.Failed + summary.Errors
	if totalValid > 0 {
		summary.ComplianceRate = (float64(summary.Passed) / float64(totalValid)) * 100.0
	}

	e.Evidence.Summary = summary

	// Write to file
	file, err := os.Create(e.LogPath)
	if err != nil {
		return fmt.Errorf("failed to create evidence log: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(e.Evidence); err != nil {
		return fmt.Errorf("failed to write evidence log: %w", err)
	}

	return nil
}

// GetSummaryText returns a human-readable summary
func (e *EvidenceLogger) GetSummaryText() string {
	s := e.Evidence.Summary
	return fmt.Sprintf(`
COMPLIANCE SCAN SUMMARY
═══════════════════════════════════════════════════════════
Scan ID:          %s
Machine:          %s (%s)
OS:               %s %s (Build %s)
Operator:         %s
Scan Time:        %s
Duration:         %s

RESULTS:
  Total Checks:   %d
  ✅ Passed:      %d
  ❌ Failed:      %d
  ⚠️  Not Found:  %d
  🔴 Errors:      %d

COMPLIANCE RATE:  %.2f%%

Evidence Log:     %s
═══════════════════════════════════════════════════════════
`,
		e.Evidence.ScanMetadata.ScanID,
		e.Evidence.MachineInfo.Hostname,
		e.Evidence.MachineInfo.Architecture,
		e.Evidence.MachineInfo.OSProductName,
		e.Evidence.MachineInfo.OSEdition,
		e.Evidence.MachineInfo.OSBuildNumber,
		e.Evidence.ScanMetadata.Operator,
		e.Evidence.ScanMetadata.StartTime.Format("2006-01-02 15:04:05 MST"),
		e.Evidence.ScanMetadata.Duration,
		s.TotalChecks,
		s.Passed,
		s.Failed,
		s.NotFound,
		s.Errors,
		s.ComplianceRate,
		e.LogPath,
	)
}
