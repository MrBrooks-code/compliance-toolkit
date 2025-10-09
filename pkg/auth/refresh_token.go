package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RefreshTokenManager handles refresh token storage and validation
type RefreshTokenManager struct {
	db        *sql.DB
	jwtConfig *JWTConfig
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID                string
	UserID            int
	TokenHash         string
	TokenFamily       string
	ExpiresAt         time.Time
	CreatedAt         time.Time
	LastUsed          *time.Time
	Revoked           bool
	RevokedAt         *time.Time
	RevokedReason     string
	UserAgent         string
	IPAddress         string
	DeviceFingerprint string
}

// NewRefreshTokenManager creates a new refresh token manager
func NewRefreshTokenManager(db *sql.DB, jwtConfig *JWTConfig) *RefreshTokenManager {
	return &RefreshTokenManager{
		db:        db,
		jwtConfig: jwtConfig,
	}
}

// StoreRefreshToken stores a refresh token in the database
func (m *RefreshTokenManager) StoreRefreshToken(ctx context.Context, userID int, token string, metadata TokenMetadata) (string, error) {
	// Extract claims from token to get token family and expiration
	claims, err := m.jwtConfig.ValidateRefreshToken(token)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Hash the token before storing (SHA-256 - tokens are already cryptographically signed)
	// We don't use bcrypt here because JWT tokens exceed bcrypt's 72-byte limit
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Generate UUID for token ID
	tokenID := uuid.New().String()

	// Insert into database
	query := `
		INSERT INTO refresh_tokens (
			id, user_id, token_hash, token_family, expires_at,
			user_agent, ip_address, device_fingerprint
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = m.db.ExecContext(ctx, query,
		tokenID,
		userID,
		string(tokenHash),
		claims.TokenFamily,
		claims.ExpiresAt.Time,
		metadata.UserAgent,
		metadata.IPAddress,
		metadata.DeviceFingerprint,
	)

	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return tokenID, nil
}

// ValidateRefreshToken validates a refresh token and returns the stored token record
func (m *RefreshTokenManager) ValidateRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	// First validate JWT signature and expiration
	claims, err := m.jwtConfig.ValidateRefreshToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Query database for all non-revoked tokens for this user in the same token family
	query := `
		SELECT id, user_id, token_hash, token_family, expires_at, created_at,
		       last_used, revoked, revoked_at, revoked_reason,
		       user_agent, ip_address, device_fingerprint
		FROM refresh_tokens
		WHERE user_id = ? AND token_family = ? AND revoked = 0
		ORDER BY created_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, claims.UserID, claims.TokenFamily)
	if err != nil {
		return nil, fmt.Errorf("failed to query refresh tokens: %w", err)
	}
	defer rows.Close()

	// Try to find matching token by comparing hash
	for rows.Next() {
		var rt RefreshToken
		var lastUsed, revokedAt sql.NullTime
		var revokedReason sql.NullString

		err := rows.Scan(
			&rt.ID, &rt.UserID, &rt.TokenHash, &rt.TokenFamily, &rt.ExpiresAt, &rt.CreatedAt,
			&lastUsed, &rt.Revoked, &revokedAt, &revokedReason,
			&rt.UserAgent, &rt.IPAddress, &rt.DeviceFingerprint,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %w", err)
		}

		// Convert nullable fields
		if lastUsed.Valid {
			rt.LastUsed = &lastUsed.Time
		}
		if revokedAt.Valid {
			rt.RevokedAt = &revokedAt.Time
		}
		if revokedReason.Valid {
			rt.RevokedReason = revokedReason.String
		}

		// Compare token hash using SHA-256
		hasher := sha256.New()
		hasher.Write([]byte(token))
		tokenHash := hex.EncodeToString(hasher.Sum(nil))
		if rt.TokenHash == tokenHash {
			// Token matches! Update last_used timestamp
			_, err = m.db.ExecContext(ctx, "UPDATE refresh_tokens SET last_used = ? WHERE id = ?", time.Now(), rt.ID)
			if err != nil {
				// Non-fatal, just log
				fmt.Printf("Warning: failed to update last_used timestamp: %v\n", err)
			}
			return &rt, nil
		}
	}

	// If we get here, token not found in database
	// This could indicate:
	// 1. Token was already used (rotation detection)
	// 2. Token was revoked
	// 3. Potential replay attack

	// Revoke entire token family as security measure
	if err := m.RevokeTokenFamily(ctx, claims.UserID, claims.TokenFamily, "token_reuse_detected"); err != nil {
		fmt.Printf("Warning: failed to revoke token family: %v\n", err)
	}

	return nil, fmt.Errorf("refresh token not found or already used (possible replay attack)")
}

