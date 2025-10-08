package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"compliancetoolkit/pkg/api"
	"golang.org/x/crypto/bcrypt"
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

	// Create initial admin user if no users exist
	if err := server.ensureAdminUser(); err != nil {
		logger.Warn("Failed to create initial admin user", "error", err)
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// ensureAdminUser creates an initial admin user if no users exist
func (s *ComplianceServer) ensureAdminUser() error {
	hasUsers, err := s.db.HasAnyUsers()
	if err != nil {
		return fmt.Errorf("failed to check for users: %w", err)
	}

	if !hasUsers {
		// Create default admin user
		defaultPassword := "admin"
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash default password: %w", err)
		}

		if err := s.db.CreateUser("admin", string(passwordHash), "admin"); err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		s.logger.Warn("Created default admin user",
			"username", "admin",
			"password", "admin",
			"warning", "PLEASE CHANGE THIS PASSWORD IMMEDIATELY",
		)
	}

	return nil
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

	// Authentication endpoints
	s.mux.HandleFunc("/login", s.handleLoginPage)
	s.mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/v1/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/v1/auth/session", s.handleGetSession)

	// Config endpoints (public for login message)
	s.mux.HandleFunc("/api/v1/config/login-message", s.handleGetLoginMessage)
	s.mux.HandleFunc("/api/v1/config/login-message/update", s.authMiddleware(s.handleUpdateLoginMessage))

	// Dashboard (if enabled)
	if s.config.Dashboard.Enabled {
		s.mux.HandleFunc(s.config.Dashboard.Path, s.requireAuth(s.handleDashboard))
		s.mux.HandleFunc("/clients", s.requireAuth(s.handleClientsPage))
		s.mux.HandleFunc("/settings", s.requireAuth(s.handleSettings))
		s.mux.HandleFunc("/policies", s.requireAuth(s.handlePoliciesPage))
		s.mux.HandleFunc("/about", s.requireAuth(s.handleAboutPage))
		s.mux.HandleFunc("/client-detail", s.requireAuth(s.handleClientDetailPage))
		s.mux.HandleFunc("/submission-detail", s.requireAuth(s.handleSubmissionDetailPage))
		s.mux.HandleFunc("/api/v1/dashboard/summary", s.requireAuth(s.handleDashboardSummary))
	}

	// Submission endpoints
	s.mux.HandleFunc("/api/v1/submissions/", s.authMiddleware(s.handleSubmissionDetail))

	// Client management endpoints
	s.mux.HandleFunc("/api/v1/clients/clear-history/", s.authMiddleware(s.handleClearClientHistory))

	// Settings API endpoints
	s.mux.HandleFunc("/api/v1/settings/config", s.authMiddleware(s.handleGetConfig))
	s.mux.HandleFunc("/api/v1/settings/config/update", s.authMiddleware(s.handleUpdateConfig))

	// User management API endpoints
	s.mux.HandleFunc("/api/v1/users", s.authMiddleware(s.handleUsers))
	s.mux.HandleFunc("/api/v1/users/create", s.authMiddleware(s.handleCreateUser))
	s.mux.HandleFunc("/api/v1/users/delete", s.authMiddleware(s.handleDeleteUser))
	s.mux.HandleFunc("/api/v1/users/change-password", s.authMiddleware(s.handleChangePassword))

	// API Key management endpoints (database-backed)
	// Register more specific routes first to avoid conflicts
	s.mux.HandleFunc("/api/v1/apikeys/generate", s.authMiddleware(s.handleGenerateAPIKey))
	s.mux.HandleFunc("/api/v1/apikeys/delete", s.authMiddleware(s.handleDeleteAPIKeyDB))
	s.mux.HandleFunc("/api/v1/apikeys/toggle", s.authMiddleware(s.handleToggleAPIKey))
	s.mux.HandleFunc("/api/v1/apikeys", s.authMiddleware(s.handleListAPIKeys))

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

// handleLoginPage serves the login page
func (s *ComplianceServer) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	html, err := os.ReadFile("login.html")
	if err != nil {
		s.logger.Error("Failed to read login.html", "error", err)
		http.Error(w, "Login page not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

// handleLogin processes login requests
func (s *ComplianceServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate inputs
	if loginReq.Username == "" || loginReq.Password == "" {
		s.sendError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	// Get user from database
	user, err := s.db.GetUser(loginReq.Username)
	if err != nil {
		s.logger.Warn("Login attempt for non-existent user", "username", loginReq.Username)
		s.sendError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginReq.Password))
	if err != nil {
		s.logger.Warn("Failed login attempt", "username", loginReq.Username, "remote_addr", r.RemoteAddr)
		s.sendError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Update last login timestamp
	if err := s.db.UpdateUserLastLogin(loginReq.Username); err != nil {
		s.logger.Error("Failed to update last login", "username", loginReq.Username, "error", err)
	}

	// Create session cookie
	sessionCookie := &http.Cookie{
		Name:     "session_user",
		Value:    loginReq.Username,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true if using HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7, // 7 days
	}
	http.SetCookie(w, sessionCookie)

	// Also set role cookie for frontend
	roleCookie := &http.Cookie{
		Name:     "session_role",
		Value:    user.Role,
		Path:     "/",
		HttpOnly: false, // Allow JS to read role for UI
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7,
	}
	http.SetCookie(w, roleCookie)

	s.logger.Info("User logged in", "username", loginReq.Username, "role", user.Role)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": user.Username,
		"role":     user.Role,
	})
}

// handleLogout processes logout requests
func (s *ComplianceServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Clear session cookies
	sessionCookie := &http.Cookie{
		Name:     "session_user",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, sessionCookie)

	roleCookie := &http.Cookie{
		Name:     "session_role",
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		MaxAge:   -1,
	}
	http.SetCookie(w, roleCookie)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// requireAuth middleware for web pages - redirects to login if not authenticated
func (s *ComplianceServer) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_user")
		if err != nil || cookie.Value == "" {
			// Not authenticated, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Verify user exists in database
		_, err = s.db.GetUser(cookie.Value)
		if err != nil {
			// Invalid session, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// User is authenticated, proceed
		next(w, r)
	}
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

// handleClientsPage serves the clients page
func (s *ComplianceServer) handleClientsPage(w http.ResponseWriter, r *http.Request) {
	// Read clients HTML file
	html, err := os.ReadFile("clients.html")
	if err != nil {
		s.logger.Error("Failed to read clients.html", "error", err)
		http.Error(w, "Clients page not available", http.StatusInternalServerError)
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

// handleAboutPage serves the about page
func (s *ComplianceServer) handleAboutPage(w http.ResponseWriter, r *http.Request) {
	// Read about HTML file
	html, err := os.ReadFile("about.html")
	if err != nil {
		s.logger.Error("Failed to read about.html", "error", err)
		http.Error(w, "About page not available", http.StatusInternalServerError)
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

		// Check for session authentication first (username/password login)
		if sessionCookie, err := r.Cookie("session_user"); err == nil && sessionCookie.Value != "" {
			// Verify session is valid
			if _, err := s.db.GetUser(sessionCookie.Value); err == nil {
				// Valid session, allow access
				next(w, r)
				return
			}
		}

		// Fall back to API key authentication
		var apiKey string

		// Try to get API key from cookie (for dashboard/web UI)
		if cookie, err := r.Cookie("api_token"); err == nil {
			apiKey = cookie.Value
		} else {
			// Fall back to Authorization header (for API clients)
			apiKey = r.Header.Get("Authorization")
			if apiKey == "" {
				s.sendError(w, http.StatusUnauthorized, "Authentication required")
				return
			}
			// Remove "Bearer " prefix if present
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}

		// Validate API key
		valid := s.validateAPIKey(apiKey)

		if !valid {
			s.logger.Warn("Invalid API key", "remote_addr", r.RemoteAddr)
			s.sendError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		next(w, r)
	}
}

// validateAPIKey checks if an API key is valid (checks database first, then config fallback)
func (s *ComplianceServer) validateAPIKey(apiKey string) bool {
	// First, check database for active API keys
	hashes, err := s.db.ListActiveAPIKeyHashes()
	if err != nil {
		s.logger.Error("Failed to list active API key hashes", "error", err)
		// Continue to config fallback if database fails
	} else {
		// Check against database hashes
		for _, hash := range hashes {
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey)); err == nil {
				// Update last_used timestamp asynchronously
				go func(keyHash string) {
					if err := s.db.UpdateAPIKeyLastUsed(keyHash); err != nil {
						s.logger.Warn("Failed to update API key last used", "error", err)
					}
				}(hash)
				return true
			}
		}
	}

	// Fall back to config-based keys for backwards compatibility
	// If using hashed keys in config, check against config hashes
	if s.config.Auth.UseHashedKeys {
		for _, hash := range s.config.Auth.APIKeyHashes {
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey)); err == nil {
				return true
			}
		}
		return false
	}

	// Fall back to plain text comparison in config (legacy)
	for _, key := range s.config.Auth.APIKeys {
		if apiKey == key {
			return true
		}
	}

	return false
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

// handleUsers lists all users
func (s *ComplianceServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	users, err := s.db.ListUsers()
	if err != nil {
		s.logger.Error("Failed to list users", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// handleCreateUser creates a new user
func (s *ComplianceServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate inputs
	if request.Username == "" || request.Password == "" || request.Role == "" {
		s.sendError(w, http.StatusBadRequest, "Username, password, and role are required")
		return
	}

	// Validate role
	validRoles := map[string]bool{"admin": true, "viewer": true, "auditor": true}
	if !validRoles[request.Role] {
		s.sendError(w, http.StatusBadRequest, "Invalid role. Must be: admin, viewer, or auditor")
		return
	}

	// Check if username already exists
	exists, err := s.db.UserExists(request.Username)
	if err != nil {
		s.logger.Error("Failed to check user existence", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if exists {
		s.sendError(w, http.StatusConflict, "Username already exists")
		return
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Create user
	if err := s.db.CreateUser(request.Username, string(passwordHash), request.Role); err != nil {
		s.logger.Error("Failed to create user", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	s.logger.Info("User created", "username", request.Username, "role", request.Role)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "User created successfully",
	})
}

// handleDeleteUser deletes a user
func (s *ComplianceServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if request.Username == "" {
		s.sendError(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Delete user
	if err := s.db.DeleteUser(request.Username); err != nil {
		if err.Error() == "user not found" {
			s.sendError(w, http.StatusNotFound, "User not found")
			return
		}
		s.logger.Error("Failed to delete user", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	s.logger.Info("User deleted", "username", request.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "User deleted successfully",
	})
}

// handleChangePassword changes a user's password
func (s *ComplianceServer) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Username    string `json:"username"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if request.Username == "" || request.NewPassword == "" {
		s.sendError(w, http.StatusBadRequest, "Username and new password are required")
		return
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}

	// Update password
	if err := s.db.UpdateUserPassword(request.Username, string(passwordHash)); err != nil {
		if err.Error() == "user not found" {
			s.sendError(w, http.StatusNotFound, "User not found")
			return
		}
		s.logger.Error("Failed to update password", "error", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}

	s.logger.Info("User password changed", "username", request.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Password changed successfully",
	})
}

// handleGetLoginMessage returns the configured login message (public endpoint)
func (s *ComplianceServer) handleGetLoginMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": s.config.Dashboard.LoginMessage,
	})
}

// handleUpdateLoginMessage updates the login message
func (s *ComplianceServer) handleUpdateLoginMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Update the login message in config (runtime only)
	s.config.Dashboard.LoginMessage = request.Message

	s.logger.Info("Login message updated", "message", request.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Login message updated successfully",
	})
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
	reportsDir := "configs/reports"
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

// generateSecureAPIKey generates a cryptographically secure random API key
func generateSecureAPIKey() (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64 URL-safe format (43 characters)
	key := base64.URLEncoding.EncodeToString(bytes)
	return key, nil
}

// handleListAPIKeys lists all API keys from database
func (s *ComplianceServer) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := s.db.ListAPIKeys()
	if err != nil {
		s.logger.Error("Failed to list API keys", "error", err)
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// handleGenerateAPIKey generates a new API key
func (s *ComplianceServer) handleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name      string  `json:"name"`
		ExpiresAt *string `json:"expires_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Get current user from session (if logged in)
	createdBy := "system"
	if sessionCookie, err := r.Cookie("session_user"); err == nil {
		createdBy = sessionCookie.Value
	}

	// Generate secure random API key
	apiKey, err := generateSecureAPIKey()
	if err != nil {
		s.logger.Error("Failed to generate API key", "error", err)
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Hash the key
	keyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash API key", "error", err)
		http.Error(w, "Failed to hash API key", http.StatusInternalServerError)
		return
	}

	// Store first 8 characters as prefix for display
	keyPrefix := apiKey[:8] + "..."

	// Save to database
	if err := s.db.CreateAPIKey(req.Name, string(keyHash), keyPrefix, createdBy, req.ExpiresAt); err != nil {
		s.logger.Error("Failed to save API key", "error", err)
		http.Error(w, "Failed to save API key", http.StatusInternalServerError)
		return
	}

	s.logger.Info("API key generated", "name", req.Name, "created_by", createdBy)

	// Return the full key ONLY once (never stored)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "API key created successfully",
		"api_key": apiKey, // Only time this is shown!
		"prefix":  keyPrefix,
		"name":    req.Name,
	})
}

// handleDeleteAPIKeyDB deletes an API key from database
func (s *ComplianceServer) handleDeleteAPIKeyDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteAPIKey(req.ID); err != nil {
		s.logger.Error("Failed to delete API key", "id", req.ID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "API key deleted successfully",
	})
}

// handleToggleAPIKey toggles an API key's active status
func (s *ComplianceServer) handleToggleAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     int  `json:"id"`
		Active bool `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var err error
	if req.Active {
		err = s.db.ActivateAPIKey(req.ID)
	} else {
		err = s.db.DeactivateAPIKey(req.ID)
	}

	if err != nil {
		s.logger.Error("Failed to toggle API key", "id", req.ID, "active", req.Active, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("API key %s successfully", map[bool]string{true: "activated", false: "deactivated"}[req.Active]),
	})
}
