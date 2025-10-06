package pkg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewAuditLogger tests creating an audit logger
func TestNewAuditLogger(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled logger", true},
		{"disabled logger", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewAuditLogger(nil, tt.enabled)
			if logger == nil {
				t.Error("NewAuditLogger() returned nil")
			}
			if logger.IsEnabled() != tt.enabled {
				t.Errorf("NewAuditLogger() enabled = %v, want %v", logger.IsEnabled(), tt.enabled)
			}
		})
	}
}

// TestNewAuditLoggerWithFile tests creating an audit logger with file output
func TestNewAuditLoggerWithFile(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit.log")

	tests := []struct {
		name    string
		path    string
		enabled bool
		wantErr bool
	}{
		{
			name:    "enabled with valid path",
			path:    auditLogPath,
			enabled: true,
			wantErr: false,
		},
		{
			name:    "disabled logger",
			path:    auditLogPath,
			enabled: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewAuditLoggerWithFile(tt.path, tt.enabled)

			if tt.wantErr {
				if err == nil {
					t.Error("NewAuditLoggerWithFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewAuditLoggerWithFile() unexpected error: %v", err)
				return
			}

			if logger == nil {
				t.Error("NewAuditLoggerWithFile() returned nil logger")
				return
			}
			defer logger.Close()

			if logger.IsEnabled() != tt.enabled {
				t.Errorf("NewAuditLoggerWithFile() enabled = %v, want %v", logger.IsEnabled(), tt.enabled)
			}

			// If enabled, check that file was created
			if tt.enabled {
				if _, err := os.Stat(tt.path); os.IsNotExist(err) {
					t.Errorf("NewAuditLoggerWithFile() did not create log file at %s", tt.path)
				}
			}
		})
	}
}

// TestAuditLogger_LogAccess tests logging access events
func TestAuditLogger_LogAccess(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_access.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name     string
		resource string
		action   string
		allowed  bool
	}{
		{"allowed access", "HKLM\\SOFTWARE\\Test", "read", true},
		{"denied access", "HKLM\\SECURITY\\Secrets", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.LogAccess(tt.resource, tt.action, tt.allowed)

			stats := logger.GetStats()
			if stats.TotalEvents == 0 {
				t.Error("LogAccess() did not update statistics")
			}

			if !tt.allowed && stats.DeniedAccess == 0 {
				t.Error("LogAccess() did not track denied access")
			}
		})
	}
}

// TestAuditLogger_LogRegistryRead tests logging registry read operations
func TestAuditLogger_LogRegistryRead(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_registry.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name      string
		rootKey   string
		path      string
		valueName string
		success   bool
		err       error
	}{
		{
			name:      "successful read",
			rootKey:   "HKLM",
			path:      "SOFTWARE\\Microsoft\\Windows",
			valueName: "TestValue",
			success:   true,
			err:       nil,
		},
		{
			name:      "failed read",
			rootKey:   "HKLM",
			path:      "SOFTWARE\\Invalid\\Path",
			valueName: "TestValue",
			success:   false,
			err:       &RegistryError{Op: "OpenKey", Key: "SOFTWARE\\Invalid\\Path", Err: os.ErrNotExist},
		},
		{
			name:      "read with empty value name",
			rootKey:   "HKCU",
			path:      "SOFTWARE\\Test",
			valueName: "",
			success:   true,
			err:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialStats := logger.GetStats()
			logger.LogRegistryRead(tt.rootKey, tt.path, tt.valueName, tt.success, tt.err)

			stats := logger.GetStats()
			if stats.TotalEvents <= initialStats.TotalEvents {
				t.Error("LogRegistryRead() did not increment total events")
			}

			if stats.AccessEvents <= initialStats.AccessEvents {
				t.Error("LogRegistryRead() did not increment access events")
			}
		})
	}

	// Verify log file was written
	fileInfo, err := os.Stat(auditLogPath)
	if err != nil {
		t.Errorf("Audit log file not created: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Error("Audit log file is empty")
	}
}

