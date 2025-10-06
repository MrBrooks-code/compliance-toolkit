package pkg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration
// with support for YAML files, environment variables, and CLI flags.
// Precedence order: CLI flags > ENV vars > YAML config > defaults
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Reports  ReportsConfig  `mapstructure:"reports"`
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig contains server/runtime configuration
type ServerConfig struct {
	// Host for future HTTP server functionality (reserved for future use)
	Host string `mapstructure:"host"`
	// Port for future HTTP server functionality (reserved for future use)
	Port int `mapstructure:"port"`
	// ReadTimeout for registry operations
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	// MaxConcurrentReads limits concurrent registry reads
	MaxConcurrentReads int `mapstructure:"max_concurrent_reads"`
	// GracefulShutdownTimeout for cleanup operations
	GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	// Level: debug, info, warn, error
	Level string `mapstructure:"level"`
	// Format: json, text
	Format string `mapstructure:"format"`
	// OutputPath: filepath or "stdout"/"stderr"
	OutputPath string `mapstructure:"output_path"`
	// EnableFileLogging writes logs to files in output/logs/
	EnableFileLogging bool `mapstructure:"enable_file_logging"`
	// MaxLogFileSizeMB before rotation (0 = no rotation)
	MaxLogFileSizeMB int `mapstructure:"max_log_file_size_mb"`
	// MaxLogFileAgeDays for log retention (0 = keep forever)
	MaxLogFileAgeDays int `mapstructure:"max_log_file_age_days"`
}

// ReportsConfig contains report generation configuration
type ReportsConfig struct {
	// ConfigPath is the directory containing report configs
	ConfigPath string `mapstructure:"config_path"`
	// OutputPath is where HTML reports are saved
	OutputPath string `mapstructure:"output_path"`
	// EvidencePath is where JSON evidence logs are saved
	EvidencePath string `mapstructure:"evidence_path"`
	// TemplatePath for custom HTML templates (optional, uses embedded by default)
	TemplatePath string `mapstructure:"template_path"`
	// EnableEvidence controls JSON evidence logging
	EnableEvidence bool `mapstructure:"enable_evidence"`
	// EnableDarkMode in HTML reports
	EnableDarkMode bool `mapstructure:"enable_dark_mode"`
	// Parallel controls concurrent report generation
	Parallel bool `mapstructure:"parallel"`
	// MaxParallelReports limits concurrent report generation (0 = CPU count)
	MaxParallelReports int `mapstructure:"max_parallel_reports"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	// RequireAdminPrivileges enforces administrator check on startup
	RequireAdminPrivileges bool `mapstructure:"require_admin_privileges"`
	// AllowedRegistryRoots restricts which registry hives can be accessed
	AllowedRegistryRoots []string `mapstructure:"allowed_registry_roots"`
	// DenyRegistryPaths blocks specific registry paths (security-sensitive keys)
	DenyRegistryPaths []string `mapstructure:"deny_registry_paths"`
	// ReadOnly enforces read-only mode (always true for compliance scanner)
	ReadOnly bool `mapstructure:"read_only"`
	// AuditMode logs all registry access attempts
	AuditMode bool `mapstructure:"audit_mode"`
	// AuditLogPath is the directory where audit logs are stored
	AuditLogPath string `mapstructure:"audit_log_path"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:                    "localhost",
			Port:                    8080,
			ReadTimeout:             5 * time.Second,
			MaxConcurrentReads:      10,
			GracefulShutdownTimeout: 30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:             "info",
			Format:            "json",
			OutputPath:        "stdout",
			EnableFileLogging: true,
			MaxLogFileSizeMB:  100,
			MaxLogFileAgeDays: 30,
		},
		Reports: ReportsConfig{
			ConfigPath:         "configs/reports",
			OutputPath:         "output/reports",
			EvidencePath:       "output/evidence",
			TemplatePath:       "", // Empty = use embedded templates
			EnableEvidence:     true,
			EnableDarkMode:     true,
			Parallel:           false,
			MaxParallelReports: 0, // 0 = use runtime.NumCPU()
		},
		Security: SecurityConfig{
			RequireAdminPrivileges: false,
			AllowedRegistryRoots: []string{
				"HKEY_LOCAL_MACHINE",
				"HKEY_CURRENT_USER",
				"HKEY_CLASSES_ROOT",
				"HKEY_USERS",
				"HKEY_CURRENT_CONFIG",
			},
			DenyRegistryPaths: []string{
				// Block security-sensitive keys
				`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts`,
				`SECURITY\Policy\Secrets`,
				`SAM\SAM\Domains\Account\Users`,
			},
			ReadOnly:     true, // Always read-only for compliance scanner
			AuditMode:    false,
			AuditLogPath: "output/audit",
		},
	}
}

// LoadConfig loads configuration from multiple sources with precedence:
// 1. CLI flags (highest priority)
// 2. Environment variables (prefixed with COMPLIANCE_TOOLKIT_)
// 3. YAML config file
// 4. Default values (lowest priority)
func LoadConfig(configPath string, flags *pflag.FlagSet) (*Config, error) {
	v := viper.New()

	// Set defaults
	defaults := DefaultConfig()
	setDefaults(v, defaults)

	// Configure Viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Search paths for config file
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	v.AddConfigPath(".")           // Current directory
	v.AddConfigPath("./config")    // ./config directory
	v.AddConfigPath("../config")   // ../config (when running from bin/)
	v.AddConfigPath("$HOME/.compliancetoolkit") // User home directory

	// Environment variables
	v.SetEnvPrefix("COMPLIANCE_TOOLKIT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional - don't error if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found - use defaults + env vars + flags
		slog.Debug("No config file found, using defaults")
	} else {
		slog.Info("Loaded config file", "path", v.ConfigFileUsed())
	}

	// Bind CLI flags (highest priority)
	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return nil, fmt.Errorf("error binding flags: %w", err)
		}
	}

	// Unmarshal into Config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults recursively sets default values in Viper from a Config struct
