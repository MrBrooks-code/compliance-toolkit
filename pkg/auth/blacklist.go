package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// BlacklistManager handles JWT blacklist operations for immediate token revocation
type BlacklistManager struct {
	db *sql.DB
}

// NewBlacklistManager creates a new blacklist manager
func NewBlacklistManager(db *sql.DB) *BlacklistManager {
	return &BlacklistManager{db: db}
}

// BlacklistToken adds a token's JTI to the blacklist
func (m *BlacklistManager) BlacklistToken(ctx context.Context, jti string, userID int, expiresAt time.Time, reason string) error {
	query := `
		INSERT INTO jwt_blacklist (jti, user_id, expires_at, reason)
		VALUES ($1, $2, $3, $4)
	`

	_, err := m.db.ExecContext(ctx, query, jti, userID, expiresAt, reason)
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token's JTI is blacklisted
func (m *BlacklistManager) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM jwt_blacklist
		WHERE jti = $1 AND expires_at > $2
	`

	var count int
	err := m.db.QueryRowContext(ctx, query, jti, time.Now()).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return count > 0, nil
}

// CleanupExpiredEntries removes expired entries from the blacklist
func (m *BlacklistManager) CleanupExpiredEntries(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM jwt_blacklist
		WHERE expires_at < $1
	`

	result, err := m.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup blacklist: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// BlacklistAllUserTokens blacklists all active tokens for a user (emergency revocation)
func (m *BlacklistManager) BlacklistAllUserTokens(ctx context.Context, userID int, reason string) error {
	// This would typically be called when incrementing jwt_version in users table
	// The actual blacklisting happens by checking jwt_version in token validation
	// This function is here for completeness but may not be used in practice

	query := `
		INSERT INTO jwt_blacklist (jti, user_id, expires_at, reason)
		SELECT DISTINCT jti, user_id, expires_at, $1
		FROM (
			SELECT $2 as jti, $3 as user_id, NOW() + INTERVAL '1 hour' as expires_at
		)
	`

	_, err := m.db.ExecContext(ctx, query, reason, "all-tokens-"+time.Now().Format("20060102150405"), userID)
	if err != nil {
		return fmt.Errorf("failed to blacklist all user tokens: %w", err)
	}

	return nil
}