// TestAuditLogger_LogSecurityEvent tests logging security events
func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_security.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name      string
		eventType AuditEventType
		resource  string
		reason    string
		details   map[string]interface{}
	}{
		{
			name:      "path traversal attempt",
			eventType: AuditEventPathTraversal,
			resource:  "SOFTWARE\\..\\System",
			reason:    "path traversal detected",
			details: map[string]interface{}{
				"attack_type": "path_traversal",
				"path":        "SOFTWARE\\..\\System",
			},
		},
		{
			name:      "injection attempt",
			eventType: AuditEventInjectionAttempt,
			resource:  "malicious_input",
			reason:    "null byte detected",
			details: map[string]interface{}{
				"attack_type": "injection",
				"input":       "Test\x00Value",
			},
		},
		{
			name:      "policy violation",
			eventType: AuditEventPolicyViolation,
			resource:  "SECURITY\\Policy\\Secrets",
			reason:    "access to denied path",
			details: map[string]interface{}{
				"policy": "deny_list",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialStats := logger.GetStats()
			logger.LogSecurityEvent(tt.eventType, tt.resource, tt.reason, tt.details)

			stats := logger.GetStats()
			if stats.SecurityEvents <= initialStats.SecurityEvents {
				t.Error("LogSecurityEvent() did not increment security events")
			}
		})
	}
}

// TestAuditLogger_LogValidationFailure tests logging validation failures
func TestAuditLogger_LogValidationFailure(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_validation.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name   string
		field  string
		value  string
		reason string
	}{
		{
			name:   "invalid root key",
			field:  "RootKey",
			value:  "INVALID",
			reason: "not a valid registry root key",
		},
		{
			name:   "path too long",
			field:  "Path",
			value:  string(make([]byte, MaxRegistryPathLength+1)),
			reason: "path exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialStats := logger.GetStats()
			logger.LogValidationFailure(tt.field, tt.value, tt.reason)

			stats := logger.GetStats()
			if stats.ValidationFails <= initialStats.ValidationFails {
				t.Error("LogValidationFailure() did not increment validation failures")
			}
		})
	}
}

// TestAuditLogger_LogReportGeneration tests logging report generation
func TestAuditLogger_LogReportGeneration(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_reports.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name       string
		reportName string
		success    bool
		err        error
	}{
		{
			name:       "successful report",
			reportName: "NIST_800_171_compliance",
			success:    true,
			err:        nil,
		},
		{
			name:       "failed report",
			reportName: "Invalid_Report",
			success:    false,
			err:        os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now().Add(-5 * time.Second)
			logger.LogReportGeneration(tt.reportName, startTime, tt.success, tt.err)

			stats := logger.GetStats()
			if stats.TotalEvents == 0 {
				t.Error("LogReportGeneration() did not update statistics")
			}
		})
	}
}

// TestAuditLogger_LogSystemEvent tests logging system events
func TestAuditLogger_LogSystemEvent(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_system.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name      string
		eventType AuditEventType
		message   string
		details   map[string]interface{}
	}{
		{
			name:      "startup event",
			eventType: AuditEventStartup,
			message:   "Application started",
			details: map[string]interface{}{
				"version": "1.1.0",
			},
		},
		{
			name:      "shutdown event",
			eventType: AuditEventShutdown,
			message:   "Application shutdown",
			details:   nil,
		},
		{
			name:      "error event",
			eventType: AuditEventError,
			message:   "Critical error occurred",
			details: map[string]interface{}{
				"error_code": 500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialStats := logger.GetStats()
			logger.LogSystemEvent(tt.eventType, tt.message, tt.details)

			stats := logger.GetStats()
			if stats.TotalEvents <= initialStats.TotalEvents {
				t.Error("LogSystemEvent() did not increment total events")
			}

			// Error events should increment error count
			if tt.eventType == AuditEventError && stats.Errors <= initialStats.Errors {
				t.Error("LogSystemEvent() did not increment error count for error event")
			}
		})
	}
}

// TestAuditLogger_GetStats tests retrieving audit statistics
func TestAuditLogger_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_stats.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Log various events
	logger.LogAccess("test_resource", "read", true)
	logger.LogAccess("denied_resource", "write", false)
	logger.LogRegistryRead("HKLM", "SOFTWARE\\Test", "Value", true, nil)
	logger.LogValidationFailure("test_field", "test_value", "test_reason")
	logger.LogSystemEvent(AuditEventError, "test error", nil)

	stats := logger.GetStats()

	if stats.TotalEvents != 5 {
		t.Errorf("GetStats() TotalEvents = %d, want 5", stats.TotalEvents)
	}

	if stats.AccessEvents != 3 {
		t.Errorf("GetStats() AccessEvents = %d, want 3 (LogAccess x2 + LogRegistryRead x1)", stats.AccessEvents)
	}

	if stats.SecurityEvents != 1 {
		t.Errorf("GetStats() SecurityEvents = %d, want 1 (LogValidationFailure)", stats.SecurityEvents)
	}

	if stats.DeniedAccess != 2 {
		t.Errorf("GetStats() DeniedAccess = %d, want 2 (1 from LogAccess denied + 1 from LogValidationFailure)", stats.DeniedAccess)
	}

	if stats.ValidationFails != 1 {
		t.Errorf("GetStats() ValidationFails = %d, want 1", stats.ValidationFails)
	}

	if stats.Errors != 1 {
		t.Errorf("GetStats() Errors = %d, want 1", stats.Errors)
	}

	if stats.LastEventTime.IsZero() {
		t.Error("GetStats() LastEventTime is zero")
	}
}

