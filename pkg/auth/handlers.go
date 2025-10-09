package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthHandlers provides HTTP handlers for authentication endpoints
type AuthHandlers struct {
	db                   *sql.DB
	jwtConfig            *JWTConfig
	refreshTokenManager  *RefreshTokenManager
	blacklistManager     *BlacklistManager
	auditLogger          *AuditLogger
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(db *sql.DB, jwtConfig *JWTConfig) *AuthHandlers {
	return &AuthHandlers{
		db:                  db,
		jwtConfig:           jwtConfig,
		refreshTokenManager: NewRefreshTokenManager(db, jwtConfig),
		blacklistManager:    NewBlacklistManager(db),
		auditLogger:         NewAuditLogger(db),
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserInfo  `json:"user"`
}

// UserInfo represents user information in responses
type UserInfo struct {
	ID       int      `json:"id"`
	Username string   `json:"username"`
	Role     string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Login handles user login and returns JWT tokens
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Query user from database
	user, err := h.getUserByUsername(r.Context(), req.Username)
	if err != nil {
		// Log failed login
		_ = h.auditLogger.LogFailedLogin(r.Context(), req.Username, "user not found", r.RemoteAddr, r.UserAgent())
		respondUnauthorized(w, "invalid credentials")
		return
	}

	// Check if account is locked
	if user.AccountLockedUntil != nil && user.AccountLockedUntil.After(time.Now()) {
		_ = h.auditLogger.LogFailedLogin(r.Context(), req.Username, "account locked", r.RemoteAddr, r.UserAgent())
		respondError(w, http.StatusForbidden, fmt.Sprintf("account locked until %s", user.AccountLockedUntil.Format(time.RFC3339)))
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed login attempts
		_ = h.incrementFailedLoginAttempts(r.Context(), user.ID)

		// Log failed login
		_ = h.auditLogger.LogFailedLogin(r.Context(), req.Username, "invalid password", r.RemoteAddr, r.UserAgent())
		respondUnauthorized(w, "invalid credentials")
		return
	}

	// Reset failed login attempts on successful login
	_ = h.resetFailedLoginAttempts(r.Context(), user.ID)

	// Update password_changed_at if not set
	if user.PasswordChangedAt == nil {
		now := time.Now()
		user.PasswordChangedAt = &now
		_ = h.updatePasswordChangedAt(r.Context(), user.ID, now)
	}

	// Generate token pair
	tokenPair, err := h.jwtConfig.GenerateTokenPair(&User{
		ID:          user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		JWTVersion:  user.JWTVersion,
	}, "") // Empty token family = new session

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	// Store refresh token in database
	_, err = h.refreshTokenManager.StoreRefreshToken(r.Context(), user.ID, tokenPair.RefreshToken, TokenMetadata{
		UserAgent:         r.UserAgent(),
		IPAddress:         r.RemoteAddr,
		DeviceFingerprint: extractDeviceFingerprint(r),
	})

	if err != nil {
		fmt.Printf("ERROR: Failed to store refresh token: %v\n", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to store refresh token: %v", err))
		return
	}

	// Log successful login
	_ = h.auditLogger.LogLogin(r.Context(), user.ID, user.Username, AuthMethodJWT, r.RemoteAddr, r.UserAgent())

	// Respond with tokens
	respondJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
		ExpiresAt:    tokenPair.ExpiresAt,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			Permissions: user.Permissions,
		},
	})
}

