package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ServerConfig represents the server configuration
type ServerConfig struct {
	Server   ServerSettings   `mapstructure:"server"`
	Database DatabaseSettings `mapstructure:"database"`
	Auth     AuthSettings     `mapstructure:"auth"`
	Dashboard DashboardSettings `mapstructure:"dashboard"`
	Logging  LoggingSettings  `mapstructure:"logging"`
}

// ServerSettings contains HTTP server configuration
type ServerSettings struct {
	Host string     `mapstructure:"host"`
	Port int        `mapstructure:"port"`
	TLS  TLSSettings `mapstructure:"tls"`
}

// TLSSettings contains TLS/HTTPS configuration
type TLSSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseSettings contains database configuration (PostgreSQL only)
type DatabaseSettings struct {
	Type     string `mapstructure:"type"`     // postgres (SQLite support removed)
	Host     string `mapstructure:"host"`     // PostgreSQL host
	Port     int    `mapstructure:"port"`     // PostgreSQL port
	Name     string `mapstructure:"name"`     // Database name
	User     string `mapstructure:"user"`     // Database user
	Password string `mapstructure:"password"` // Database password
	SSLMode  string `mapstructure:"sslmode"`  // SSL mode (disable, require, verify-ca, verify-full)
}

// AuthSettings contains authentication configuration
type AuthSettings struct {
	Enabled       bool     `mapstructure:"enabled"`

	// DEPRECATED: Static API keys in configuration will be removed in v2.0
	// Use database-backed API keys via /api/v1/apikeys instead
	// Security issues: no auditing, no rotation, easily leaked in version control
	APIKeys       []string `mapstructure:"api_keys"`        // DEPRECATED - Plain text keys (DO NOT USE)
	APIKeyHashes  []string `mapstructure:"api_key_hashes"`  // DEPRECATED - Bcrypt hashed keys (DO NOT USE)
	UseHashedKeys bool     `mapstructure:"use_hashed_keys"` // DEPRECATED - Whether to use hashed keys

	RequireKey    bool     `mapstructure:"require_key"`     // Set to true to enforce authentication
	JWT           JWTAuthSettings `mapstructure:"jwt"`       // JWT authentication settings
}

// JWTAuthSettings contains JWT-specific authentication configuration
type JWTAuthSettings struct {
	Enabled              bool   `mapstructure:"enabled"`                // Enable JWT authentication
	SecretKey            string `mapstructure:"secret_key"`             // Secret key for signing tokens (auto-generated if empty)
	AccessTokenLifetime  int    `mapstructure:"access_token_lifetime"`  // Access token lifetime in minutes (default: 15)
	RefreshTokenLifetime int    `mapstructure:"refresh_token_lifetime"` // Refresh token lifetime in days (default: 7)
	Issuer               string `mapstructure:"issuer"`                 // Token issuer (default: compliance-toolkit)
	Audience             string `mapstructure:"audience"`               // Token audience (default: compliance-api)
}

// DashboardSettings contains web dashboard configuration
type DashboardSettings struct {
	Enabled      bool   `mapstructure:"enabled"`
	Path         string `mapstructure:"path"`          // URL path for dashboard
	LoginMessage string `mapstructure:"login_message"` // Message displayed on login page
}

// LoggingSettings contains logging configuration
type LoggingSettings struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // text, json
	OutputPath string `mapstructure:"output_path"` // stdout, stderr, or file path
}

