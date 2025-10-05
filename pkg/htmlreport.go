package pkg

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

//go:embed templates/html templates/css
var templateFS embed.FS

// HTMLReport generates HTML reports from registry scan results using templates
type HTMLReport struct {
	Title          string
	Timestamp      time.Time
	Results        map[string]ReportResult
	OutputPath     string
	Metadata       ReportMetadata
	tmpl           *template.Template
	registryReader RegistryService // Changed from *RegistryReader to interface
	logger         *slog.Logger    // Added for dependency injection
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

// NewHTMLReport creates a new HTML report with dependency injection
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

	// Gather system information for evidence panel
	systemInfo := r.gatherSystemInfo()

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

	// Sort results alphabetically by Name
	sort.Slice(queryResults, func(i, j int) bool {
		return queryResults[i].Name < queryResults[j].Name
	})

	// Build report data
	data := &ReportData{
		Metadata:    r.Metadata,
		GeneratedAt: r.Timestamp,
		MachineName: machineName,
		SystemInfo:  systemInfo,
		Results:     queryResults,
	}

	// Calculate statistics
	data.CalculateStats()

	return data
}

// SetMetadata sets the report metadata
func (r *HTMLReport) SetMetadata(metadata ReportMetadata) {
	r.Metadata = metadata
}

// GetOutputPath returns the output path of the report
func (r *HTMLReport) GetOutputPath() string {
	return r.OutputPath
}

// gatherSystemInfo collects system information for the evidence panel
func (r *HTMLReport) gatherSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()

	info := SystemInfo{
		Hostname:        hostname,
		OSProductName:   os.Getenv("OS"),
		OSEdition:       "N/A",
		OSBuildNumber:   "N/A",
		OSVersion:       "N/A",
		RegisteredOwner: os.Getenv("USERNAME"),
		RegisteredOrg:   os.Getenv("USERDOMAIN"),
		Architecture:    os.Getenv("PROCESSOR_ARCHITECTURE"),
		InstallDate:     "N/A",
		SystemRoot:      os.Getenv("SystemRoot"),
	}

	// If registry reader available, get detailed info from registry
	if r.registryReader != nil {
		info = r.gatherDetailedSystemInfo()
	}

	return info
}

// gatherDetailedSystemInfo uses registry reader to get comprehensive system info
func (r *HTMLReport) gatherDetailedSystemInfo() SystemInfo {
	ctx := context.Background()
	info := SystemInfo{}

	// Helper to safely read registry values
	readValue := func(path, valueName string) string {
		value, err := r.registryReader.ReadString(ctx, registry.LOCAL_MACHINE, path, valueName)
		if err != nil {
			return "N/A"
		}
		return value
	}

	// Gather comprehensive system information
	info.Hostname = readValue(
		`SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName`,
		"ComputerName")

	info.OSProductName = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"ProductName")

	info.OSEdition = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"EditionID")

	info.OSBuildNumber = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"CurrentBuild")

	info.OSVersion = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"CurrentVersion")

	info.RegisteredOwner = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"RegisteredOwner")

	info.RegisteredOrg = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"RegisteredOrganization")

	info.SystemRoot = readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"SystemRoot")

	// Get architecture
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")
	if arch == "" {
		arch = readValue(
			`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
			"PROCESSOR_ARCHITECTURE")
	}
	info.Architecture = arch

	// Get install date (this is a DWORD unix timestamp)
	installDateStr := readValue(
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"InstallDate")
	if installDateStr != "N/A" {
		info.InstallDate = installDateStr
	}

	// Network Information
	info.DomainName = r.getDomainName(ctx)
	info.IPAddress = r.getIPAddress()
	info.MACAddress = r.getMACAddress()

	return info
}

// getDomainName retrieves the domain name from registry
func (r *HTMLReport) getDomainName(ctx context.Context) string {
	// Try to get domain from registry
	domain, err := r.registryReader.ReadString(ctx, registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`,
		"Domain")
	if err == nil && domain != "" {
		return domain
	}

	// Try NV Domain (non-volatile)
	domain, err = r.registryReader.ReadString(ctx, registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`,
		"NV Domain")
	if err == nil && domain != "" {
		return domain
	}

	// Fallback to USERDNSDOMAIN environment variable
	domain = os.Getenv("USERDNSDOMAIN")
	if domain != "" {
		return domain
	}

	return "WORKGROUP"
}

// getIPAddress retrieves the primary IP address
func (r *HTMLReport) getIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "N/A"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "N/A"
}

// getMACAddress retrieves the MAC address of the primary network adapter
func (r *HTMLReport) getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "N/A"
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Get MAC address
		if len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String()
		}
	}

	return "N/A"
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
