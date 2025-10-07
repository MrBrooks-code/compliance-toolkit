package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"compliancetoolkit/pkg/api"
)

// ComplianceServer is the main server instance
type ComplianceServer struct {
	config     *ServerConfig
	logger     *slog.Logger
	httpServer *http.Server
	db         *Database
	mux        *http.ServeMux
}

// NewComplianceServer creates a new server instance
func NewComplianceServer(config *ServerConfig, logger *slog.Logger) (*ComplianceServer, error) {
	// Initialize database
	db, err := NewDatabase(config.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	server := &ComplianceServer{
		config: config,
		logger: logger,
		db:     db,
		mux:    http.NewServeMux(),
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// registerRoutes sets up HTTP handlers
func (s *ComplianceServer) registerRoutes() {
	// API endpoints
	s.mux.HandleFunc("/api/v1/health", s.handleHealth)
	s.mux.HandleFunc("/api/v1/compliance/submit", s.authMiddleware(s.handleSubmit))
	s.mux.HandleFunc("/api/v1/clients/register", s.authMiddleware(s.handleRegister))
	s.mux.HandleFunc("/api/v1/compliance/status/", s.authMiddleware(s.handleStatus))

	// Client detail endpoints (must be before /api/v1/clients to avoid conflict)
	s.mux.HandleFunc("/api/v1/clients/", s.authMiddleware(s.handleClientDetail))
	s.mux.HandleFunc("/api/v1/clients", s.authMiddleware(s.handleListClients))

	// Dashboard (if enabled)
	if s.config.Dashboard.Enabled {
		s.mux.HandleFunc(s.config.Dashboard.Path, s.handleDashboard)
		s.mux.HandleFunc("/settings", s.handleSettings)
		s.mux.HandleFunc("/policies", s.handlePoliciesPage)
		s.mux.HandleFunc("/client-detail", s.handleClientDetailPage)
		s.mux.HandleFunc("/submission-detail", s.handleSubmissionDetailPage)
		s.mux.HandleFunc("/api/v1/dashboard/summary", s.handleDashboardSummary)
		s.mux.HandleFunc("/api/v1/auth/session", s.handleGetSession)
	}

	// Submission endpoints
	s.mux.HandleFunc("/api/v1/submissions/", s.authMiddleware(s.handleSubmissionDetail))

	// Client management endpoints
	s.mux.HandleFunc("/api/v1/clients/clear-history/", s.authMiddleware(s.handleClearClientHistory))

	// Settings API endpoints
	s.mux.HandleFunc("/api/v1/settings/config", s.authMiddleware(s.handleGetConfig))
	s.mux.HandleFunc("/api/v1/settings/config/update", s.authMiddleware(s.handleUpdateConfig))
	s.mux.HandleFunc("/api/v1/settings/apikeys", s.authMiddleware(s.handleAPIKeys))
	s.mux.HandleFunc("/api/v1/settings/apikeys/add", s.authMiddleware(s.handleAddAPIKey))
	s.mux.HandleFunc("/api/v1/settings/apikeys/delete", s.authMiddleware(s.handleDeleteAPIKey))

	// Policy API endpoints
	s.mux.HandleFunc("/api/v1/policies/import", s.authMiddleware(s.handleImportPolicies))
	s.mux.HandleFunc("/api/v1/policies/", s.authMiddleware(s.handlePolicyDetail))
	s.mux.HandleFunc("/api/v1/policies", s.authMiddleware(s.handlePolicies))

	// Root handler
	s.mux.HandleFunc("/", s.handleRoot)
}

// Start starts the HTTP server
func (s *ComplianceServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.loggingMiddleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		var err error
		if s.config.Server.TLS.Enabled {
			s.logger.Info("Starting HTTPS server",
				"addr", addr,
				"cert", s.config.Server.TLS.CertFile,
			)
			err = s.httpServer.ListenAndServeTLS(
				s.config.Server.TLS.CertFile,
				s.config.Server.TLS.KeyFile,
			)
		} else {
			s.logger.Info("Starting HTTP server", "addr", addr)
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error", "error", err)
		}
	}()

	s.logger.Info("Server started successfully")
	return nil
}

// Shutdown gracefully shuts down the server
func (s *ComplianceServer) Shutdown() error {
	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Close database
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("database close failed: %w", err)
	}

	s.logger.Info("Server shutdown complete")
	return nil
}

// handleRoot handles root path requests
func (s *ComplianceServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "Compliance Toolkit Server",
		"version": version,
		"status":  "running",
	})
}

