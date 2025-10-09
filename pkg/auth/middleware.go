package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Middleware provides JWT authentication middleware for HTTP handlers
type Middleware struct {
	jwtConfig        *JWTConfig
	blacklistManager *BlacklistManager
	db               *sql.DB
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(jwtConfig *JWTConfig, db *sql.DB) *Middleware {
	return &Middleware{
		jwtConfig:        jwtConfig,
		blacklistManager: NewBlacklistManager(db),
		db:               db,
	}
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserClaimsKey is the context key for storing user claims
	UserClaimsKey contextKey = "user_claims"
	// UserIDKey is the context key for storing user ID
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for storing username
	UsernameKey contextKey = "username"
)

// RequireAuth is middleware that requires a valid JWT access token
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token := extractTokenFromHeader(r)
		if token == "" {
			respondUnauthorized(w, "missing or invalid authorization header")
			return
		}

		// Validate token
		claims, err := m.jwtConfig.ValidateAccessToken(token)
		if err != nil {
			respondUnauthorized(w, fmt.Sprintf("invalid token: %v", err))
			return
		}

		// Check if token is blacklisted
		isBlacklisted, err := m.blacklistManager.IsTokenBlacklisted(r.Context(), claims.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to check token blacklist")
			return
		}
		if isBlacklisted {
			respondUnauthorized(w, "token has been revoked")
			return
		}

		// Check user's JWT version (for global token invalidation)
		currentJWTVersion, err := m.getUserJWTVersion(r.Context(), claims.UserID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to verify user session")
			return
		}
		if claims.JWTVersion != currentJWTVersion {
			respondUnauthorized(w, "token version mismatch (session invalidated)")
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is middleware that requires a specific role
func (m *Middleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			if claims == nil {
				respondUnauthorized(w, "authentication required")
				return
			}

			if claims.Role != role {
				respondForbidden(w, fmt.Sprintf("role '%s' required", role))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission is middleware that requires a specific permission
func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			if claims == nil {
				respondUnauthorized(w, "authentication required")
				return
			}

			if !hasPermission(claims.Permissions, permission) {
				respondForbidden(w, fmt.Sprintf("permission '%s' required", permission))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth is middleware that extracts token if present but doesn't require it
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractTokenFromHeader(r)
		if token == "" {
			// No token present, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Validate token if present
		claims, err := m.jwtConfig.ValidateAccessToken(token)
		if err != nil {
			// Invalid token, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// GetUserClaims extracts user claims from request context
func GetUserClaims(ctx context.Context) *CustomClaims {
	claims, ok := ctx.Value(UserClaimsKey).(*CustomClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetUserID extracts user ID from request context
func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

// GetUsername extracts username from request context
func GetUsername(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UsernameKey).(string)
	return username, ok
}

// getUserJWTVersion retrieves the current JWT version for a user from database
func (m *Middleware) getUserJWTVersion(ctx context.Context, userID int) (int, error) {
	var jwtVersion int
	query := "SELECT jwt_version FROM users WHERE id = ?"
	err := m.db.QueryRowContext(ctx, query, userID).Scan(&jwtVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get user JWT version: %w", err)
	}
	return jwtVersion, nil
}

// hasPermission checks if a user has a specific permission
func hasPermission(permissions []string, required string) bool {
	for _, perm := range permissions {
		if perm == required || perm == "*" {
			return true
		}
	}
	return false
}

// HTTP response helpers

func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	respondJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "unauthorized",
		"message": message,
	})
}

func respondForbidden(w http.ResponseWriter, message string) {
	respondJSON(w, http.StatusForbidden, map[string]string{
		"error": "forbidden",
		"message": message,
	})
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, map[string]string{
		"error": "error",
		"message": message,
	})
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback error response
		fmt.Fprintf(w, `{"error":"encoding_error","message":"failed to encode response"}`)
	}
}
