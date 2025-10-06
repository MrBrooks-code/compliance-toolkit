package pkg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// AuditEventType categorizes audit events
type AuditEventType string

const (
	// Access events
	AuditEventRegistryRead     AuditEventType = "registry.read"
	AuditEventRegistryReadAll  AuditEventType = "registry.read_all"
	AuditEventConfigLoad       AuditEventType = "config.load"
	AuditEventReportGenerate   AuditEventType = "report.generate"
	AuditEventReportComplete   AuditEventType = "report.complete"

	// Security events
	AuditEventAccessDenied     AuditEventType = "security.access_denied"
	AuditEventValidationFailed AuditEventType = "security.validation_failed"
	AuditEventPathTraversal    AuditEventType = "security.path_traversal"
	AuditEventInjectionAttempt AuditEventType = "security.injection_attempt"
	AuditEventPolicyViolation  AuditEventType = "security.policy_violation"

	// System events
	AuditEventStartup          AuditEventType = "system.startup"
	AuditEventShutdown         AuditEventType = "system.shutdown"
	AuditEventError            AuditEventType = "system.error"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	User        string                 `json:"user,omitempty"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Result      string                 `json:"result"` // "success", "denied", "failed", "error"
	Details     map[string]interface{} `json:"details,omitempty"`
	Severity    string                 `json:"severity"` // "info", "warning", "error", "critical"
	Source      string                 `json:"source"`   // File and line number
	SessionID   string                 `json:"session_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// AuditLogger provides structured audit logging for compliance and forensics
type AuditLogger struct {
	logger    *slog.Logger
	enabled   bool
	mu        sync.RWMutex
	sessionID string
	stats     AuditStats
	logFile   *os.File // Underlying file for proper cleanup
}

// AuditStats tracks audit logging statistics
type AuditStats struct {
	TotalEvents      int64
	AccessEvents     int64
	SecurityEvents   int64
	DeniedAccess     int64
	ValidationFails  int64
	Errors           int64
	LastEventTime    time.Time
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *slog.Logger, enabled bool) *AuditLogger {
	return &AuditLogger{
		logger:    logger,
		enabled:   enabled,
		sessionID: generateSessionID(),
		stats:     AuditStats{},
	}
}

// NewAuditLoggerWithFile creates an audit logger with dedicated file output
func NewAuditLoggerWithFile(auditLogPath string, enabled bool) (*AuditLogger, error) {
	if !enabled {
		// Return disabled logger
		return &AuditLogger{
			logger:    slog.Default(),
			enabled:   false,
			sessionID: generateSessionID(),
		}, nil
	}

	// Create audit log directory
	auditDir := filepath.Dir(auditLogPath)
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open audit log file
	file, err := os.OpenFile(auditLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	// Create JSON handler for structured audit logs
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add custom formatting for audit logs
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339Nano)),
				}
			}
			return a
		},
	})

	logger := slog.New(handler)

	return &AuditLogger{
		logger:    logger,
		enabled:   enabled,
		sessionID: generateSessionID(),
		stats:     AuditStats{},
		logFile:   file, // Store file handle for cleanup
	}, nil
}

// Close closes the audit log file if one is open
func (a *AuditLogger) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.logFile != nil {
		err := a.logFile.Close()
		a.logFile = nil
		return err
	}
	return nil
}

// IsEnabled returns whether audit logging is enabled
func (a *AuditLogger) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// LogAccess logs a resource access event
func (a *AuditLogger) LogAccess(resource, action string, allowed bool) {
	if !a.IsEnabled() {
		return
	}

	result := "success"
	severity := "info"
	if !allowed {
		result = "denied"
		severity = "warning"
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventRegistryRead,
		User:      getCurrentUser(),
		Resource:  resource,
		Action:    action,
		Result:    result,
		Severity:  severity,
		Source:    getCallerInfo(2),
		SessionID: a.sessionID,
	}

	a.logEvent(event)
	a.updateStats(event)
}

// LogRegistryRead logs a registry read operation
func (a *AuditLogger) LogRegistryRead(rootKey, path, valueName string, success bool, err error) {
	if !a.IsEnabled() {
		return
	}

	resource := fmt.Sprintf("%s\\%s", rootKey, path)
	if valueName != "" {
		resource = fmt.Sprintf("%s\\%s", resource, valueName)
	}

	result := "success"
	severity := "info"
	errMsg := ""

	if !success {
		result = "failed"
		severity = "warning"
		if err != nil {
			errMsg = err.Error()
		}
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventRegistryRead,
		User:      getCurrentUser(),
		Resource:  resource,
		Action:    "read",
		Result:    result,
		Severity:  severity,
		Source:    getCallerInfo(2),
		SessionID: a.sessionID,
		Error:     errMsg,
		Details: map[string]interface{}{
			"root_key":   rootKey,
			"path":       path,
			"value_name": valueName,
		},
	}

	a.logEvent(event)
	a.updateStats(event)
}

// LogSecurityEvent logs a security-related event
func (a *AuditLogger) LogSecurityEvent(eventType AuditEventType, resource, reason string, details map[string]interface{}) {
	if !a.IsEnabled() {
		return
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      getCurrentUser(),
		Resource:  resource,
		Action:    "validate",
		Result:    "denied",
		Severity:  "warning",
		Source:    getCallerInfo(2),
		SessionID: a.sessionID,
		Error:     reason,
		Details:   details,
	}

	a.logEvent(event)
	a.updateStats(event)
}