// handleHealth handles health check requests
func (s *ComplianceServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check database connection
	if err := s.db.Ping(); err != nil {
		s.logger.Error("Health check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"version": version,
	})
}

// handleSubmit handles compliance submission requests
func (s *ComplianceServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse submission
	var submission api.ComplianceSubmission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		s.logger.Warn("Invalid submission JSON", "error", err)
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate submission
	if err := submission.Validate(); err != nil {
		s.logger.Warn("Submission validation failed", "error", err)
		s.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Info("Received compliance submission",
		"submission_id", submission.SubmissionID,
		"client_id", submission.ClientID,
		"hostname", submission.Hostname,
		"report_type", submission.ReportType,
	)

	// Store submission in database
	if err := s.db.SaveSubmission(&submission); err != nil {
		s.logger.Error("Failed to save submission", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to save submission")
		return
	}

	// Update client last_seen and system info
	if err := s.db.UpdateClientLastSeen(submission.ClientID, submission.Hostname, &submission.SystemInfo); err != nil {
		s.logger.Warn("Failed to update client last_seen", "error", err)
		// Non-fatal - continue
	}

	// Send response
	response := api.SubmissionResponse{
		SubmissionID: submission.SubmissionID,
		Status:       "accepted",
		Message:      "Submission received and stored successfully",
		ReceivedAt:   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleRegister handles client registration requests
func (s *ComplianceServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse registration
	var registration api.ClientRegistration
	if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	s.logger.Info("Client registration",
		"client_id", registration.ClientID,
		"hostname", registration.Hostname,
	)

	// Register client in database
	if err := s.db.RegisterClient(&registration); err != nil {
		s.logger.Error("Failed to register client", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to register client")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "registered",
		"message": "Client registered successfully",
	})
}

// handleStatus handles submission status requests
func (s *ComplianceServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract submission ID from path
	submissionID := strings.TrimPrefix(r.URL.Path, "/api/v1/compliance/status/")
	if submissionID == "" {
		s.sendError(w, http.StatusBadRequest, "Submission ID required")
		return
	}

	// Get submission from database
	submission, err := s.db.GetSubmission(submissionID)
	if err != nil {
		s.logger.Error("Failed to get submission", "error", err)
		s.sendError(w, http.StatusNotFound, "Submission not found")
		return
	}

	// Create summary
	summary := api.SubmissionSummary{
		SubmissionID:  submission.SubmissionID,
		ClientID:      submission.ClientID,
		Hostname:      submission.Hostname,
		Timestamp:     submission.Timestamp,
		ReportType:    submission.ReportType,
		OverallStatus: submission.Compliance.OverallStatus,
		PassedChecks:  submission.Compliance.PassedChecks,
		FailedChecks:  submission.Compliance.FailedChecks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// handleListClients handles client list requests
func (s *ComplianceServer) handleListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clients, err := s.db.ListClients()
	if err != nil {
		s.logger.Error("Failed to list clients", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to list clients")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

// handleDashboard serves the web dashboard
func (s *ComplianceServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Read dashboard HTML file
	html, err := os.ReadFile("dashboard.html")
	if err != nil {
		s.logger.Error("Failed to read dashboard.html", "error", err)
		http.Error(w, "Dashboard not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

// handleSettings serves the settings page
func (s *ComplianceServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	// Read settings HTML file
	html, err := os.ReadFile("settings.html")
	if err != nil {
		s.logger.Error("Failed to read settings.html", "error", err)
		http.Error(w, "Settings not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

func (s *ComplianceServer) handlePoliciesPage(w http.ResponseWriter, r *http.Request) {
	// Read policies HTML file
	html, err := os.ReadFile("policies.html")
	if err != nil {
		s.logger.Error("Failed to read policies.html", "error", err)
		http.Error(w, "Policies page not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

// handleDashboardSummary provides dashboard data
func (s *ComplianceServer) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary, err := s.db.GetDashboardSummary()
	if err != nil {
		s.logger.Error("Failed to get dashboard summary", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to get dashboard summary")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// authMiddleware checks API key authentication
func (s *ComplianceServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled
		if !s.config.Auth.Enabled || !s.config.Auth.RequireKey {
			next(w, r)
			return
		}

		var apiKey string

		// Try to get API key from cookie first (for dashboard/web UI)
		if cookie, err := r.Cookie("api_token"); err == nil {
			apiKey = cookie.Value
		} else {
			// Fall back to Authorization header (for API clients)
			apiKey = r.Header.Get("Authorization")
			if apiKey == "" {
				s.sendError(w, http.StatusUnauthorized, "API key required")
				return
			}
			// Remove "Bearer " prefix if present
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}

		// Validate API key
		valid := false
		for _, key := range s.config.Auth.APIKeys {
			if apiKey == key {
				valid = true
				break
			}
		}

		if !valid {
			s.logger.Warn("Invalid API key", "remote_addr", r.RemoteAddr)
			s.sendError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		next(w, r)
	}
}

// loggingMiddleware logs all HTTP requests
func (s *ComplianceServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"status", wrapped.statusCode,
			"duration", duration.Milliseconds(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// sendError sends a JSON error response
func (s *ComplianceServer) sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(api.ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}

// handleGetConfig returns current server configuration (sanitized)
func (s *ComplianceServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create sanitized config (don't expose sensitive data like API keys)
	configResponse := map[string]interface{}{
		"server": map[string]interface{}{
			"host": s.config.Server.Host,
			"port": s.config.Server.Port,
			"tls": map[string]interface{}{
				"enabled":   s.config.Server.TLS.Enabled,
				"cert_file": s.config.Server.TLS.CertFile,
				"key_file":  s.config.Server.TLS.KeyFile,
			},
		},
		"database": map[string]interface{}{
			"type": s.config.Database.Type,
			"path": s.config.Database.Path,
		},
		"auth": map[string]interface{}{
			"enabled":     s.config.Auth.Enabled,
			"require_key": s.config.Auth.RequireKey,
			"key_count":   len(s.config.Auth.APIKeys),
		},
		"dashboard": map[string]interface{}{
			"enabled": s.config.Dashboard.Enabled,
			"path":    s.config.Dashboard.Path,
		},
		"logging": map[string]interface{}{
			"level":  s.config.Logging.Level,
			"format": s.config.Logging.Format,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResponse)
}

// handleUpdateConfig updates server configuration
func (s *ComplianceServer) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Apply updates (limited to non-sensitive settings for safety)
	// Note: This updates runtime config only, not the YAML file
	// TODO: Add YAML file persistence if needed

	if logging, ok := updates["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			s.config.Logging.Level = level
			s.logger.Info("Logging level updated", "new_level", level)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Configuration updated (runtime only)",
	})
}

// handleAPIKeys returns list of API keys (masked)
func (s *ComplianceServer) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Mask API keys for security
	maskedKeys := make([]map[string]string, 0, len(s.config.Auth.APIKeys))
	for i, key := range s.config.Auth.APIKeys {
		masked := maskAPIKey(key)
		maskedKeys = append(maskedKeys, map[string]string{
			"id":     fmt.Sprintf("key-%d", i+1),
			"key":    masked,
			"masked": "true",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(maskedKeys)
}

// handleAddAPIKey adds a new API key
func (s *ComplianceServer) handleAddAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Key string `json:"key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if request.Key == "" {
		s.sendError(w, http.StatusBadRequest, "API key cannot be empty")
		return
	}

	// Check for duplicates
	for _, existingKey := range s.config.Auth.APIKeys {
		if existingKey == request.Key {
			s.sendError(w, http.StatusConflict, "API key already exists")
			return
		}
	}

	// Add to runtime config
	s.config.Auth.APIKeys = append(s.config.Auth.APIKeys, request.Key)

	s.logger.Info("API key added", "key_preview", maskAPIKey(request.Key))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "API key added successfully",
	})
}

// handleDeleteAPIKey deletes an API key
func (s *ComplianceServer) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Key string `json:"key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Find and remove key
	found := false
	newKeys := make([]string, 0, len(s.config.Auth.APIKeys))
	for _, key := range s.config.Auth.APIKeys {
		if key != request.Key {
			newKeys = append(newKeys, key)
		} else {
			found = true
		}
	}

	if !found {
		s.sendError(w, http.StatusNotFound, "API key not found")
		return
	}

	s.config.Auth.APIKeys = newKeys

	s.logger.Info("API key deleted", "key_preview", maskAPIKey(request.Key))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "API key deleted successfully",
	})
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// handleClientDetail handles client detail requests (API endpoint)
func (s *ComplianceServer) handleClientDetail(w http.ResponseWriter, r *http.Request) {
	// If path is exactly /api/v1/clients (no trailing slash), this is list endpoint
	if r.URL.Path == "/api/v1/clients" {
		s.handleListClients(w, r)
		return
	}

	// Extract client_id from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/clients/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		s.sendError(w, http.StatusBadRequest, "Client ID required")
		return
	}

	clientID := parts[0]

	// Handle /api/v1/clients/{client_id}/submissions endpoint
	if len(parts) > 1 && parts[1] == "submissions" {
		s.handleClientSubmissions(w, r, clientID)
		return
	}

	// Handle GET /api/v1/clients/{client_id} endpoint
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client from database
	client, err := s.db.GetClient(clientID)
	if err != nil {
		s.logger.Error("Failed to get client", "error", err, "client_id", clientID)
		s.sendError(w, http.StatusNotFound, "Client not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client)
}

// handleClientSubmissions handles client submission history requests
func (s *ComplianceServer) handleClientSubmissions(w http.ResponseWriter, r *http.Request, clientID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get submissions from database
	submissions, err := s.db.GetClientSubmissions(clientID)
	if err != nil {
		s.logger.Error("Failed to get client submissions", "error", err, "client_id", clientID)
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve submissions")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submissions)
}

// handleClientDetailPage serves the client detail HTML page
func (s *ComplianceServer) handleClientDetailPage(w http.ResponseWriter, r *http.Request) {
	// Read client detail HTML file
	html, err := os.ReadFile("client-detail.html")
	if err != nil {
		s.logger.Error("Failed to read client-detail.html", "error", err)
		http.Error(w, "Client detail page not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

// handleGetSession returns session info (used by dashboard to check if authenticated)
func (s *ComplianceServer) handleGetSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for cookie
	cookie, err := r.Cookie("api_token")
	if err != nil || cookie.Value == "" {
		// No session cookie, return first API key for convenience (dashboard use only)
		if len(s.config.Auth.APIKeys) > 0 {
			// Set cookie with first API key
			http.SetCookie(w, &http.Cookie{
				Name:     "api_token",
				Value:    s.config.Auth.APIKeys[0],
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   86400, // 24 hours
			})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":         "authenticated",
				"authentication": "cookie",
			})
			return
		}

		s.sendError(w, http.StatusUnauthorized, "No authentication found")
		return
	}

	// Validate existing cookie
	valid := false
	for _, key := range s.config.Auth.APIKeys {
		if cookie.Value == key {
			valid = true
			break
		}
	}

	if !valid {
		s.sendError(w, http.StatusUnauthorized, "Invalid session")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":         "authenticated",
		"authentication": "cookie",
	})
}

// handleSubmissionDetail handles submission detail requests (API endpoint)
func (s *ComplianceServer) handleSubmissionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract submission_id from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/submissions/")
	submissionID := strings.TrimSuffix(path, "/")

	if submissionID == "" {
		s.sendError(w, http.StatusBadRequest, "Submission ID required")
		return
	}

	// Get submission from database
	submission, err := s.db.GetSubmission(submissionID)
	if err != nil {
		s.logger.Error("Failed to get submission", "error", err, "submission_id", submissionID)
		s.sendError(w, http.StatusNotFound, "Submission not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submission)
}

// handleSubmissionDetailPage serves the submission detail HTML page
func (s *ComplianceServer) handleSubmissionDetailPage(w http.ResponseWriter, r *http.Request) {
	// Read submission detail HTML file
	html, err := os.ReadFile("submission-detail.html")
	if err != nil {
		s.logger.Error("Failed to read submission-detail.html", "error", err)
		http.Error(w, "Submission detail page not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

// handleClearClientHistory clears all submission history for a client
func (s *ComplianceServer) handleClearClientHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract client_id from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/clients/clear-history/")
	clientID := strings.TrimSuffix(path, "/")

	if clientID == "" {
		s.sendError(w, http.StatusBadRequest, "Client ID required")
		return
	}

	// Verify client exists
	_, err := s.db.GetClient(clientID)
	if err != nil {
		s.logger.Error("Client not found", "error", err, "client_id", clientID)
		s.sendError(w, http.StatusNotFound, "Client not found")
		return
	}

	// Clear client history
	deletedCount, err := s.db.ClearClientHistory(clientID)
	if err != nil {
		s.logger.Error("Failed to clear client history", "error", err, "client_id", clientID)
		s.sendError(w, http.StatusInternalServerError, "Failed to clear history")
		return
	}

	s.logger.Info("Client history cleared", "client_id", clientID, "deleted_count", deletedCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"message":       fmt.Sprintf("Cleared %d submissions for client %s", deletedCount, clientID),
		"deleted_count": deletedCount,
	})
}

// handlePolicies handles GET (list), POST (create) for /api/v1/policies
func (s *ComplianceServer) handlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListPolicies(w, r)
	case http.MethodPost:
		s.handleCreatePolicy(w, r)
	default:
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handlePolicyDetail handles GET (detail), PUT (update), DELETE for /api/v1/policies/{policy_id}
func (s *ComplianceServer) handlePolicyDetail(w http.ResponseWriter, r *http.Request) {
	// Handle /api/v1/policies endpoint (no trailing slash)
	if r.URL.Path == "/api/v1/policies" {
		s.handlePolicies(w, r)
		return
	}

	// Extract policy_id from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
	policyID := strings.TrimSuffix(path, "/")

	if policyID == "" {
		s.sendError(w, http.StatusBadRequest, "Policy ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicy(w, r, policyID)
	case http.MethodPut:
		s.handleUpdatePolicy(w, r, policyID)
	case http.MethodDelete:
		s.handleDeletePolicy(w, r, policyID)
	default:
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListPolicies returns all policies
func (s *ComplianceServer) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := s.db.ListPolicies()
	if err != nil {
		s.logger.Error("Failed to list policies", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve policies")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

// handleGetPolicy returns a specific policy
func (s *ComplianceServer) handleGetPolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	policy, err := s.db.GetPolicy(policyID)
	if err != nil {
		s.logger.Error("Failed to get policy", "error", err, "policy_id", policyID)
		if err.Error() == "policy not found" {
			s.sendError(w, http.StatusNotFound, "Policy not found")
		} else {
			s.sendError(w, http.StatusInternalServerError, "Failed to retrieve policy")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policy)
}

// handleCreatePolicy creates a new policy
func (s *ComplianceServer) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var policy Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if policy.PolicyID == "" || policy.Name == "" || policy.PolicyData == "" {
		s.sendError(w, http.StatusBadRequest, "Missing required fields: policy_id, name, policy_data")
		return
	}

	// Set default status if not provided
	if policy.Status == "" {
		policy.Status = "active"
	}

	if err := s.db.CreatePolicy(&policy); err != nil {
		s.logger.Error("Failed to create policy", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to create policy")
		return
	}

	s.logger.Info("Policy created", "policy_id", policy.PolicyID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Policy created successfully",
		"policy_id": policy.PolicyID,
	})
}

// handleUpdatePolicy updates an existing policy
func (s *ComplianceServer) handleUpdatePolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	var policy Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.db.UpdatePolicy(policyID, &policy); err != nil {
		s.logger.Error("Failed to update policy", "error", err, "policy_id", policyID)
		if err.Error() == "policy not found" {
			s.sendError(w, http.StatusNotFound, "Policy not found")
		} else {
			s.sendError(w, http.StatusInternalServerError, "Failed to update policy")
		}
		return
	}

	s.logger.Info("Policy updated", "policy_id", policyID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Policy updated successfully",
	})
}

// handleDeletePolicy deletes a policy
func (s *ComplianceServer) handleDeletePolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	if err := s.db.DeletePolicy(policyID); err != nil {
		s.logger.Error("Failed to delete policy", "error", err, "policy_id", policyID)
		if err.Error() == "policy not found" {
			s.sendError(w, http.StatusNotFound, "Policy not found")
		} else {
			s.sendError(w, http.StatusInternalServerError, "Failed to delete policy")
		}
		return
	}

	s.logger.Info("Policy deleted", "policy_id", policyID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Policy deleted successfully",
	})
}

// handleImportPolicies imports policies from configs/reports directory
func (s *ComplianceServer) handleImportPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Look for report files in configs/reports directory
	reportsDir := "../../configs/reports"
	files, err := filepath.Glob(filepath.Join(reportsDir, "*.json"))
	if err != nil {
		s.logger.Error("Failed to list report files", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to list report files")
		return
	}

	imported := 0
	skipped := 0
	errors := []string{}

	for _, file := range files {
		// Read the report file
		data, err := os.ReadFile(file)
		if err != nil {
			s.logger.Warn("Failed to read report file", "file", file, "error", err)
			errors = append(errors, fmt.Sprintf("Failed to read %s: %v", filepath.Base(file), err))
			continue
		}

		// Parse the report to extract metadata
		var reportConfig struct {
			Version  string `json:"version"`
			Metadata struct {
				ReportTitle   string `json:"report_title"`
				ReportVersion string `json:"report_version"`
				Author        string `json:"author"`
				Description   string `json:"description"`
				Category      string `json:"category"`
				Compliance    string `json:"compliance"`
			} `json:"metadata"`
		}

		if err := json.Unmarshal(data, &reportConfig); err != nil {
			s.logger.Warn("Failed to parse report file", "file", file, "error", err)
			errors = append(errors, fmt.Sprintf("Failed to parse %s: %v", filepath.Base(file), err))
			continue
		}

		// Extract framework from compliance field (e.g., "NIST 800-171 Rev 2" -> "NIST")
		framework := ""
		if reportConfig.Metadata.Compliance != "" {
			parts := strings.Split(reportConfig.Metadata.Compliance, " ")
			if len(parts) > 0 {
				framework = parts[0]
			}
		}

		// Generate policy_id from filename
		filename := filepath.Base(file)
		policyID := strings.TrimSuffix(filename, filepath.Ext(filename))

		// Check if policy already exists
		existing, _ := s.db.GetPolicy(policyID)
		if existing != nil {
			s.logger.Info("Policy already exists, skipping", "policy_id", policyID)
			skipped++
			continue
		}

		// Create policy
		policy := Policy{
			PolicyID:    policyID,
			Name:        reportConfig.Metadata.ReportTitle,
			Description: reportConfig.Metadata.Description,
			Framework:   framework,
			Version:     reportConfig.Metadata.ReportVersion,
			Category:    reportConfig.Metadata.Category,
			Author:      reportConfig.Metadata.Author,
			Status:      "active",
			PolicyData:  string(data),
		}

		if err := s.db.CreatePolicy(&policy); err != nil {
			s.logger.Error("Failed to import policy", "policy_id", policyID, "error", err)
			errors = append(errors, fmt.Sprintf("Failed to import %s: %v", policyID, err))
			continue
		}

		s.logger.Info("Policy imported", "policy_id", policyID, "name", policy.Name)
		imported++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
		"message":  fmt.Sprintf("Imported %d policies, skipped %d existing", imported, skipped),
	})
}