// LoadServerConfig loads configuration from file
func LoadServerConfig(configPath string) (*ServerConfig, error) {
	v := viper.New()

	// Set defaults
	setConfigDefaults(v)

	// Determine config file path
	if configPath != "" {
		// Use specified config file
		v.SetConfigFile(configPath)
	} else {
		// Look for server.yaml in current directory
		v.SetConfigName("server")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file not found - use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Use all defaults
			return unmarshalConfig(v)
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return unmarshalConfig(v)
}

// setConfigDefaults sets default values
func setConfigDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8443)
	v.SetDefault("server.tls.enabled", true)
	v.SetDefault("server.tls.cert_file", "certs/server.crt")
	v.SetDefault("server.tls.key_file", "certs/server.key")

	// Database defaults (PostgreSQL only)
	v.SetDefault("database.type", "postgres")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "compliance")
	v.SetDefault("database.user", "compliance")
	v.SetDefault("database.password", "compliance")
	v.SetDefault("database.sslmode", "disable")

	// Auth defaults
	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.require_key", true)
	v.SetDefault("auth.api_keys", []string{})
	v.SetDefault("auth.api_key_hashes", []string{})
	v.SetDefault("auth.use_hashed_keys", false) // Default to false for backwards compatibility

	// JWT defaults
	v.SetDefault("auth.jwt.enabled", true) // Enabled by default (migration complete)
	v.SetDefault("auth.jwt.secret_key", "") // Auto-generated on first run if empty
	v.SetDefault("auth.jwt.access_token_lifetime", 15) // 15 minutes
	v.SetDefault("auth.jwt.refresh_token_lifetime", 7) // 7 days
	v.SetDefault("auth.jwt.issuer", "ComplianceToolkit")
	v.SetDefault("auth.jwt.audience", "ComplianceToolkit")

	// Dashboard defaults
	v.SetDefault("dashboard.enabled", true)
	v.SetDefault("dashboard.path", "/dashboard")
	v.SetDefault("dashboard.login_message", "Welcome to Compliance Toolkit")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output_path", "stdout")
}

// unmarshalConfig unmarshals viper config into ServerConfig
func unmarshalConfig(v *viper.Viper) (*ServerConfig, error) {
	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	// Validate port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	// Validate TLS settings
	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			return fmt.Errorf("TLS enabled but cert_file not specified")
		}
		if c.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but key_file not specified")
		}
	}

	// Validate database settings (PostgreSQL only)
	if c.Database.Type != "postgres" {
		return fmt.Errorf("database type must be 'postgres' (SQLite support removed)")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	// Validate auth settings
	// NOTE: Static API keys (c.Auth.APIKeys) are DEPRECATED
	// With JWT enabled, users can log in and create database-backed API keys
	// Only fail if auth is required but BOTH static keys AND JWT are disabled
	if c.Auth.Enabled && c.Auth.RequireKey {
		hasStaticKeys := len(c.Auth.APIKeys) > 0 || len(c.Auth.APIKeyHashes) > 0
		hasJWT := c.Auth.JWT.Enabled

		if !hasStaticKeys && !hasJWT {
			return fmt.Errorf("auth enabled with require_key but neither static API keys nor JWT authentication is configured")
		}
	}

	return nil
}

// generateDefaultConfig generates a default configuration file
func generateDefaultConfig(configPath string) error {
	path := getConfigPath(configPath)

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate YAML content
	content := `# Compliance Toolkit Server Configuration

# HTTP/HTTPS server settings
server:
  host: "0.0.0.0"       # Bind to all interfaces
  port: 8080            # HTTP port (use 8443 for HTTPS)
  tls:
    enabled: false      # Set to true for HTTPS
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

# Database configuration (PostgreSQL only)
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  name: "compliance"
  user: "compliance"
  password: "compliance"
  sslmode: "disable"    # disable, require, verify-ca, verify-full

# Authentication settings
auth:
  enabled: true
  require_key: true

  # JWT authentication (recommended)
  jwt:
    enabled: true        # JWT enabled by default
    secret_key: ""       # Auto-generated on first run if empty
    access_token_lifetime: 15   # Access token lifetime in minutes
    refresh_token_lifetime: 7   # Refresh token lifetime in days
    issuer: "ComplianceToolkit"
    audience: "ComplianceToolkit"

# Web dashboard
dashboard:
  enabled: true
  path: "/dashboard"    # URL path for dashboard

# Logging configuration
logging:
  level: "info"         # debug, info, warn, error
  format: "text"        # text, json
  output_path: "stdout" # stdout, stderr, or file path
`

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
