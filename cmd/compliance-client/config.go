package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// ClientConfig represents the complete client configuration
type ClientConfig struct {
	Client   ClientSettings   `mapstructure:"client"`
	Server   ServerSettings   `mapstructure:"server"`
	Reports  ReportSettings   `mapstructure:"reports"`
	Schedule ScheduleSettings `mapstructure:"schedule"`
	Retry    RetrySettings    `mapstructure:"retry"`
	Cache    CacheSettings    `mapstructure:"cache"`
	Logging  LoggingSettings  `mapstructure:"logging"`
}

// ClientSettings contains client identification and behavior
type ClientSettings struct {
	ID       string `mapstructure:"id"`        // Unique client ID (auto-generated if empty)
	Hostname string `mapstructure:"hostname"`  // Override hostname (auto-detected if empty)
	Enabled  bool   `mapstructure:"enabled"`   // Master enable/disable switch
}

// ServerSettings contains server connection configuration
type ServerSettings struct {
	URL            string        `mapstructure:"url"`              // Server URL (empty = standalone mode)
	APIKey         string        `mapstructure:"api_key"`          // API key for authentication
	TLSVerify      bool          `mapstructure:"tls_verify"`       // Verify TLS certificates
	Timeout        time.Duration `mapstructure:"timeout"`          // Request timeout
	RetryOnStartup bool          `mapstructure:"retry_on_startup"` // Retry cached submissions on startup
}

// ReportSettings contains report execution configuration
type ReportSettings struct {
	ConfigPath string   `mapstructure:"config_path"` // Path to report configs
	OutputPath string   `mapstructure:"output_path"` // Local output directory
	Reports    []string `mapstructure:"reports"`     // List of reports to run
	SaveLocal  bool     `mapstructure:"save_local"`  // Save HTML reports locally
}

// ScheduleSettings contains scheduling configuration
type ScheduleSettings struct {
	Enabled bool   `mapstructure:"enabled"` // Enable scheduled execution
	Cron    string `mapstructure:"cron"`    // Cron expression (e.g., "0 2 * * *")
}

// RetrySettings contains retry logic configuration
type RetrySettings struct {
	MaxAttempts        int           `mapstructure:"max_attempts"`        // Maximum retry attempts
	InitialBackoff     time.Duration `mapstructure:"initial_backoff"`     // Initial backoff duration
	MaxBackoff         time.Duration `mapstructure:"max_backoff"`         // Maximum backoff duration
	BackoffMultiplier  float64       `mapstructure:"backoff_multiplier"`  // Backoff multiplier
	RetryOnServerError bool          `mapstructure:"retry_on_server_error"` // Retry on 5xx errors
}

// CacheSettings contains local cache configuration
type CacheSettings struct {
	Enabled    bool          `mapstructure:"enabled"`      // Enable local caching
	Path       string        `mapstructure:"path"`         // Cache directory path
	MaxSizeMB  int           `mapstructure:"max_size_mb"`  // Maximum cache size in MB
	MaxAge     time.Duration `mapstructure:"max_age"`      // Maximum age for cached items
	AutoClean  bool          `mapstructure:"auto_clean"`   // Automatically clean old cache
}

// LoggingSettings contains logging configuration
type LoggingSettings struct {
	Level      string `mapstructure:"level"`       // Log level: debug, info, warn, error
	Format     string `mapstructure:"format"`      // Log format: json, text
	OutputPath string `mapstructure:"output_path"` // Log file path (or stdout/stderr)
}

// DefaultClientConfig returns a ClientConfig with sensible defaults
func DefaultClientConfig() *ClientConfig {
	hostname, _ := os.Hostname()

	return &ClientConfig{
		Client: ClientSettings{
			ID:       "",    // Auto-generated
			Hostname: hostname,
			Enabled:  true,
		},
		Server: ServerSettings{
			URL:            "",    // Standalone mode by default
			APIKey:         "",
			TLSVerify:      true,
			Timeout:        30 * time.Second,
			RetryOnStartup: true,
		},
		Reports: ReportSettings{
			ConfigPath: "configs/reports",
			OutputPath: "output/reports",
			Reports: []string{
				"NIST_800_171_compliance.json",
			},
			SaveLocal: true,
		},
		Schedule: ScheduleSettings{
			Enabled: false,
			Cron:    "0 2 * * *", // Daily at 2 AM
		},
		Retry: RetrySettings{
			MaxAttempts:        3,
			InitialBackoff:     30 * time.Second,
			MaxBackoff:         5 * time.Minute,
			BackoffMultiplier:  2.0,
			RetryOnServerError: true,
		},
		Cache: CacheSettings{
			Enabled:   true,
			Path:      "cache/submissions",
			MaxSizeMB: 100,
			MaxAge:    7 * 24 * time.Hour, // 7 days
			AutoClean: true,
		},
		Logging: LoggingSettings{
			Level:      "info",
			Format:     "text",
			OutputPath: "stdout",
		},
	}
}

