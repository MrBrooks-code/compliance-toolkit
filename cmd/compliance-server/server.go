package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	s.mux.HandleFunc("/api/v1/clients", s.authMiddleware(s.handleListClients))

	// Dashboard (if enabled)
	if s.config.Dashboard.Enabled {
		s.mux.HandleFunc(s.config.Dashboard.Path, s.handleDashboard)
		s.mux.HandleFunc("/settings", s.handleSettings)
		s.mux.HandleFunc("/api/v1/dashboard/summary", s.handleDashboardSummary)
	}

	// Settings API endpoints
	s.mux.HandleFunc("/api/v1/settings/config", s.authMiddleware(s.handleGetConfig))
	s.mux.HandleFunc("/api/v1/settings/config/update", s.authMiddleware(s.handleUpdateConfig))
	s.mux.HandleFunc("/api/v1/settings/apikeys", s.authMiddleware(s.handleAPIKeys))
	s.mux.HandleFunc("/api/v1/settings/apikeys/add", s.authMiddleware(s.handleAddAPIKey))
	s.mux.HandleFunc("/api/v1/settings/apikeys/delete", s.authMiddleware(s.handleDeleteAPIKey))

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

	// Update client last_seen
	if err := s.db.UpdateClientLastSeen(submission.ClientID, submission.Hostname); err != nil {
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

		// Get API key from header
		apiKey := r.Header.Get("Authorization")
		if apiKey == "" {
			s.sendError(w, http.StatusUnauthorized, "API key required")
			return
		}

		// Remove "Bearer " prefix if present
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")

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