// TestAuditLogger_DisabledLogger tests that disabled logger doesn't log
func TestAuditLogger_DisabledLogger(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_disabled.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, false)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}

	if logger.IsEnabled() {
		t.Error("Disabled logger reports as enabled")
	}

	// Try to log events - these should be no-ops
	logger.LogAccess("test_resource", "read", true)
	logger.LogRegistryRead("HKLM", "SOFTWARE\\Test", "TestValue", true, nil)
	logger.LogSecurityEvent(AuditEventPathTraversal, "test", "reason", nil)

	stats := logger.GetStats()
	if stats.TotalEvents != 0 {
		t.Errorf("Disabled logger recorded events: TotalEvents = %d, want 0", stats.TotalEvents)
	}

	// Verify no log file was created (disabled logger shouldn't create file)
	if _, err := os.Stat(auditLogPath); !os.IsNotExist(err) {
		t.Error("Disabled logger created a log file")
	}
}

// TestAuditLogger_ConcurrentAccess tests thread safety
func TestAuditLogger_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	auditLogPath := filepath.Join(tempDir, "audit_concurrent.log")

	logger, err := NewAuditLoggerWithFile(auditLogPath, true)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Launch multiple goroutines to log concurrently
	const numGoroutines = 10
	const eventsPerGoroutine = 100

	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < eventsPerGoroutine; j++ {
				logger.LogAccess("test_resource", "read", true)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	stats := logger.GetStats()
	expectedEvents := numGoroutines * eventsPerGoroutine
	if stats.TotalEvents != int64(expectedEvents) {
		t.Errorf("Concurrent logging: TotalEvents = %d, want %d", stats.TotalEvents, expectedEvents)
	}
}

// TestGenerateSessionID tests session ID generation
func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()

	// Sleep to ensure time changes for next ID
	time.Sleep(2 * time.Millisecond)

	id2 := generateSessionID()

	if id1 == "" {
		t.Error("generateSessionID() returned empty string")
	}

	if id1 == id2 {
		t.Error("generateSessionID() returned duplicate IDs")
	}
}

// TestGetCurrentUser tests user retrieval
func TestGetCurrentUser(t *testing.T) {
	user := getCurrentUser()

	if user == "" {
		t.Error("getCurrentUser() returned empty string")
	}

	if user == "unknown" {
		// This is acceptable if no env vars are set, but log a warning
		t.Log("Warning: getCurrentUser() returned 'unknown' - no USERNAME or USER env var set")
	}
}

// TestGetCallerInfo tests caller info retrieval
func TestGetCallerInfo(t *testing.T) {
	info := getCallerInfo(0)

	if info == "" {
		t.Error("getCallerInfo() returned empty string")
	}

	if info == "unknown" {
		t.Error("getCallerInfo() returned 'unknown'")
	}

	// Should contain a filename and line number
	if len(info) < 3 {
		t.Errorf("getCallerInfo() returned invalid format: %s", info)
	}
}

// TestNewAuditContext tests audit context creation
func TestNewAuditContext(t *testing.T) {
	ctx := NewAuditContext(nil)

	if ctx == nil {
		t.Error("NewAuditContext() returned nil")
	}

	if ctx.RequestID == "" {
		t.Error("NewAuditContext() did not generate RequestID")
	}

	if ctx.StartTime.IsZero() {
		t.Error("NewAuditContext() did not set StartTime")
	}

	// Test Elapsed()
	time.Sleep(10 * time.Millisecond)
	elapsed := ctx.Elapsed()
	if elapsed < 10*time.Millisecond {
		t.Errorf("AuditContext.Elapsed() = %v, want >= 10ms", elapsed)
	}
}
