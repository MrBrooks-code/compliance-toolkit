package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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

	-- Policies table
	CREATE TABLE IF NOT EXISTS policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		policy_id TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		framework TEXT,  -- NIST, FIPS, CIS, etc.
		version TEXT,
		category TEXT,
		author TEXT,
		status TEXT DEFAULT 'active',  -- active, inactive, draft
		policy_data TEXT NOT NULL,  -- JSON policy configuration
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Client policy assignments (for future use)
	CREATE TABLE IF NOT EXISTS client_policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id TEXT NOT NULL,
		policy_id TEXT NOT NULL,
		assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		assigned_by TEXT,
		FOREIGN KEY (client_id) REFERENCES clients(client_id),
		FOREIGN KEY (policy_id) REFERENCES policies(policy_id),
		UNIQUE(client_id, policy_id)
	);

	-- Users table for authentication
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('admin', 'viewer', 'auditor')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login TIMESTAMP
	);

	-- API Keys table for secure key management
	CREATE TABLE IF NOT EXISTS api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		key_prefix TEXT NOT NULL,  -- First 8 chars for display
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used TIMESTAMP,
		expires_at TIMESTAMP,
		is_active BOOLEAN DEFAULT 1
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_submissions_client_id ON submissions(client_id);
	CREATE INDEX IF NOT EXISTS idx_submissions_timestamp ON submissions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_submissions_report_type ON submissions(report_type);
	CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);
	CREATE INDEX IF NOT EXISTS idx_policies_framework ON policies(framework);
	CREATE INDEX IF NOT EXISTS idx_policies_status ON policies(status);
	CREATE INDEX IF NOT EXISTS idx_client_policies_client_id ON client_policies(client_id);
	CREATE INDEX IF NOT EXISTS idx_client_policies_policy_id ON client_policies(policy_id);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
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
		submission.Timestamp.Format(time.RFC3339),
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
	var timestampStr string

	err := d.db.QueryRow(query, submissionID).Scan(
		&submission.SubmissionID,
		&submission.ClientID,
		&submission.Hostname,
		&timestampStr,
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

	// Parse timestamp from string
	submission.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
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
		var timestampStr string
		err := rows.Scan(
			&sub.SubmissionID,
			&sub.ClientID,
			&sub.Hostname,
			&timestampStr,
			&sub.ReportType,
			&sub.OverallStatus,
			&sub.PassedChecks,
			&sub.FailedChecks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		// Parse timestamp from string
		sub.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
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
		var timestampStr string
		err := rows.Scan(
			&sub.SubmissionID,
			&sub.ClientID,
			&sub.Hostname,
			&timestampStr,
			&sub.ReportType,
			&sub.OverallStatus,
			&sub.TotalChecks,
			&sub.PassedChecks,
			&sub.FailedChecks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		// Parse timestamp from string
		sub.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
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

// Policy represents a compliance policy
type Policy struct {
	ID          int    `json:"id"`
	PolicyID    string `json:"policy_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Framework   string `json:"framework"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Author      string `json:"author"`
	Status      string `json:"status"`
	PolicyData  string `json:"policy_data"` // JSON
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ListPolicies retrieves all policies
func (d *Database) ListPolicies() ([]Policy, error) {
	query := `
		SELECT id, policy_id, name, description, framework, version, category, author, status,
		       policy_data, created_at, updated_at
		FROM policies
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		var description, framework, version, category, author sql.NullString

		err := rows.Scan(
			&p.ID,
			&p.PolicyID,
			&p.Name,
			&description,
			&framework,
			&version,
			&category,
			&author,
			&p.Status,
			&p.PolicyData,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		// Handle NULL fields
		if description.Valid {
			p.Description = description.String
		}
		if framework.Valid {
			p.Framework = framework.String
		}
		if version.Valid {
			p.Version = version.String
		}
		if category.Valid {
			p.Category = category.String
		}
		if author.Valid {
			p.Author = author.String
		}

		policies = append(policies, p)
	}

	return policies, nil
}

// GetPolicy retrieves a specific policy by policy_id
func (d *Database) GetPolicy(policyID string) (*Policy, error) {
	query := `
		SELECT id, policy_id, name, description, framework, version, category, author, status,
		       policy_data, created_at, updated_at
		FROM policies
		WHERE policy_id = ?
	`

	var p Policy
	var description, framework, version, category, author sql.NullString

	err := d.db.QueryRow(query, policyID).Scan(
		&p.ID,
		&p.PolicyID,
		&p.Name,
		&description,
		&framework,
		&version,
		&category,
		&author,
		&p.Status,
		&p.PolicyData,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query policy: %w", err)
	}

	// Handle NULL fields
	if description.Valid {
		p.Description = description.String
	}
	if framework.Valid {
		p.Framework = framework.String
	}
	if version.Valid {
		p.Version = version.String
	}
	if category.Valid {
		p.Category = category.String
	}
	if author.Valid {
		p.Author = author.String
	}

	return &p, nil
}

// CreatePolicy creates a new policy
func (d *Database) CreatePolicy(p *Policy) error {
	query := `
		INSERT INTO policies (
			policy_id, name, description, framework, version, category, author, status, policy_data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(
		query,
		p.PolicyID,
		p.Name,
		p.Description,
		p.Framework,
		p.Version,
		p.Category,
		p.Author,
		p.Status,
		p.PolicyData,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	d.logger.Info("Policy created", "policy_id", p.PolicyID, "name", p.Name)
	return nil
}

// UpdatePolicy updates an existing policy
func (d *Database) UpdatePolicy(policyID string, p *Policy) error {
	query := `
		UPDATE policies
		SET name = ?, description = ?, framework = ?, version = ?, category = ?,
		    author = ?, status = ?, policy_data = ?, updated_at = CURRENT_TIMESTAMP
		WHERE policy_id = ?
	`

	result, err := d.db.Exec(
		query,
		p.Name,
		p.Description,
		p.Framework,
		p.Version,
		p.Category,
		p.Author,
		p.Status,
		p.PolicyData,
		policyID,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found")
	}

	d.logger.Info("Policy updated", "policy_id", policyID)
	return nil
}

// DeletePolicy deletes a policy
func (d *Database) DeletePolicy(policyID string) error {
	query := `DELETE FROM policies WHERE policy_id = ?`

	result, err := d.db.Exec(query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found")
	}

	d.logger.Info("Policy deleted", "policy_id", policyID)
	return nil
}

// User represents a user account
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // Never expose in JSON
	Role         string `json:"role"`
	CreatedAt    string `json:"created_at"`
	LastLogin    string `json:"last_login,omitempty"`
}

// CreateUser creates a new user with hashed password
func (d *Database) CreateUser(username, passwordHash, role string) error {
	query := `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`

	_, err := d.db.Exec(query, username, passwordHash, role)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	d.logger.Info("User created", "username", username, "role", role)
	return nil
}

// GetUser retrieves a user by username
func (d *Database) GetUser(username string) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login FROM users WHERE username = ?`

	var user User
	var lastLogin sql.NullString

	err := d.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&lastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	if lastLogin.Valid {
		user.LastLogin = lastLogin.String
	}

	return &user, nil
}

// ListUsers retrieves all users
func (d *Database) ListUsers() ([]User, error) {
	query := `SELECT id, username, role, created_at, last_login FROM users ORDER BY created_at DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var lastLogin sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Role,
			&user.CreatedAt,
			&lastLogin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if lastLogin.Valid {
			user.LastLogin = lastLogin.String
		}

		users = append(users, user)
	}

	return users, nil
}

// UpdateUserLastLogin updates the last_login timestamp
func (d *Database) UpdateUserLastLogin(username string) error {
	query := `UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE username = ?`

	_, err := d.db.Exec(query, username)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// UpdateUserPassword updates a user's password hash
func (d *Database) UpdateUserPassword(username, passwordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE username = ?`

	result, err := d.db.Exec(query, passwordHash, username)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	d.logger.Info("User password updated", "username", username)
	return nil
}

// DeleteUser deletes a user
func (d *Database) DeleteUser(username string) error {
	query := `DELETE FROM users WHERE username = ?`

	result, err := d.db.Exec(query, username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	d.logger.Info("User deleted", "username", username)
	return nil
}

// UserExists checks if a user exists
func (d *Database) UserExists(username string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE username = ?`

	var count int
	err := d.db.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return count > 0, nil
}

// HasAnyUsers checks if any users exist in the database
func (d *Database) HasAnyUsers() (bool, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	err := d.db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check users: %w", err)
	}

	return count > 0, nil
}

// APIKey represents an API key in the database
type APIKey struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	KeyHash   string `json:"-"`           // Never expose hash in JSON
	KeyPrefix string `json:"key_prefix"`  // First 8 chars for display
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	LastUsed  string `json:"last_used,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	IsActive  bool   `json:"is_active"`
}

// CreateAPIKey creates a new API key in the database
func (d *Database) CreateAPIKey(name, keyHash, keyPrefix, createdBy string, expiresAt *string) error {
	query := `
		INSERT INTO api_keys (name, key_hash, key_prefix, created_by, expires_at, is_active)
		VALUES (?, ?, ?, ?, ?, 1)
	`

	_, err := d.db.Exec(query, name, keyHash, keyPrefix, createdBy, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	d.logger.Info("API key created", "name", name, "created_by", createdBy)
	return nil
}

// ListAPIKeys retrieves all API keys
func (d *Database) ListAPIKeys() ([]APIKey, error) {
	query := `
		SELECT id, name, key_hash, key_prefix, created_by, created_at, last_used, expires_at, is_active
		FROM api_keys
		ORDER BY created_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil to avoid JSON null
	keys := []APIKey{}
	for rows.Next() {
		var key APIKey
		var lastUsed, expiresAt sql.NullString

		err := rows.Scan(
			&key.ID,
			&key.Name,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.CreatedBy,
			&key.CreatedAt,
			&lastUsed,
			&expiresAt,
			&key.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if lastUsed.Valid {
			key.LastUsed = lastUsed.String
		}
		if expiresAt.Valid {
			key.ExpiresAt = expiresAt.String
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (d *Database) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	query := `
		SELECT id, name, key_hash, key_prefix, created_by, created_at, last_used, expires_at, is_active
		FROM api_keys
		WHERE key_hash = ? AND is_active = 1
	`

	var key APIKey
	var lastUsed, expiresAt sql.NullString

	err := d.db.QueryRow(query, keyHash).Scan(
		&key.ID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.CreatedBy,
		&key.CreatedAt,
		&lastUsed,
		&expiresAt,
		&key.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query API key: %w", err)
	}

	if lastUsed.Valid {
		key.LastUsed = lastUsed.String
	}
	if expiresAt.Valid {
		key.ExpiresAt = expiresAt.String
	}

	return &key, nil
}

// ListActiveAPIKeyHashes retrieves all active API key hashes for authentication
func (d *Database) ListActiveAPIKeyHashes() ([]string, error) {
	query := `
		SELECT key_hash
		FROM api_keys
		WHERE is_active = 1 AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active API key hashes: %w", err)
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil
	hashes := []string{}
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("failed to scan key hash: %w", err)
		}
		hashes = append(hashes, hash)
	}

	return hashes, nil
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key
func (d *Database) UpdateAPIKeyLastUsed(keyHash string) error {
	query := `UPDATE api_keys SET last_used = CURRENT_TIMESTAMP WHERE key_hash = ?`

	_, err := d.db.Exec(query, keyHash)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key by ID
func (d *Database) DeleteAPIKey(id int) error {
	query := `DELETE FROM api_keys WHERE id = ?`

	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	d.logger.Info("API key deleted", "id", id)
	return nil
}

// DeactivateAPIKey deactivates an API key by ID
func (d *Database) DeactivateAPIKey(id int) error {
	query := `UPDATE api_keys SET is_active = 0 WHERE id = ?`

	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	d.logger.Info("API key deactivated", "id", id)
	return nil
}

// ActivateAPIKey activates an API key by ID
func (d *Database) ActivateAPIKey(id int) error {
	query := `UPDATE api_keys SET is_active = 1 WHERE id = ?`

	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to activate API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	d.logger.Info("API key activated", "id", id)
	return nil
}