// Refresh handles token refresh
func (h *AuthHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse request body
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	// Validate refresh token
	claims, err := h.jwtConfig.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		respondUnauthorized(w, "invalid refresh token")
		return
	}

	// Get user from database
	user, err := h.getUserByID(r.Context(), claims.UserID)
	if err != nil {
		respondUnauthorized(w, "user not found")
		return
	}

	// Rotate refresh token (validates old token and issues new one)
	tokenPair, err := h.refreshTokenManager.RotateRefreshToken(r.Context(), req.RefreshToken, &User{
		ID:          user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		JWTVersion:  user.JWTVersion,
	}, TokenMetadata{
		UserAgent:         r.UserAgent(),
		IPAddress:         r.RemoteAddr,
		DeviceFingerprint: extractDeviceFingerprint(r),
	})

	if err != nil {
		respondUnauthorized(w, fmt.Sprintf("failed to rotate token: %v", err))
		return
	}

	// Log token refresh
	_ = h.auditLogger.LogTokenRefresh(r.Context(), user.ID, user.Username, r.RemoteAddr, r.UserAgent())

	// Respond with new tokens
	respondJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
		ExpiresAt:    tokenPair.ExpiresAt,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// Logout handles user logout (revokes refresh tokens)
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get user from context (requires authentication middleware)
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondUnauthorized(w, "authentication required")
		return
	}

	username, _ := GetUsername(r.Context())

	// Parse request body for refresh token (optional)
	var req struct {
		RefreshToken string `json:"refresh_token,omitempty"`
		All          bool   `json:"all,omitempty"` // Logout all sessions
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.All {
		// Revoke all refresh tokens for user
		if err := h.refreshTokenManager.RevokeAllUserTokens(r.Context(), userID, "logout_all"); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to revoke tokens")
			return
		}
	} else if req.RefreshToken != "" {
		// Revoke specific refresh token
		claims, err := h.jwtConfig.ValidateRefreshToken(req.RefreshToken)
		if err == nil {
			// Extract token ID from database
			storedToken, err := h.refreshTokenManager.ValidateRefreshToken(r.Context(), req.RefreshToken)
			if err == nil {
				_ = h.refreshTokenManager.RevokeRefreshToken(r.Context(), storedToken.ID, "logout")
			}
		}
		_ = claims // Unused
	}

	// Blacklist current access token (extract from header)
	accessToken := extractTokenFromHeader(r)
	if accessToken != "" {
		jti, err := h.jwtConfig.ExtractJTI(accessToken)
		if err == nil {
			accessClaims, _ := h.jwtConfig.ValidateAccessToken(accessToken)
			if accessClaims != nil {
				_ = h.blacklistManager.BlacklistToken(r.Context(), jti, userID, accessClaims.ExpiresAt.Time, "logout")
			}
		}
	}

	// Log logout
	_ = h.auditLogger.LogLogout(r.Context(), userID, username, r.RemoteAddr, r.UserAgent())

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// Me returns information about the current authenticated user
func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims := GetUserClaims(r.Context())
	if claims == nil {
		respondUnauthorized(w, "authentication required")
		return
	}

	respondJSON(w, http.StatusOK, UserInfo{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		Permissions: claims.Permissions,
	})
}

// Helper functions

// DBUser represents a user record from the database
type DBUser struct {
	ID                  int
	Username            string
	PasswordHash        string
	Role                string
	Permissions         []string
	JWTVersion          int
	PasswordChangedAt   *time.Time
	FailedLoginAttempts int
	AccountLockedUntil  *time.Time
}

func (h *AuthHandlers) getUserByUsername(ctx context.Context, username string) (*DBUser, error) {
	query := `
		SELECT id, username, password_hash, role, jwt_version,
		       password_changed_at, failed_login_attempts, account_locked_until
		FROM users
		WHERE username = ?
	`

	var user DBUser
	var passwordChangedAt, accountLockedUntil sql.NullTime

	err := h.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.JWTVersion,
		&passwordChangedAt,
		&user.FailedLoginAttempts,
		&accountLockedUntil,
	)

	if err != nil {
		return nil, err
	}

	if passwordChangedAt.Valid {
		user.PasswordChangedAt = &passwordChangedAt.Time
	}
	if accountLockedUntil.Valid {
		user.AccountLockedUntil = &accountLockedUntil.Time
	}

	// For simplicity, assume all users have basic permissions
	// In production, fetch from permissions table
	user.Permissions = []string{"read", "write"}

	return &user, nil
}

func (h *AuthHandlers) getUserByID(ctx context.Context, userID int) (*DBUser, error) {
	query := `
		SELECT id, username, password_hash, role, jwt_version,
		       password_changed_at, failed_login_attempts, account_locked_until
		FROM users
		WHERE id = ?
	`

	var user DBUser
	var passwordChangedAt, accountLockedUntil sql.NullTime

	err := h.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.JWTVersion,
		&passwordChangedAt,
		&user.FailedLoginAttempts,
		&accountLockedUntil,
	)

	if err != nil {
		return nil, err
	}

	if passwordChangedAt.Valid {
		user.PasswordChangedAt = &passwordChangedAt.Time
	}
	if accountLockedUntil.Valid {
		user.AccountLockedUntil = &accountLockedUntil.Time
	}

	user.Permissions = []string{"read", "write"}

	return &user, nil
}

func (h *AuthHandlers) incrementFailedLoginAttempts(ctx context.Context, userID int) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    account_locked_until = CASE
		        WHEN failed_login_attempts >= 4 THEN datetime('now', '+30 minutes')
		        ELSE account_locked_until
		    END
		WHERE id = ?
	`

	_, err := h.db.ExecContext(ctx, query, userID)
	return err
}

func (h *AuthHandlers) resetFailedLoginAttempts(ctx context.Context, userID int) error {
	query := "UPDATE users SET failed_login_attempts = 0, account_locked_until = NULL WHERE id = ?"
	_, err := h.db.ExecContext(ctx, query, userID)
	return err
}

func (h *AuthHandlers) updatePasswordChangedAt(ctx context.Context, userID int, changedAt time.Time) error {
	query := "UPDATE users SET password_changed_at = ? WHERE id = ?"
	_, err := h.db.ExecContext(ctx, query, changedAt, userID)
	return err
}

func extractDeviceFingerprint(r *http.Request) string {
	// Simple device fingerprint based on User-Agent
	// In production, use more sophisticated fingerprinting
	return r.UserAgent()
}