// LoadClientConfig loads configuration from file, environment, and flags
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	v := viper.New()

	// Set defaults
	defaults := DefaultClientConfig()
	setClientDefaults(v, defaults)

	// Configure Viper
	if configPath != "" {
		// Explicit config file provided
		v.SetConfigFile(configPath)
	} else {
		// Search for default config
		v.SetConfigName("client")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.compliancetoolkit")
		v.AddConfigPath("C:/ProgramData/ComplianceToolkit")
	}

	// Environment variables
	v.SetEnvPrefix("COMPLIANCE_CLIENT")
	v.AutomaticEnv()

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found - use defaults
	}

	// Unmarshal into config struct
	var config ClientConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Post-process config
	if err := processConfig(&config); err != nil {
		return nil, fmt.Errorf("error processing config: %w", err)
	}

	return &config, nil
}

// setClientDefaults sets default values in Viper
func setClientDefaults(v *viper.Viper, cfg *ClientConfig) {
	// Client
	v.SetDefault("client.id", cfg.Client.ID)
	v.SetDefault("client.hostname", cfg.Client.Hostname)
	v.SetDefault("client.enabled", cfg.Client.Enabled)

	// Server
	v.SetDefault("server.url", cfg.Server.URL)
	v.SetDefault("server.api_key", cfg.Server.APIKey)
	v.SetDefault("server.tls_verify", cfg.Server.TLSVerify)
	v.SetDefault("server.timeout", cfg.Server.Timeout)
	v.SetDefault("server.retry_on_startup", cfg.Server.RetryOnStartup)

	// Reports
	v.SetDefault("reports.config_path", cfg.Reports.ConfigPath)
	v.SetDefault("reports.output_path", cfg.Reports.OutputPath)
	v.SetDefault("reports.reports", cfg.Reports.Reports)
	v.SetDefault("reports.save_local", cfg.Reports.SaveLocal)

	// Schedule
	v.SetDefault("schedule.enabled", cfg.Schedule.Enabled)
	v.SetDefault("schedule.cron", cfg.Schedule.Cron)

	// Retry
	v.SetDefault("retry.max_attempts", cfg.Retry.MaxAttempts)
	v.SetDefault("retry.initial_backoff", cfg.Retry.InitialBackoff)
	v.SetDefault("retry.max_backoff", cfg.Retry.MaxBackoff)
	v.SetDefault("retry.backoff_multiplier", cfg.Retry.BackoffMultiplier)
	v.SetDefault("retry.retry_on_server_error", cfg.Retry.RetryOnServerError)

	// Cache
	v.SetDefault("cache.enabled", cfg.Cache.Enabled)
	v.SetDefault("cache.path", cfg.Cache.Path)
	v.SetDefault("cache.max_size_mb", cfg.Cache.MaxSizeMB)
	v.SetDefault("cache.max_age", cfg.Cache.MaxAge)
	v.SetDefault("cache.auto_clean", cfg.Cache.AutoClean)

	// Logging
	v.SetDefault("logging.level", cfg.Logging.Level)
	v.SetDefault("logging.format", cfg.Logging.Format)
	v.SetDefault("logging.output_path", cfg.Logging.OutputPath)
}

// processConfig performs post-processing on the loaded config
func processConfig(cfg *ClientConfig) error {
	// Ensure hostname is set first
	if cfg.Client.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}
		cfg.Client.Hostname = hostname
	}

	// Generate client ID if not set (after hostname is determined)
	if cfg.Client.ID == "" {
		cfg.Client.ID = generateClientID(cfg.Client.Hostname)
	}

	// Validate paths exist or can be created
	paths := []string{
		cfg.Reports.ConfigPath,
		cfg.Reports.OutputPath,
		cfg.Cache.Path,
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	return nil
}

// generateClientID generates a unique client ID based on hostname
func generateClientID(hostname string) string {
	// Simple ID: hostname-based
	// In production, might want to use machine GUID or similar
	return fmt.Sprintf("client-%s", hostname)
}

// IsStandaloneMode returns true if client is running in standalone mode (no server)
func (c *ClientConfig) IsStandaloneMode() bool {
	return c.Server.URL == ""
}

// IsServerMode returns true if client is configured to submit to a server
func (c *ClientConfig) IsServerMode() bool {
	return c.Server.URL != ""
}

// Validate validates the configuration
func (c *ClientConfig) Validate() error {
	if !c.Client.Enabled {
		return fmt.Errorf("client is disabled in configuration")
	}

	if c.Client.Hostname == "" {
		return fmt.Errorf("client hostname is required")
	}

	if len(c.Reports.Reports) == 0 {
		return fmt.Errorf("at least one report must be configured")
	}

	// If server mode, validate server config
	if c.IsServerMode() {
		if c.Server.APIKey == "" {
			return fmt.Errorf("server.api_key is required when server.url is set")
		}
		if c.Server.Timeout <= 0 {
			return fmt.Errorf("server.timeout must be positive")
		}
	}

	// Validate retry settings
	if c.Retry.MaxAttempts < 0 {
		return fmt.Errorf("retry.max_attempts must be >= 0")
	}
	if c.Retry.BackoffMultiplier < 1.0 {
		return fmt.Errorf("retry.backoff_multiplier must be >= 1.0")
	}

	// Validate cache settings
	if c.Cache.Enabled {
		if c.Cache.MaxSizeMB <= 0 {
			return fmt.Errorf("cache.max_size_mb must be positive")
		}
		if c.Cache.MaxAge <= 0 {
			return fmt.Errorf("cache.max_age must be positive")
		}
	}

	return nil
}
