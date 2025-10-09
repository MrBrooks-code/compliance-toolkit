package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"compliancetoolkit/pkg/auth"
)

// initializeJWT initializes JWT authentication components
func (s *ComplianceServer) initializeJWT() error {
	if !s.config.Auth.JWT.Enabled {
		s.logger.Info("JWT authentication is disabled")
		return nil
	}

	s.logger.Info("Initializing JWT authentication...")

	// Generate or load JWT secret key
	secretKey := s.config.Auth.JWT.SecretKey
	if secretKey == "" {
		// Auto-generate secret key
		var err error
		secretKey, err = auth.GenerateSecretKey()
		if err != nil {
			return fmt.Errorf("failed to generate JWT secret key: %w", err)
		}

		// Store in config for this session
		s.config.Auth.JWT.SecretKey = secretKey

		s.logger.Warn("Auto-generated JWT secret key",
			"warning", "Store this in server.yaml to persist across restarts",
			"secret_prefix", secretKey[:16]+"...",
		)
	}

	// Create JWT config
	s.jwtConfig = auth.NewJWTConfig(secretKey)

	// Apply custom lifetimes if configured
	if s.config.Auth.JWT.AccessTokenLifetime > 0 {
		s.jwtConfig.AccessTokenLifetime = time.Duration(s.config.Auth.JWT.AccessTokenLifetime) * time.Minute
	}
	if s.config.Auth.JWT.RefreshTokenLifetime > 0 {
		s.jwtConfig.RefreshTokenLifetime = time.Duration(s.config.Auth.JWT.RefreshTokenLifetime) * 24 * time.Hour
	}
	if s.config.Auth.JWT.Issuer != "" {
		s.jwtConfig.Issuer = s.config.Auth.JWT.Issuer
	}
	if s.config.Auth.JWT.Audience != "" {
		s.jwtConfig.Audience = s.config.Auth.JWT.Audience
	}

	// Initialize JWT handlers
	s.jwtHandlers = auth.NewAuthHandlers(s.db.db, s.jwtConfig)

	// Initialize JWT middleware
	s.jwtMiddleware = auth.NewMiddleware(s.jwtConfig, s.db.db)

	s.logger.Info("JWT authentication initialized",
		"access_token_lifetime", s.jwtConfig.AccessTokenLifetime,
		"refresh_token_lifetime", s.jwtConfig.RefreshTokenLifetime,
		"issuer", s.jwtConfig.Issuer,
		"audience", s.jwtConfig.Audience,
	)

	return nil
}

// registerJWTRoutes registers JWT authentication endpoints
func (s *ComplianceServer) registerJWTRoutes() {
	if !s.config.Auth.JWT.Enabled || s.jwtHandlers == nil {
		return
	}

	// JWT authentication endpoints (public)
	s.mux.HandleFunc("/api/auth/login", s.jwtHandlers.Login)
	s.mux.HandleFunc("/api/auth/refresh", s.jwtHandlers.Refresh)

	// Protected endpoints (require JWT token)
	s.mux.Handle("/api/auth/logout", s.jwtMiddleware.RequireAuth(http.HandlerFunc(s.jwtHandlers.Logout)))
	s.mux.Handle("/api/auth/me", s.jwtMiddleware.RequireAuth(http.HandlerFunc(s.jwtHandlers.Me)))

	s.logger.Info("JWT endpoints registered",
		"endpoints", []string{
			"POST /api/auth/login",
			"POST /api/auth/refresh",
			"POST /api/auth/logout",
			"GET /api/auth/me",
		},
	)
}

// startCleanupTasks starts background cleanup tasks
func (s *ComplianceServer) startCleanupTasks() {
	if !s.config.Auth.JWT.Enabled {
		return
	}

	s.logger.Info("Starting background cleanup tasks...")

	// Cleanup expired tokens every hour
	go s.cleanupExpiredTokens()

	// Cleanup JWT blacklist every hour
	go s.cleanupJWTBlacklist()

	// Cleanup old audit logs every day
	go s.cleanupOldAuditLogs()

	s.logger.Info("Cleanup tasks started")
}

// cleanupExpiredTokens periodically removes expired refresh tokens
func (s *ComplianceServer) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		refreshManager := auth.NewRefreshTokenManager(s.db.db, s.jwtConfig)
		count, err := refreshManager.CleanupExpiredTokens(context.Background())
		if err != nil {
			s.logger.Error("Failed to cleanup expired tokens", "error", err)
		} else if count > 0 {
			s.logger.Info("Cleaned up expired refresh tokens", "count", count)
		}
	}
}

// cleanupJWTBlacklist periodically removes expired entries from JWT blacklist
func (s *ComplianceServer) cleanupJWTBlacklist() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		blacklistManager := auth.NewBlacklistManager(s.db.db)
		count, err := blacklistManager.CleanupExpiredEntries(context.Background())
		if err != nil {
			s.logger.Error("Failed to cleanup JWT blacklist", "error", err)
		} else if count > 0 {
			s.logger.Info("Cleaned up expired JWT blacklist entries", "count", count)
		}
	}
}

// cleanupOldAuditLogs periodically removes old audit log entries
func (s *ComplianceServer) cleanupOldAuditLogs() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Keep audit logs for 90 days
	retentionPeriod := 90 * 24 * time.Hour

	for range ticker.C {
		auditLogger := auth.NewAuditLogger(s.db.db)
		count, err := auditLogger.CleanupOldEntries(context.Background(), retentionPeriod)
		if err != nil {
			s.logger.Error("Failed to cleanup old audit logs", "error", err)
		} else if count > 0 {
			s.logger.Info("Cleaned up old audit log entries", "count", count, "retention_days", 90)
		}
	}
}