func setDefaults(v *viper.Viper, cfg *Config) {
	// Server defaults
	v.SetDefault("server.host", cfg.Server.Host)
	v.SetDefault("server.port", cfg.Server.Port)
	v.SetDefault("server.read_timeout", cfg.Server.ReadTimeout)
	v.SetDefault("server.max_concurrent_reads", cfg.Server.MaxConcurrentReads)
	v.SetDefault("server.graceful_shutdown_timeout", cfg.Server.GracefulShutdownTimeout)

	// Logging defaults
	v.SetDefault("logging.level", cfg.Logging.Level)
	v.SetDefault("logging.format", cfg.Logging.Format)
	v.SetDefault("logging.output_path", cfg.Logging.OutputPath)
	v.SetDefault("logging.enable_file_logging", cfg.Logging.EnableFileLogging)
	v.SetDefault("logging.max_log_file_size_mb", cfg.Logging.MaxLogFileSizeMB)
	v.SetDefault("logging.max_log_file_age_days", cfg.Logging.MaxLogFileAgeDays)

	// Reports defaults
	v.SetDefault("reports.config_path", cfg.Reports.ConfigPath)
	v.SetDefault("reports.output_path", cfg.Reports.OutputPath)
	v.SetDefault("reports.evidence_path", cfg.Reports.EvidencePath)
	v.SetDefault("reports.template_path", cfg.Reports.TemplatePath)
	v.SetDefault("reports.enable_evidence", cfg.Reports.EnableEvidence)
	v.SetDefault("reports.enable_dark_mode", cfg.Reports.EnableDarkMode)
	v.SetDefault("reports.parallel", cfg.Reports.Parallel)
	v.SetDefault("reports.max_parallel_reports", cfg.Reports.MaxParallelReports)

	// Security defaults
	v.SetDefault("security.require_admin_privileges", cfg.Security.RequireAdminPrivileges)
	v.SetDefault("security.allowed_registry_roots", cfg.Security.AllowedRegistryRoots)
	v.SetDefault("security.deny_registry_paths", cfg.Security.DenyRegistryPaths)
	v.SetDefault("security.read_only", cfg.Security.ReadOnly)
	v.SetDefault("security.audit_mode", cfg.Security.AuditMode)
	v.SetDefault("security.audit_log_path", cfg.Security.AuditLogPath)
}

// validateConfig performs validation on the loaded configuration
func validateConfig(cfg *Config) error {
	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(cfg.Logging.Level)] {
		return fmt.Errorf("invalid logging level: %s (must be debug, info, warn, or error)", cfg.Logging.Level)
	}

	// Validate logging format
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[strings.ToLower(cfg.Logging.Format)] {
		return fmt.Errorf("invalid logging format: %s (must be json or text)", cfg.Logging.Format)
	}

	// Validate server timeouts
	if cfg.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server.read_timeout must be positive (got %v)", cfg.Server.ReadTimeout)
	}
	if cfg.Server.GracefulShutdownTimeout <= 0 {
		return fmt.Errorf("server.graceful_shutdown_timeout must be positive (got %v)", cfg.Server.GracefulShutdownTimeout)
	}

	// Validate concurrent reads
	if cfg.Server.MaxConcurrentReads <= 0 {
		return fmt.Errorf("server.max_concurrent_reads must be positive (got %d)", cfg.Server.MaxConcurrentReads)
	}

	// Validate paths exist or can be created
	pathsToCheck := []struct {
		name string
		path string
	}{
		{"reports.config_path", cfg.Reports.ConfigPath},
		{"reports.output_path", cfg.Reports.OutputPath},
		{"reports.evidence_path", cfg.Reports.EvidencePath},
	}

	for _, p := range pathsToCheck {
		if p.path == "" {
			return fmt.Errorf("%s cannot be empty", p.name)
		}
		// Try to create directory if it doesn't exist
		if err := os.MkdirAll(p.path, 0755); err != nil {
			return fmt.Errorf("cannot create directory %s (%s): %w", p.name, p.path, err)
		}
	}

	// Validate security: ReadOnly must always be true
	if !cfg.Security.ReadOnly {
		return fmt.Errorf("security.read_only must be true (compliance scanner is read-only)")
	}

	// Validate allowed registry roots
	if len(cfg.Security.AllowedRegistryRoots) == 0 {
		return fmt.Errorf("security.allowed_registry_roots cannot be empty")
	}

	return nil
}

// SaveDefaultConfig creates a default config.yaml file at the specified path
func SaveDefaultConfig(configPath string) error {
	cfg := DefaultConfig()

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create Viper instance and set values
	v := viper.New()
	setDefaults(v, cfg)

	// Write config file
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	slog.Info("Created default config file", "path", configPath)
	return nil
}

// GetLogLevel converts string level to slog.Level
func (lc LoggingConfig) GetLogLevel() slog.Level {
	switch strings.ToLower(lc.Level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// IsJSONFormat returns true if logging format is JSON
func (lc LoggingConfig) IsJSONFormat() bool {
	return strings.ToLower(lc.Format) == "json"
}