// RevokeRefreshToken revokes a specific refresh token
func (m *RefreshTokenManager) RevokeRefreshToken(ctx context.Context, tokenID string, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = 1, revoked_at = ?, revoked_reason = ?
		WHERE id = ?
	`

	result, err := m.db.ExecContext(ctx, query, time.Now(), reason, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found: %s", tokenID)
	}

	return nil
}

// RevokeTokenFamily revokes all tokens in a token family (security incident)
func (m *RefreshTokenManager) RevokeTokenFamily(ctx context.Context, userID int, tokenFamily string, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = 1, revoked_at = ?, revoked_reason = ?
		WHERE user_id = ? AND token_family = ? AND revoked = 0
	`

	_, err := m.db.ExecContext(ctx, query, time.Now(), reason, userID, tokenFamily)
	if err != nil {
		return fmt.Errorf("failed to revoke token family: %w", err)
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user (logout all sessions)
func (m *RefreshTokenManager) RevokeAllUserTokens(ctx context.Context, userID int, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = 1, revoked_at = ?, revoked_reason = ?
		WHERE user_id = ? AND revoked = 0
	`

	_, err := m.db.ExecContext(ctx, query, time.Now(), reason, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired refresh tokens from the database
func (m *RefreshTokenManager) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < ? OR (revoked = 1 AND revoked_at < ?)
	`

	// Delete tokens that expired more than 30 days ago or were revoked more than 30 days ago
	cutoffTime := time.Now().Add(-30 * 24 * time.Hour)

	result, err := m.db.ExecContext(ctx, query, time.Now(), cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// RotateRefreshToken implements token rotation: validate old token and issue new one
func (m *RefreshTokenManager) RotateRefreshToken(ctx context.Context, oldToken string, user *User, metadata TokenMetadata) (*TokenPair, error) {
	// Validate the old refresh token
	storedToken, err := m.ValidateRefreshToken(ctx, oldToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	// Revoke the old token (it's been used)
	if err := m.RevokeRefreshToken(ctx, storedToken.ID, "rotation"); err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Generate new token pair with same token family (rotation chain)
	tokenPair, err := m.jwtConfig.GenerateTokenPair(user, storedToken.TokenFamily)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token pair: %w", err)
	}

	// Store new refresh token
	_, err = m.StoreRefreshToken(ctx, user.ID, tokenPair.RefreshToken, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	return tokenPair, nil
}

// TokenMetadata holds metadata for token storage
type TokenMetadata struct {
	UserAgent         string
	IPAddress         string
	DeviceFingerprint string
}

// GetUserActiveTokens returns all active (non-revoked, non-expired) tokens for a user
func (m *RefreshTokenManager) GetUserActiveTokens(ctx context.Context, userID int) ([]RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, token_family, expires_at, created_at,
		       last_used, revoked, revoked_at, revoked_reason,
		       user_agent, ip_address, device_fingerprint
		FROM refresh_tokens
		WHERE user_id = ? AND revoked = 0 AND expires_at > ?
		ORDER BY created_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query active tokens: %w", err)
	}
	defer rows.Close()

	var tokens []RefreshToken
	for rows.Next() {
		var rt RefreshToken
		var lastUsed, revokedAt sql.NullTime
		var revokedReason sql.NullString

		err := rows.Scan(
			&rt.ID, &rt.UserID, &rt.TokenHash, &rt.TokenFamily, &rt.ExpiresAt, &rt.CreatedAt,
			&lastUsed, &rt.Revoked, &revokedAt, &revokedReason,
			&rt.UserAgent, &rt.IPAddress, &rt.DeviceFingerprint,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		// Convert nullable fields
		if lastUsed.Valid {
			rt.LastUsed = &lastUsed.Time
		}
		if revokedAt.Valid {
			rt.RevokedAt = &revokedAt.Time
		}
		if revokedReason.Valid {
			rt.RevokedReason = revokedReason.String
		}

		tokens = append(tokens, rt)
	}

	return tokens, nil
}
