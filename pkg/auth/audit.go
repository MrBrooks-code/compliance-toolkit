package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AuditLogger handles authentication audit logging for security monitoring
type AuditLogger struct {
	db *sql.DB
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *sql.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// EventType represents different types of authentication events
type EventType string

const (
	EventLogin        EventType = "login"
	EventLogout       EventType = "logout"
	EventRefresh      EventType = "refresh"
	EventFailedLogin  EventType = "failed_login"
	EventTokenRevoked EventType = "token_revoked"
	EventPasswordChange EventType = "password_change"
	EventAccountLocked EventType = "account_locked"
	EventMFAEnabled   EventType = "mfa_enabled"
	EventMFADisabled  EventType = "mfa_disabled"
)

// AuthMethod represents the authentication method used
type AuthMethod string

const (
	AuthMethodJWT     AuthMethod = "jwt"
	AuthMethodSession AuthMethod = "session"
	AuthMethodAPIKey  AuthMethod = "api_key"
)

// AuditEvent represents an authentication audit event
type AuditEvent struct {
	UserID        int
	Username      string
	EventType     EventType
	AuthMethod    AuthMethod
	IPAddress     string
	UserAgent     string
	Success       bool
	FailureReason string
	Metadata      map[string]interface{}
}

// Log logs an authentication event to the audit log
func (a *AuditLogger) Log(ctx context.Context, event AuditEvent) error {
	// Serialize metadata to JSON
	var metadataJSON []byte
	var err error
	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO auth_audit_log (
			user_id, username, event_type, auth_method,
			ip_address, user_agent, success, failure_reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	var userIDVal interface{}
	if event.UserID == 0 {
		userIDVal = nil
	} else {
		userIDVal = event.UserID
	}

	_, err = a.db.ExecContext(ctx, query,
		userIDVal,
		event.Username,
		event.EventType,
		event.AuthMethod,
		event.IPAddress,
		event.UserAgent,
		event.Success,
		event.FailureReason,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to log audit event: %w", err)
	}

	return nil
}

// LogLogin logs a successful login event
func (a *AuditLogger) LogLogin(ctx context.Context, userID int, username string, authMethod AuthMethod, ipAddress, userAgent string) error {
	return a.Log(ctx, AuditEvent{
		UserID:     userID,
		Username:   username,
		EventType:  EventLogin,
		AuthMethod: authMethod,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    true,
	})
}

// LogFailedLogin logs a failed login attempt
func (a *AuditLogger) LogFailedLogin(ctx context.Context, username string, reason string, ipAddress, userAgent string) error {
	return a.Log(ctx, AuditEvent{
		Username:      username,
		EventType:     EventFailedLogin,
		AuthMethod:    AuthMethodJWT,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Success:       false,
		FailureReason: reason,
	})
}

// LogLogout logs a logout event
func (a *AuditLogger) LogLogout(ctx context.Context, userID int, username string, ipAddress, userAgent string) error {
	return a.Log(ctx, AuditEvent{
		UserID:     userID,
		Username:   username,
		EventType:  EventLogout,
		AuthMethod: AuthMethodJWT,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    true,
	})
}

// LogTokenRefresh logs a token refresh event
func (a *AuditLogger) LogTokenRefresh(ctx context.Context, userID int, username string, ipAddress, userAgent string) error {
	return a.Log(ctx, AuditEvent{
		UserID:     userID,
		Username:   username,
		EventType:  EventRefresh,
		AuthMethod: AuthMethodJWT,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    true,
	})
}

// LogTokenRevoked logs a token revocation event
func (a *AuditLogger) LogTokenRevoked(ctx context.Context, userID int, username string, reason string) error {
	return a.Log(ctx, AuditEvent{
		UserID:     userID,
		Username:   username,
		EventType:  EventTokenRevoked,
		AuthMethod: AuthMethodJWT,
		Success:    true,
		Metadata: map[string]interface{}{
			"revocation_reason": reason,
		},
	})
}

// AuditLogEntry represents a record from the audit log
type AuditLogEntry struct {
	ID            int
	UserID        *int
	Username      string
	EventType     string
	AuthMethod    string
	IPAddress     string
	UserAgent     string
	Success       bool
	FailureReason string
	Timestamp     time.Time
	Metadata      map[string]interface{}
}

// GetUserAuditLog retrieves audit log entries for a specific user
func (a *AuditLogger) GetUserAuditLog(ctx context.Context, userID int, limit int) ([]AuditLogEntry, error) {
	query := `
		SELECT id, user_id, username, event_type, auth_method,
		       ip_address, user_agent, success, failure_reason, timestamp, metadata
		FROM auth_audit_log
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := a.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	return a.scanAuditLogEntries(rows)
}

// GetRecentFailedLogins retrieves recent failed login attempts
func (a *AuditLogger) GetRecentFailedLogins(ctx context.Context, username string, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) FROM auth_audit_log
		WHERE username = $1 AND event_type = $2 AND success = false AND timestamp > $3
	`

	var count int
	err := a.db.QueryRowContext(ctx, query, username, EventFailedLogin, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count failed logins: %w", err)
	}

	return count, nil
}

// GetAuditLogByTimeRange retrieves audit log entries within a time range
func (a *AuditLogger) GetAuditLogByTimeRange(ctx context.Context, start, end time.Time, limit int) ([]AuditLogEntry, error) {
	query := `
		SELECT id, user_id, username, event_type, auth_method,
		       ip_address, user_agent, success, failure_reason, timestamp, metadata
		FROM auth_audit_log
		WHERE timestamp BETWEEN $1 AND $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := a.db.QueryContext(ctx, query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	return a.scanAuditLogEntries(rows)
}

// scanAuditLogEntries scans rows into AuditLogEntry structs
func (a *AuditLogger) scanAuditLogEntries(rows *sql.Rows) ([]AuditLogEntry, error) {
	var entries []AuditLogEntry

	for rows.Next() {
		var entry AuditLogEntry
		var userID sql.NullInt64
		var failureReason sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&entry.ID,
			&userID,
			&entry.Username,
			&entry.EventType,
			&entry.AuthMethod,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.Success,
			&failureReason,
			&entry.Timestamp,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log entry: %w", err)
		}

		// Convert nullable fields
		if userID.Valid {
			uid := int(userID.Int64)
			entry.UserID = &uid
		}
		if failureReason.Valid {
			entry.FailureReason = failureReason.String
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &entry.Metadata); err != nil {
				// Non-fatal, just log
				fmt.Printf("Warning: failed to unmarshal metadata: %v\n", err)
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// CleanupOldEntries removes audit log entries older than the specified duration
func (a *AuditLogger) CleanupOldEntries(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM auth_audit_log
		WHERE timestamp < $1
	`

	cutoffTime := time.Now().Add(-olderThan)

	result, err := a.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old audit entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
