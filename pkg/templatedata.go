package pkg

import "time"

// ReportData represents the complete data structure passed to HTML templates
type ReportData struct {
	Metadata       ReportMetadata
	GeneratedAt    time.Time
	MachineName    string
	SystemInfo     SystemInfo // Added system information panel
	ComplianceRate float64
	TotalQueries   int
	PassedQueries  int
	FailedQueries  int
	Results        []QueryResult
}

// SystemInfo contains system details for the report evidence
type SystemInfo struct {
	Hostname        string
	OSProductName   string
	OSEdition       string
	OSBuildNumber   string
	OSVersion       string
	RegisteredOwner string
	RegisteredOrg   string
	Architecture    string
	InstallDate     string
	SystemRoot      string
}

// QueryResult represents a single registry query result for template rendering
type QueryResult struct {
	Name          string
	Description   string
	RootKey       string
	Path          string
	ValueName     string
	Operation     string
	Value         string
	Values        map[string]string // For read_all operations
	Error         string
	ExpectedValue string            // Expected value for compliance checks
}

// CalculateStats computes compliance statistics from results
func (rd *ReportData) CalculateStats() {
	rd.TotalQueries = len(rd.Results)
	rd.PassedQueries = 0
	rd.FailedQueries = 0

	for _, result := range rd.Results {
		if result.Error == "" {
			rd.PassedQueries++
		} else {
			rd.FailedQueries++
		}
	}

	if rd.TotalQueries > 0 {
		rd.ComplianceRate = (float64(rd.PassedQueries) / float64(rd.TotalQueries)) * 100.0
	} else {
		rd.ComplianceRate = 0.0
	}
}
