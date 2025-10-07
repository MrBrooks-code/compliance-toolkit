package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"compliancetoolkit/pkg/api"
)

// Database handles all database operations
type Database struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewDatabase creates and initializes a new database connection
func NewDatabase(config DatabaseSettings, logger *slog.Logger) (*Database, error) {
	if config.Type != "sqlite" {
		return nil, fmt.Errorf("unsupported database type: %s (only sqlite supported)", config.Type)
	}

	// Ensure directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := database.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("Database initialized", "path", config.Path)
	return database, nil
}

// initSchema creates database tables if they don't exist
func (d *Database) initSchema() error {
	schema := `
	-- Clients table
	CREATE TABLE IF NOT EXISTS clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id TEXT UNIQUE NOT NULL,
		hostname TEXT NOT NULL,
		first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		os_version TEXT,
		build_number TEXT,
		architecture TEXT,
		domain TEXT,
		ip_address TEXT,
		mac_address TEXT,
		status TEXT DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Submissions table
	CREATE TABLE IF NOT EXISTS submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		submission_id TEXT UNIQUE NOT NULL,
		client_id TEXT NOT NULL,
		hostname TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		report_type TEXT NOT NULL,
		report_version TEXT,
		overall_status TEXT,
		total_checks INTEGER DEFAULT 0,
		passed_checks INTEGER DEFAULT 0,
		failed_checks INTEGER DEFAULT 0,
		warning_checks INTEGER DEFAULT 0,
		error_checks INTEGER DEFAULT 0,
		compliance_data TEXT,  -- JSON
		evidence TEXT,         -- JSON array
		system_info TEXT,      -- JSON
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (client_id) REFERENCES clients(client_id)
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_submissions_client_id ON submissions(client_id);
	CREATE INDEX IF NOT EXISTS idx_submissions_timestamp ON submissions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_submissions_report_type ON submissions(report_type);
	CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);
	`

	if _, err := d.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Ping checks if the database connection is alive
func (d *Database) Ping() error {
	return d.db.Ping()
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// SaveSubmission saves a compliance submission to the database
func (d *Database) SaveSubmission(submission *api.ComplianceSubmission) error {
	// Marshal complex fields to JSON
	complianceData, err := json.Marshal(submission.Compliance)
	if err != nil {
		return fmt.Errorf("failed to marshal compliance data: %w", err)
	}

	evidence, err := json.Marshal(submission.Evidence)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}

	systemInfo, err := json.Marshal(submission.SystemInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal system info: %w", err)
	}

	query := `
		INSERT INTO submissions (
			submission_id, client_id, hostname, timestamp, report_type, report_version,
			overall_status, total_checks, passed_checks, failed_checks, warning_checks, error_checks,
			compliance_data, evidence, system_info
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.Exec(query,
		submission.SubmissionID,
		submission.ClientID,
		submission.Hostname,
		submission.Timestamp,
		submission.ReportType,
		submission.ReportVersion,
		submission.Compliance.OverallStatus,
		submission.Compliance.TotalChecks,
		submission.Compliance.PassedChecks,
		submission.Compliance.FailedChecks,
		submission.Compliance.WarningChecks,
		submission.Compliance.ErrorChecks,
		complianceData,
		evidence,
		systemInfo,
	)

	if err != nil {
		return fmt.Errorf("failed to insert submission: %w", err)
	}

	d.logger.Debug("Saved submission", "submission_id", submission.SubmissionID)
	return nil
}

// GetSubmission retrieves a submission by ID
func (d *Database) GetSubmission(submissionID string) (*api.ComplianceSubmission, error) {
	query := `
		SELECT submission_id, client_id, hostname, timestamp, report_type, report_version,
		       compliance_data, evidence, system_info
		FROM submissions
		WHERE submission_id = ?
	`

	var submission api.ComplianceSubmission
	var complianceData, evidence, systemInfo string

	err := d.db.QueryRow(query, submissionID).Scan(
		&submission.SubmissionID,
		&submission.ClientID,
		&submission.Hostname,
		&submission.Timestamp,
		&submission.ReportType,
		&submission.ReportVersion,
		&complianceData,
		&evidence,
		&systemInfo,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("submission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query submission: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal([]byte(complianceData), &submission.Compliance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compliance data: %w", err)
	}
	if err := json.Unmarshal([]byte(evidence), &submission.Evidence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evidence: %w", err)
	}
	if err := json.Unmarshal([]byte(systemInfo), &submission.SystemInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal system info: %w", err)
	}

	return &submission, nil
}

// RegisterClient registers or updates a client
func (d *Database) RegisterClient(registration *api.ClientRegistration) error {
	query := `
		INSERT INTO clients (
			client_id, hostname, os_version, build_number, architecture,
			domain, ip_address, mac_address, first_seen, last_seen
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(client_id) DO UPDATE SET
			hostname = excluded.hostname,
			os_version = excluded.os_version,
			build_number = excluded.build_number,
			architecture = excluded.architecture,
			domain = excluded.domain,
			ip_address = excluded.ip_address,
			mac_address = excluded.mac_address,
			last_seen = CURRENT_TIMESTAMP
	`

	_, err := d.db.Exec(query,
		registration.ClientID,
		registration.Hostname,
		registration.SystemInfo.OSVersion,
		registration.SystemInfo.BuildNumber,
		registration.SystemInfo.Architecture,
		registration.SystemInfo.Domain,
		registration.SystemInfo.IPAddress,
		registration.SystemInfo.MacAddress,
	)

	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}

	d.logger.Debug("Registered client", "client_id", registration.ClientID)
	return nil
}

// UpdateClientLastSeen updates the last_seen timestamp and system info for a client
func (d *Database) UpdateClientLastSeen(clientID, hostname string, systemInfo *api.SystemInfo) error {
	query := `
		INSERT INTO clients (
			client_id, hostname, os_version, build_number, architecture,
			domain, ip_address, mac_address, first_seen, last_seen
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(client_id) DO UPDATE SET
			hostname = excluded.hostname,
			os_version = excluded.os_version,
			build_number = excluded.build_number,
			architecture = excluded.architecture,
			domain = excluded.domain,
			ip_address = excluded.ip_address,
			mac_address = excluded.mac_address,
			last_seen = CURRENT_TIMESTAMP
	`

	var osVersion, buildNumber, architecture, domain, ipAddress, macAddress string
	if systemInfo != nil {
		osVersion = systemInfo.OSVersion
		buildNumber = systemInfo.BuildNumber
		architecture = systemInfo.Architecture
		domain = systemInfo.Domain
		ipAddress = systemInfo.IPAddress
		macAddress = systemInfo.MacAddress
	}

	_, err := d.db.Exec(query, clientID, hostname, osVersion, buildNumber, architecture, domain, ipAddress, macAddress)
	if err != nil {
		return fmt.Errorf("failed to update client last_seen: %w", err)
	}

	return nil
}

// ListClients returns all registered clients
func (d *Database) ListClients() ([]api.ClientInfo, error) {
	query := `
		SELECT
			c.id, c.client_id, c.hostname, c.first_seen, c.last_seen, c.status,
			c.os_version, c.build_number, c.architecture, c.domain, c.ip_address, c.mac_address,
			(SELECT submission_id FROM submissions WHERE client_id = c.client_id ORDER BY timestamp DESC LIMIT 1) as last_submission,
			(SELECT COUNT(*) FROM submissions WHERE client_id = c.client_id AND overall_status = 'compliant') * 100.0 /
			NULLIF((SELECT COUNT(*) FROM submissions WHERE client_id = c.client_id), 0) as compliance_score
		FROM clients c
		ORDER BY c.last_seen DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query clients: %w", err)
	}
	defer rows.Close()

	var clients []api.ClientInfo
	for rows.Next() {
		var client api.ClientInfo
		var lastSubmission sql.NullString
		var complianceScore sql.NullFloat64

		// Use NullString for all nullable fields
		var osVersion, buildNumber, architecture, domain, ipAddress, macAddress sql.NullString

		err := rows.Scan(
			&client.ID,
			&client.ClientID,
			&client.Hostname,
			&client.FirstSeen,
			&client.LastSeen,
			&client.Status,
			&osVersion,
			&buildNumber,
			&architecture,
			&domain,
			&ipAddress,
			&macAddress,
			&lastSubmission,
			&complianceScore,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		// Populate system info from nullable fields
		if osVersion.Valid {
			client.SystemInfo.OSVersion = osVersion.String
		}
		if buildNumber.Valid {
			client.SystemInfo.BuildNumber = buildNumber.String
		}
		if architecture.Valid {
			client.SystemInfo.Architecture = architecture.String
		}
		if domain.Valid {
			client.SystemInfo.Domain = domain.String
		}
		if ipAddress.Valid {
			client.SystemInfo.IPAddress = ipAddress.String
		}
		if macAddress.Valid {
			client.SystemInfo.MacAddress = macAddress.String
		}
		if lastSubmission.Valid {
			client.LastSubmission = lastSubmission.String
		}
		if complianceScore.Valid {
			client.ComplianceScore = complianceScore.Float64
		}

		clients = append(clients, client)
	}

	return clients, nil
}

// GetDashboardSummary returns summary data for the dashboard
func (d *Database) GetDashboardSummary() (*api.DashboardSummary, error) {
	summary := &api.DashboardSummary{
		ComplianceByType: make(map[string]api.ComplianceStats),
	}

	// Get total and active clients
	err := d.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN last_seen > datetime('now', '-24 hours') THEN 1 END) as active
		FROM clients
	`).Scan(&summary.TotalClients, &summary.ActiveClients)

	if err != nil {
		return nil, fmt.Errorf("failed to get client counts: %w", err)
	}

	// Get compliant clients (last submission was compliant)
	err = d.db.QueryRow(`
		SELECT COUNT(DISTINCT client_id)
		FROM submissions s1
		WHERE overall_status = 'compliant'
		AND timestamp = (
			SELECT MAX(timestamp)
			FROM submissions s2
			WHERE s2.client_id = s1.client_id
		)
	`).Scan(&summary.CompliantClients)

	if err != nil {
		return nil, fmt.Errorf("failed to get compliant client count: %w", err)
	}

	// Get recent submissions
	rows, err := d.db.Query(`
		SELECT submission_id, client_id, hostname, timestamp, report_type,
		       overall_status, passed_checks, failed_checks
		FROM submissions
		ORDER BY timestamp DESC
		LIMIT 10
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to query recent submissions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sub api.SubmissionSummary
		err := rows.Scan(
			&sub.SubmissionID,
			&sub.ClientID,
			&sub.Hostname,
			&sub.Timestamp,
			&sub.ReportType,
			&sub.OverallStatus,
			&sub.PassedChecks,
			&sub.FailedChecks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		summary.RecentSubmissions = append(summary.RecentSubmissions, sub)
	}

	// Get compliance stats by report type
	statsRows, err := d.db.Query(`
		SELECT
			report_type,
			COUNT(*) as total_submissions,
			AVG(passed_checks * 100.0 / NULLIF(total_checks, 0)) as avg_score,
			SUM(CASE WHEN overall_status = 'compliant' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as pass_rate,
			SUM(CASE WHEN overall_status != 'compliant' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as fail_rate
		FROM submissions
		GROUP BY report_type
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to query compliance stats: %w", err)
	}
	defer statsRows.Close()

	for statsRows.Next() {
		var reportType string
		var stats api.ComplianceStats
		err := statsRows.Scan(
			&reportType,
			&stats.TotalSubmissions,
			&stats.AverageScore,
			&stats.PassRate,
			&stats.FailRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		summary.ComplianceByType[reportType] = stats
	}

	return summary, nil
}

// GetClient retrieves detailed information for a specific client
func (d *Database) GetClient(clientID string) (*api.ClientInfo, error) {
	query := `
		SELECT
			c.id, c.client_id, c.hostname, c.first_seen, c.last_seen, c.status,
			c.os_version, c.build_number, c.architecture, c.domain, c.ip_address, c.mac_address,
			(SELECT submission_id FROM submissions WHERE client_id = c.client_id ORDER BY timestamp DESC LIMIT 1) as last_submission,
			(SELECT COUNT(*) FROM submissions WHERE client_id = c.client_id AND overall_status = 'compliant') * 100.0 /
			NULLIF((SELECT COUNT(*) FROM submissions WHERE client_id = c.client_id), 0) as compliance_score
		FROM clients c
		WHERE c.client_id = ?
	`

	var client api.ClientInfo
	var lastSubmission sql.NullString
	var complianceScore sql.NullFloat64
	var osVersion, buildNumber, architecture, domain, ipAddress, macAddress sql.NullString

	err := d.db.QueryRow(query, clientID).Scan(
		&client.ID,
		&client.ClientID,
		&client.Hostname,
		&client.FirstSeen,
		&client.LastSeen,
		&client.Status,
		&osVersion,
		&buildNumber,
		&architecture,
		&domain,
		&ipAddress,
		&macAddress,
		&lastSubmission,
		&complianceScore,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	// Populate system info from nullable fields
	if osVersion.Valid {
		client.SystemInfo.OSVersion = osVersion.String
	}
	if buildNumber.Valid {
		client.SystemInfo.BuildNumber = buildNumber.String
	}
	if architecture.Valid {
		client.SystemInfo.Architecture = architecture.String
	}
	if domain.Valid {
		client.SystemInfo.Domain = domain.String
	}
	if ipAddress.Valid {
		client.SystemInfo.IPAddress = ipAddress.String
	}
	if macAddress.Valid {
		client.SystemInfo.MacAddress = macAddress.String
	}
	if lastSubmission.Valid {
		client.LastSubmission = lastSubmission.String
	}
	if complianceScore.Valid {
		client.ComplianceScore = complianceScore.Float64
	}

	return &client, nil
}

// GetClientSubmissions retrieves all submissions for a specific client
func (d *Database) GetClientSubmissions(clientID string) ([]api.SubmissionSummary, error) {
	query := `
		SELECT submission_id, client_id, hostname, timestamp, report_type,
		       overall_status, total_checks, passed_checks, failed_checks
		FROM submissions
		WHERE client_id = ?
		ORDER BY timestamp DESC
	`

	rows, err := d.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query client submissions: %w", err)
	}
	defer rows.Close()

	var submissions []api.SubmissionSummary
	for rows.Next() {
		var sub api.SubmissionSummary
		err := rows.Scan(
			&sub.SubmissionID,
			&sub.ClientID,
			&sub.Hostname,
			&sub.Timestamp,
			&sub.ReportType,
			&sub.OverallStatus,
			&sub.TotalChecks,
			&sub.PassedChecks,
			&sub.FailedChecks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, sub)
	}

	return submissions, nil
}

// ClearClientHistory deletes all submissions for a specific client
func (d *Database) ClearClientHistory(clientID string) (int64, error) {
	query := `DELETE FROM submissions WHERE client_id = ?`

	result, err := d.db.Exec(query, clientID)
	if err != nil {
		return 0, fmt.Errorf("failed to clear client history: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	d.logger.Info("Cleared client history", "client_id", clientID, "submissions_deleted", rowsAffected)
	return rowsAffected, nil
}
