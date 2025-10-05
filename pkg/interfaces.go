package pkg

import (
	"context"

	"golang.org/x/sys/windows/registry"
)

// RegistryService defines operations for reading Windows Registry
type RegistryService interface {
	// ReadString reads a string value from the registry
	ReadString(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)

	// ReadInteger reads a DWORD/QWORD value from the registry
	ReadInteger(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error)

	// ReadBinary reads a binary value from the registry
	ReadBinary(ctx context.Context, rootKey registry.Key, path, valueName string) ([]byte, error)

	// ReadStrings reads a multi-string value from the registry
	ReadStrings(ctx context.Context, rootKey registry.Key, path, valueName string) ([]string, error)

	// ReadValue reads any registry value and returns it as a string (auto-detects type)
	ReadValue(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)

	// BatchRead reads multiple values from the same registry key efficiently
	BatchRead(ctx context.Context, rootKey registry.Key, path string, values []string) (map[string]interface{}, error)
}

// ReportService defines operations for generating compliance reports
type ReportService interface {
	// Generate creates the HTML file using the template system
	Generate() error

	// AddResult adds a result to the report
	AddResult(name, description string, value interface{}, err error)

	// AddResultWithDetails adds a result with full query details for compliance reporting
	AddResultWithDetails(name, description, rootKey, path, valueName, expectedValue string, value interface{}, err error)

	// SetMetadata sets report metadata
	SetMetadata(metadata ReportMetadata)

	// GetOutputPath returns the output path of the report
	GetOutputPath() string
}

// EvidenceService defines operations for compliance evidence logging
type EvidenceService interface {
	// GatherMachineInfo collects system information for evidence
	GatherMachineInfo(reader RegistryService) error

	// LogResult adds a scan result to the evidence
	LogResult(checkName, description, regPath, valueName string, actualValue interface{}, err error)

	// Finalize completes the evidence log and writes to file
	Finalize() error

	// GetSummaryText returns a human-readable summary
	GetSummaryText() string

	// GetLogPath returns the log file path
	GetLogPath() string
}

// UIService defines operations for user interaction
type UIService interface {
	// ShowHeader displays the application header
	ShowHeader()

	// ShowMainMenu displays the main menu and returns the user's choice
	ShowMainMenu() int

	// ShowReportMenuDynamic displays the report selection menu with dynamically loaded reports
	ShowReportMenuDynamic(reports []ReportInfo) int

	// ShowError displays an error message
	ShowError(message string)

	// ShowSuccess displays a success message
	ShowSuccess(message string)

	// ShowInfo displays an info message
	ShowInfo(message string)

	// ShowProgress displays a progress message
	ShowProgress(message string)

	// Pause waits for user to press enter
	Pause()

	// GetIntInput reads an integer from stdin
	GetIntInput() int

	// GetStringInput reads a string from stdin
	GetStringInput() string

	// Confirm asks for yes/no confirmation
	Confirm(message string) bool

	// ShowAbout displays the about screen
	ShowAbout()
}

// ConfigService defines operations for loading configurations
type ConfigService interface {
	// LoadConfig loads a configuration from file
	LoadConfig(path string) (*RegistryConfig, error)

	// ParseRootKey parses a root key string to registry.Key
	ParseRootKey(rootKeyStr string) (registry.Key, error)
}

// FileService defines operations for file and directory management
type FileService interface {
	// FindReportsDirectory looks for the reports directory in multiple locations
	FindReportsDirectory(exeDir string) string

	// ResolveDirectory converts relative paths to absolute paths based on exe location
	ResolveDirectory(dir, exeDir string) string

	// ListReports lists all available reports in a directory
	ListReports(reportsDir string) ([]ReportInfo, error)

	// OpenBrowser opens a URL in the default browser
	OpenBrowser(url string) error

	// OpenFile opens a file with the default program
	OpenFile(filePath string) error
}
