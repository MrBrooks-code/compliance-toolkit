package api

import (
	"fmt"
	"time"
)

// ComplianceSubmission represents a complete compliance report submission from a client
type ComplianceSubmission struct {
	SubmissionID  string          `json:"submission_id"`
	ClientID      string          `json:"client_id"`
	Hostname      string          `json:"hostname"`
	Timestamp     time.Time       `json:"timestamp"`
	ReportType    string          `json:"report_type"`
	ReportVersion string          `json:"report_version"`
	Compliance    ComplianceData  `json:"compliance"`
	Evidence      []EvidenceRecord `json:"evidence,omitempty"`
	SystemInfo    SystemInfo      `json:"system_info"`
}

// ComplianceData contains the actual compliance check results
type ComplianceData struct {
	OverallStatus string        `json:"overall_status"` // "compliant", "non-compliant", "partial"
	TotalChecks   int           `json:"total_checks"`
	PassedChecks  int           `json:"passed_checks"`
	FailedChecks  int           `json:"failed_checks"`
	WarningChecks int           `json:"warning_checks"`
	ErrorChecks   int           `json:"error_checks"`
	Queries       []QueryResult `json:"queries"`
}

// QueryResult represents the result of a single compliance check
type QueryResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Status      string `json:"status"` // "pass", "fail", "warning", "error"
	Expected    string `json:"expected"`
	Actual      string `json:"actual"`
	Message     string `json:"message,omitempty"`
	RootKey     string `json:"root_key,omitempty"`
	Path        string `json:"path,omitempty"`
	ValueName   string `json:"value_name,omitempty"`
}

// EvidenceRecord contains evidence/audit trail for a compliance check
type EvidenceRecord struct {
	QueryName string                 `json:"query_name"`
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SystemInfo contains information about the system being scanned
type SystemInfo struct {
	OSVersion    string `json:"os_version"`
	BuildNumber  string `json:"build_number"`
	Architecture string `json:"architecture"`
	Domain       string `json:"domain,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	MacAddress   string `json:"mac_address,omitempty"`
	LastBootTime string `json:"last_boot_time,omitempty"`
}

// SubmissionResponse is returned after successfully submitting a compliance report
type SubmissionResponse struct {
	SubmissionID string    `json:"submission_id"`
	Status       string    `json:"status"` // "accepted", "rejected", "queued"
	Message      string    `json:"message,omitempty"`
	ReceivedAt   time.Time `json:"received_at"`
}

// ClientRegistration represents a client registration request
type ClientRegistration struct {
	ClientID string     `json:"client_id"`
	Hostname string     `json:"hostname"`
	SystemInfo SystemInfo `json:"system_info"`
}

// ClientInfo represents information about a registered client
type ClientInfo struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"client_id"`
	Hostname        string    `json:"hostname"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	Status          string    `json:"status"` // "active", "inactive", "error"
	LastSubmission  string    `json:"last_submission_id,omitempty"`
	ComplianceScore float64   `json:"compliance_score,omitempty"`
	SystemInfo      SystemInfo `json:"system_info"`
}

// DashboardSummary provides a high-level overview for the dashboard
type DashboardSummary struct {
	TotalClients      int                    `json:"total_clients"`
	ActiveClients     int                    `json:"active_clients"`
	CompliantClients  int                    `json:"compliant_clients"`
	RecentSubmissions []SubmissionSummary    `json:"recent_submissions"`
	ComplianceByType  map[string]ComplianceStats `json:"compliance_by_type"`
	Alerts            []Alert                `json:"alerts,omitempty"`
}

// SubmissionSummary provides summary info for a submission
type SubmissionSummary struct {
	SubmissionID  string    `json:"submission_id"`
	ClientID      string    `json:"client_id"`
	Hostname      string    `json:"hostname"`
	Timestamp     time.Time `json:"timestamp"`
	ReportType    string    `json:"report_type"`
	OverallStatus string    `json:"overall_status"`
	PassedChecks  int       `json:"passed_checks"`
	FailedChecks  int       `json:"failed_checks"`
}

// ComplianceStats provides statistics for a specific compliance type
type ComplianceStats struct {
	TotalSubmissions int     `json:"total_submissions"`
	AverageScore     float64 `json:"average_score"`
	PassRate         float64 `json:"pass_rate"`
	FailRate         float64 `json:"fail_rate"`
}

// Alert represents a compliance alert/notification
type Alert struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"` // "info", "warning", "critical"
	Type        string    `json:"type"`
	ClientID    string    `json:"client_id,omitempty"`
	Hostname    string    `json:"hostname,omitempty"`
	Message     string    `json:"message"`
	Acknowledged bool     `json:"acknowledged"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// Validate validates a ComplianceSubmission
func (s *ComplianceSubmission) Validate() error {
	if s.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if s.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if s.ReportType == "" {
		return fmt.Errorf("report_type is required")
	}
	if s.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if len(s.Compliance.Queries) == 0 {
		return fmt.Errorf("compliance queries cannot be empty")
	}
	return nil
}

// CalculateOverallStatus determines the overall compliance status
func (c *ComplianceData) CalculateOverallStatus() string {
	if c.TotalChecks == 0 {
		return "unknown"
	}

	// If any checks failed, not compliant
	if c.FailedChecks > 0 {
		return "non-compliant"
	}

	// If all checks passed, compliant
	if c.PassedChecks == c.TotalChecks {
		return "compliant"
	}

	// If there are warnings or errors, partial
	if c.WarningChecks > 0 || c.ErrorChecks > 0 {
		return "partial"
	}

	return "unknown"
}

// ComplianceScore calculates a 0-100 compliance score
func (c *ComplianceData) ComplianceScore() float64 {
	if c.TotalChecks == 0 {
		return 0.0
	}
	return (float64(c.PassedChecks) / float64(c.TotalChecks)) * 100.0
}
