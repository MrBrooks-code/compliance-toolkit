package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	// Secret key for signing tokens (HS256)
	SecretKey string

	// Token lifetimes
	AccessTokenLifetime  time.Duration // Default: 15 minutes
	RefreshTokenLifetime time.Duration // Default: 7 days

	// Issuer and audience for token validation
	Issuer   string // Default: "compliance-toolkit"
	Audience string // Default: "compliance-api"
}

// CustomClaims represents JWT claims for access tokens
type CustomClaims struct {
	UserID      int      `json:"user_id"`
	Username    string   `json:"username"`
	Role        string   `json:"role"`        // "admin", "viewer", "analyst"
	Permissions []string `json:"permissions"` // granular permissions
	JWTVersion  int      `json:"jwt_version"` // for global token invalidation
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents JWT claims for refresh tokens
type RefreshTokenClaims struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	TokenFamily string `json:"token_family"` // for rotation tracking
	JWTVersion  int    `json:"jwt_version"`
	jwt.RegisteredClaims
}

// User represents a user for token generation
type User struct {
	ID          int
	Username    string
	Role        string
	Permissions []string
	JWTVersion  int
}

// NewJWTConfig creates a new JWT configuration with defaults
func NewJWTConfig(secretKey string) *JWTConfig {
	return &JWTConfig{
		SecretKey:            secretKey,
		AccessTokenLifetime:  15 * time.Minute,
		RefreshTokenLifetime: 7 * 24 * time.Hour, // 7 days
		Issuer:               "compliance-toolkit",
		Audience:             "compliance-api",
	}
}

// GenerateSecretKey generates a cryptographically secure random secret key
func GenerateSecretKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secret key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateAccessToken creates a new JWT access token for the given user
func (c *JWTConfig) GenerateAccessToken(user *User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	now := time.Now()
	jti := uuid.New().String()

	claims := CustomClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		JWTVersion:  user.JWTVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    c.Issuer,
			Audience:  jwt.ClaimStrings{c.Audience},
			Subject:   fmt.Sprintf("%d", user.ID),
			ExpiresAt: jwt.NewNumericDate(now.Add(c.AccessTokenLifetime)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(c.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken creates a new JWT refresh token for the given user
func (c *JWTConfig) GenerateRefreshToken(user *User, tokenFamily string) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	// Generate token family if not provided (new login session)
	if tokenFamily == "" {
		tokenFamily = uuid.New().String()
	}

	now := time.Now()
	jti := uuid.New().String()

	claims := RefreshTokenClaims{
		UserID:      user.ID,
		Username:    user.Username,
		TokenFamily: tokenFamily,
		JWTVersion:  user.JWTVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    c.Issuer,
			Audience:  jwt.ClaimStrings{c.Audience},
			Subject:   fmt.Sprintf("%d", user.ID),
			ExpiresAt: jwt.NewNumericDate(now.Add(c.RefreshTokenLifetime)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(c.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

// ValidateAccessToken validates and parses an access token
func (c *JWTConfig) ValidateAccessToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid access token claims")
	}

	// Validate issuer and audience
	if err := c.validateClaims(&claims.RegisteredClaims); err != nil {
		return nil, err
	}

	return claims, nil
}

// ValidateRefreshToken validates and parses a refresh token
func (c *JWTConfig) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	// Validate issuer and audience
	if err := c.validateClaims(&claims.RegisteredClaims); err != nil {
		return nil, err
	}

	return claims, nil
}

// ParseToken parses a JWT token without validation (for extracting JTI for blacklist)
func (c *JWTConfig) ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}

// ExtractJTI extracts the JTI (JWT ID) claim from a token string
func (c *JWTConfig) ExtractJTI(tokenString string) (string, error) {
	token, err := c.ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("failed to extract claims")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return "", fmt.Errorf("jti claim not found or invalid")
	}

	return jti, nil
}

// validateClaims validates standard JWT claims (issuer, audience, expiration)
func (c *JWTConfig) validateClaims(claims *jwt.RegisteredClaims) error {
	// Validate issuer
	if claims.Issuer != c.Issuer {
		return fmt.Errorf("invalid token issuer: expected %s, got %s", c.Issuer, claims.Issuer)
	}

	// Validate audience
	if len(claims.Audience) == 0 || claims.Audience[0] != c.Audience {
		return fmt.Errorf("invalid token audience")
	}

	// Expiration is already validated by jwt.Parse
	return nil
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"` // seconds until access token expires
	ExpiresAt    time.Time `json:"expires_at"` // absolute expiration time
}

// GenerateTokenPair generates both access and refresh tokens for a user
func (c *JWTConfig) GenerateTokenPair(user *User, tokenFamily string) (*TokenPair, error) {
	accessToken, err := c.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := c.GenerateRefreshToken(user, tokenFamily)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(c.AccessTokenLifetime.Seconds()),
		ExpiresAt:    time.Now().Add(c.AccessTokenLifetime),
	}, nil
}
