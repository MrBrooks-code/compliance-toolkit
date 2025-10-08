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

// DatabaseSettings contains database configuration
type DatabaseSettings struct {
	Type string `mapstructure:"type"` // sqlite, postgres (future)
	Path string `mapstructure:"path"` // For SQLite
	DSN  string `mapstructure:"dsn"`  // For other databases (future)
}

// AuthSettings contains authentication configuration
type AuthSettings struct {
	Enabled       bool     `mapstructure:"enabled"`
	APIKeys       []string `mapstructure:"api_keys"`        // Plain text keys (legacy)
	APIKeyHashes  []string `mapstructure:"api_key_hashes"`  // Bcrypt hashed keys (recommended)
	RequireKey    bool     `mapstructure:"require_key"`
	UseHashedKeys bool     `mapstructure:"use_hashed_keys"` // Whether to use hashed keys
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

	// Database defaults
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.path", "data/compliance.db")

	// Auth defaults
	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.require_key", true)
	v.SetDefault("auth.api_keys", []string{})
	v.SetDefault("auth.api_key_hashes", []string{})
	v.SetDefault("auth.use_hashed_keys", false) // Default to false for backwards compatibility

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

	// Validate database settings
	if c.Database.Type == "" {
		return fmt.Errorf("database type is required")
	}
	if c.Database.Type == "sqlite" && c.Database.Path == "" {
		return fmt.Errorf("database path is required for SQLite")
	}

	// Validate auth settings
	if c.Auth.Enabled && c.Auth.RequireKey && len(c.Auth.APIKeys) == 0 {
		return fmt.Errorf("auth enabled with require_key but no API keys configured")
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
  port: 8443            # HTTPS port
  tls:
    enabled: true
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

# Database configuration
database:
  type: "sqlite"        # Currently only SQLite supported
  path: "data/compliance.db"
  # dsn: ""             # For future PostgreSQL support

# Authentication settings
auth:
  enabled: true
  require_key: true
  api_keys:
    - "your-api-key-here"
    # - "another-api-key"

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
