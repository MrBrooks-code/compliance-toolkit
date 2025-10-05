package pkg

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed templates/html templates/css
var templateFS embed.FS

// HTMLReport generates HTML reports from registry scan results using templates
type HTMLReport struct {
	Title      string
	Timestamp  time.Time
	Results    map[string]ReportResult
	OutputPath string
	Metadata   ReportMetadata
	tmpl       *template.Template
}

// ReportResult represents a single query result
type ReportResult struct {
	Description   string
	Value         interface{}
	Error         string
	Success       bool
	RootKey       string
	Path          string
	ValueName     string
	ExpectedValue string
}

// NewHTMLReport creates a new HTML report
func NewHTMLReport(title, outputDir string) *HTMLReport {
	timestamp := time.Now()
	filename := fmt.Sprintf("%s_%s.html",
		sanitizeFilename(title),
		timestamp.Format("20060102_150405"))

	return &HTMLReport{
		Title:      title,
		Timestamp:  timestamp,
		Results:    make(map[string]ReportResult),
		OutputPath: filepath.Join(outputDir, filename),
	}
}

// AddResult adds a result to the report
func (r *HTMLReport) AddResult(name, description string, value interface{}, err error) {
	result := ReportResult{
		Description: description,
		Value:       value,
		Success:     err == nil,
	}

	if err != nil {
		result.Error = err.Error()
	}

	r.Results[name] = result
}

// AddResultWithDetails adds a result with full query details for compliance reporting
func (r *HTMLReport) AddResultWithDetails(name, description, rootKey, path, valueName, expectedValue string, value interface{}, err error) {
	result := ReportResult{
		Description:   description,
		Value:         value,
		Success:       err == nil,
		RootKey:       rootKey,
		Path:          path,
		ValueName:     valueName,
		ExpectedValue: expectedValue,
	}

	if err != nil {
		result.Error = err.Error()
	}

	r.Results[name] = result
}

// Generate creates the HTML file using the template system
func (r *HTMLReport) Generate() error {
	// Parse templates
	if err := r.loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Build report data
	data := r.buildReportData()

	// Ensure directory exists
	dir := filepath.Dir(r.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(r.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := r.tmpl.ExecuteTemplate(file, "base.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// loadTemplates loads and parses all HTML and CSS templates
func (r *HTMLReport) loadTemplates() error {
	// Define template functions
	funcMap := template.FuncMap{
		"formatValue": formatValue,
	}

	// Parse base template
	tmpl, err := template.New("base.html").Funcs(funcMap).ParseFS(templateFS, "templates/html/base.html")
	if err != nil {
		return fmt.Errorf("failed to parse base template: %w", err)
	}

	// Parse component templates
	tmpl, err = tmpl.ParseFS(templateFS, "templates/html/components/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse component templates: %w", err)
	}

	// Load CSS files as templates
	mainCSS, err := templateFS.ReadFile("templates/css/main.css")
	if err != nil {
		return fmt.Errorf("failed to read main.css: %w", err)
	}
	tmpl, err = tmpl.New("main.css").Parse(string(mainCSS))
	if err != nil {
		return fmt.Errorf("failed to parse main.css: %w", err)
	}

	printCSS, err := templateFS.ReadFile("templates/css/print.css")
	if err != nil {
		return fmt.Errorf("failed to read print.css: %w", err)
	}
	tmpl, err = tmpl.New("print.css").Parse(string(printCSS))
	if err != nil {
		return fmt.Errorf("failed to parse print.css: %w", err)
	}

	r.tmpl = tmpl
	return nil
}

// buildReportData constructs the data structure for template rendering
func (r *HTMLReport) buildReportData() *ReportData {
	// Get machine name
	machineName, _ := os.Hostname()

	// Convert Results map to QueryResult slice
	queryResults := make([]QueryResult, 0, len(r.Results))
	for name, result := range r.Results {
		qr := QueryResult{
			Name:          name,
			Description:   result.Description,
			Operation:     "read",
			Error:         result.Error,
			RootKey:       result.RootKey,
			Path:          result.Path,
			ValueName:     result.ValueName,
			ExpectedValue: result.ExpectedValue,
		}

		// Format value
		if result.Success {
			switch v := result.Value.(type) {
			case map[string]interface{}:
				// Convert to map[string]string for template
				qr.Values = make(map[string]string)
				for k, val := range v {
					qr.Values[k] = fmt.Sprintf("%v", val)
				}
			default:
				qr.Value = formatValue(result.Value)
			}
		}

		queryResults = append(queryResults, qr)
	}

	// Build report data
	data := &ReportData{
		Metadata:    r.Metadata,
		GeneratedAt: r.Timestamp,
		MachineName: machineName,
		Results:     queryResults,
	}

	// Calculate statistics
	data.CalculateStats()

	return data
}

// Helper functions

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case map[string]interface{}:
		result := ""
		for k, v := range val {
			result += fmt.Sprintf("%s = %v\n", k, v)
		}
		return strings.TrimSpace(result)
	case []string:
		return strings.Join(val, "\n")
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func sanitizeFilename(s string) string {
	// Replace spaces and special chars with underscores
	result := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			result += string(c)
		} else if c == ' ' {
			result += "_"
		}
	}
	return result
}