// LogValidationFailure logs a validation failure
func (a *AuditLogger) LogValidationFailure(field, value, reason string) {
	if !a.IsEnabled() {
		return
	}

	details := map[string]interface{}{
		"field":  field,
		"value":  value,
		"reason": reason,
	}

	a.LogSecurityEvent(AuditEventValidationFailed, field, reason, details)
}

// LogPathTraversal logs a path traversal attempt
func (a *AuditLogger) LogPathTraversal(path string) {
	if !a.IsEnabled() {
		return
	}

	details := map[string]interface{}{
		"path":        path,
		"attack_type": "path_traversal",
	}

	a.LogSecurityEvent(AuditEventPathTraversal, path, "path traversal attempt detected", details)
}

// LogInjectionAttempt logs an injection attempt
func (a *AuditLogger) LogInjectionAttempt(input, inputType string) {
	if !a.IsEnabled() {
		return
	}

	details := map[string]interface{}{
		"input":       input,
		"input_type":  inputType,
		"attack_type": "injection",
	}

	a.LogSecurityEvent(AuditEventInjectionAttempt, inputType, "injection attempt detected", details)
}

// LogPolicyViolation logs a security policy violation
func (a *AuditLogger) LogPolicyViolation(resource, policy, reason string) {
	if !a.IsEnabled() {
		return
	}

	details := map[string]interface{}{
		"policy": policy,
		"reason": reason,
	}

	a.LogSecurityEvent(AuditEventPolicyViolation, resource, reason, details)
}

// LogReportGeneration logs report generation activity
func (a *AuditLogger) LogReportGeneration(reportName string, startTime time.Time, success bool, err error) {
	if !a.IsEnabled() {
		return
	}

	duration := time.Since(startTime)
	result := "success"
	severity := "info"
	errMsg := ""
	eventType := AuditEventReportComplete

	if !success {
		result = "failed"
		severity = "error"
		if err != nil {
			errMsg = err.Error()
		}
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      getCurrentUser(),
		Resource:  reportName,
		Action:    "generate",
		Result:    result,
		Severity:  severity,
		Source:    getCallerInfo(2),
		SessionID: a.sessionID,
		Duration:  duration,
		Error:     errMsg,
		Details: map[string]interface{}{
			"report_name":     reportName,
			"duration_ms":     duration.Milliseconds(),
			"duration_human":  duration.String(),
		},
	}

	a.logEvent(event)
	a.updateStats(event)
}

// LogSystemEvent logs system-level events
func (a *AuditLogger) LogSystemEvent(eventType AuditEventType, message string, details map[string]interface{}) {
	if !a.IsEnabled() {
		return
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      getCurrentUser(),
		Resource:  "system",
		Action:    string(eventType),
		Result:    "success",
		Severity:  "info",
		Source:    getCallerInfo(2),
		SessionID: a.sessionID,
		Details:   details,
	}

	if eventType == AuditEventError {
		event.Result = "error"
		event.Severity = "error"
		event.Error = message
	}

	a.logEvent(event)
	a.updateStats(event)
}

// GetStats returns current audit statistics
func (a *AuditLogger) GetStats() AuditStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// logEvent writes the audit event to the log
func (a *AuditLogger) logEvent(event AuditEvent) {
	a.logger.Info("audit",
		"timestamp", event.Timestamp.Format(time.RFC3339Nano),
		"event_type", event.EventType,
		"user", event.User,
		"resource", event.Resource,
		"action", event.Action,
		"result", event.Result,
		"severity", event.Severity,
		"source", event.Source,
		"session_id", event.SessionID,
		"request_id", event.RequestID,
		"duration_ms", event.Duration.Milliseconds(),
		"error", event.Error,
		"details", event.Details,
	)
}

// updateStats updates internal statistics
func (a *AuditLogger) updateStats(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.stats.TotalEvents++
	a.stats.LastEventTime = event.Timestamp

	// Categorize by event type
	switch event.EventType {
	case AuditEventRegistryRead, AuditEventRegistryReadAll, AuditEventConfigLoad:
		a.stats.AccessEvents++
	case AuditEventAccessDenied, AuditEventValidationFailed, AuditEventPathTraversal,
		AuditEventInjectionAttempt, AuditEventPolicyViolation:
		a.stats.SecurityEvents++
	}

	// Track specific conditions
	if event.Result == "denied" {
		a.stats.DeniedAccess++
	}
	if event.EventType == AuditEventValidationFailed {
		a.stats.ValidationFails++
	}
	if event.Severity == "error" || event.Severity == "critical" {
		a.stats.Errors++
	}
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("%d-%s-%d",
		os.Getpid(),
		time.Now().Format("20060102150405"),
		time.Now().UnixNano()%1000000,
	)
}

// getCurrentUser returns the current username
func getCurrentUser() string {
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "unknown"
}

// getCallerInfo returns the caller's file and line number
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	// Return just the filename, not full path
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// AuditContext wraps a context with audit metadata
type AuditContext struct {
	context.Context
	RequestID string
	StartTime time.Time
}

// NewAuditContext creates a new audit context
func NewAuditContext(ctx context.Context) *AuditContext {
	return &AuditContext{
		Context:   ctx,
		RequestID: generateRequestID(),
		StartTime: time.Now(),
	}
}

// generateRequestID creates a unique request identifier
func generateRequestID() string {
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), os.Getpid())
}

// Elapsed returns the time elapsed since the context was created
func (ac *AuditContext) Elapsed() time.Duration {
	return time.Since(ac.StartTime)
}
