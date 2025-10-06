package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"compliancetoolkit/pkg/api"
)

// ComplianceClient is the main client application
type ComplianceClient struct {
	config *ClientConfig
	logger *slog.Logger
	runner *ReportRunner
	cache  *SubmissionCache
	api    *api.Client
}

// NewComplianceClient creates a new compliance client
func NewComplianceClient(config *ClientConfig, logger *slog.Logger) *ComplianceClient {
	client := &ComplianceClient{
		config: config,
		logger: logger,
	}

	// Create report runner
	client.runner = NewReportRunner(config, logger)

	// Create cache if enabled
	if config.Cache.Enabled {
		cache, err := NewSubmissionCache(config.Cache.Path, config.Cache.MaxSizeMB, config.Cache.MaxAge)
		if err != nil {
			logger.Warn("Failed to create submission cache", "error", err)
		} else {
			client.cache = cache
			if config.Cache.AutoClean {
				if err := cache.Clean(); err != nil {
					logger.Warn("Failed to clean cache", "error", err)
				}
			}
		}
	}

	// Create API client if in server mode
	if config.IsServerMode() {
		opts := []api.ClientOption{
			api.WithTimeout(config.Server.Timeout),
		}
		if !config.Server.TLSVerify {
			opts = append(opts, api.WithInsecureSkipVerify())
		}
		client.api = api.NewClient(config.Server.URL, config.Server.APIKey, opts...)
	}

	return client
}

// Run executes the client based on configuration
func (c *ComplianceClient) Run() error {
	// Check if scheduling is enabled
	if c.config.Schedule.Enabled {
		return c.runScheduled()
	}

	// Run once
	return c.runOnce()
}

// runOnce executes reports once and exits
func (c *ComplianceClient) runOnce() error {
	c.logger.Info("Running in once mode")

	// Retry cached submissions first if configured
	if c.config.Server.RetryOnStartup && c.cache != nil && c.api != nil {
		if err := c.retryCachedSubmissions(); err != nil {
			c.logger.Warn("Failed to retry cached submissions", "error", err)
		}
	}

	// Execute all configured reports
	for _, reportName := range c.config.Reports.Reports {
		if err := c.executeReport(reportName); err != nil {
			c.logger.Error("Report execution failed",
				"report", reportName,
				"error", err,
			)
			return err
		}
	}

	return nil
}

// runScheduled runs reports on a schedule
func (c *ComplianceClient) runScheduled() error {
	c.logger.Info("Running in scheduled mode", "cron", c.config.Schedule.Cron)

	// Create cron scheduler with default logger
	scheduler := cron.New()

	// Add scheduled job
	_, err := scheduler.AddFunc(c.config.Schedule.Cron, func() {
		c.logger.Info("Scheduled execution triggered")

		// Execute all configured reports
		for _, reportName := range c.config.Reports.Reports {
			if err := c.executeReport(reportName); err != nil {
				c.logger.Error("Scheduled report execution failed",
					"report", reportName,
					"error", err,
				)
				// Continue with next report even if one fails
			}
		}
	})

	if err != nil {
		return fmt.Errorf("failed to add scheduled job: %w", err)
	}

	// Start scheduler
	scheduler.Start()
	c.logger.Info("Scheduler started successfully", "cron", c.config.Schedule.Cron)
	defer scheduler.Stop()

	// Run once immediately on startup if configured
	if c.config.Server.RetryOnStartup && c.cache != nil && c.api != nil {
		c.logger.Info("Running initial execution on startup")
		if err := c.retryCachedSubmissions(); err != nil {
			c.logger.Warn("Failed to retry cached submissions", "error", err)
		}
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigChan
	c.logger.Info("Received shutdown signal, stopping scheduler", "signal", sig.String())

	// Scheduler will be stopped by defer
	c.logger.Info("Scheduler stopped, exiting gracefully")
	return nil
}

// executeReport executes a single report
func (c *ComplianceClient) executeReport(reportName string) error {
	startTime := time.Now()

	c.logger.Info("Executing report", "report", reportName)

	// Run the report
	submission, err := c.runner.Run(reportName)
	if err != nil {
		return fmt.Errorf("report execution failed: %w", err)
	}

	duration := time.Since(startTime)
	c.logger.Info("Report completed",
		"report", reportName,
		"duration", duration,
		"status", submission.Compliance.OverallStatus,
		"passed", submission.Compliance.PassedChecks,
		"failed", submission.Compliance.FailedChecks,
	)

	// Submit to server if configured
	if c.api != nil {
		if err := c.submitToServer(submission); err != nil {
			c.logger.Error("Failed to submit to server", "error", err)

			// Cache the submission for later retry
			if c.cache != nil {
				if err := c.cache.Store(submission); err != nil {
					c.logger.Error("Failed to cache submission", "error", err)
				} else {
					c.logger.Info("Submission cached for later retry")
				}
			}

			// Don't return error - local report was generated successfully
			c.logger.Warn("Server submission failed but local report saved")
		}
	}

	return nil
}

// submitToServer submits a compliance report to the server
func (c *ComplianceClient) submitToServer(submission *api.ComplianceSubmission) error {
	startTime := time.Now()
	c.logger.Info("Submitting to server", "submission_id", submission.SubmissionID)

	// Submit with retry logic
	var lastErr error
	totalBackoff := time.Duration(0)

	for attempt := 0; attempt <= c.config.Retry.MaxAttempts; attempt++ {
		attemptStart := time.Now()

		if attempt > 0 {
			backoff := c.calculateBackoff(attempt)
			totalBackoff += backoff
			c.logger.Info("Retrying submission",
				"attempt", attempt,
				"max_attempts", c.config.Retry.MaxAttempts+1,
				"backoff", backoff,
				"total_backoff", totalBackoff,
			)
			time.Sleep(backoff)
		}

		resp, err := c.api.Submit(submission)
		attemptDuration := time.Since(attemptStart)

		if err == nil {
			totalDuration := time.Since(startTime)
			c.logger.Info("Submission accepted",
				"submission_id", resp.SubmissionID,
				"status", resp.Status,
				"attempts", attempt+1,
				"total_duration", totalDuration,
				"total_backoff", totalBackoff,
			)
			return nil
		}

		lastErr = err
		c.logger.Warn("Submission attempt failed",
			"attempt", attempt+1,
			"max_attempts", c.config.Retry.MaxAttempts+1,
			"duration", attemptDuration,
			"error", err,
		)

		// Check if we should retry
		if !c.shouldRetry(err) {
			totalDuration := time.Since(startTime)
			c.logger.Error("Submission failed with non-retryable error",
				"attempts", attempt+1,
				"total_duration", totalDuration,
				"error", err,
			)
			return fmt.Errorf("submission failed (non-retryable): %w", err)
		}
	}

	totalDuration := time.Since(startTime)
	c.logger.Error("Submission failed after all retry attempts",
		"attempts", c.config.Retry.MaxAttempts+1,
		"total_duration", totalDuration,
		"total_backoff", totalBackoff,
		"error", lastErr,
	)
	return fmt.Errorf("submission failed after %d attempts: %w", c.config.Retry.MaxAttempts+1, lastErr)
}

// retryCachedSubmissions attempts to submit all cached submissions
func (c *ComplianceClient) retryCachedSubmissions() error {
	if c.cache == nil {
		return nil
	}

	submissions, err := c.cache.List()
	if err != nil {
		return fmt.Errorf("failed to list cached submissions: %w", err)
	}

	if len(submissions) == 0 {
		c.logger.Debug("No cached submissions to retry")
		return nil
	}

	c.logger.Info("Retrying cached submissions", "count", len(submissions))

	for _, sub := range submissions {
		c.logger.Info("Retrying cached submission", "submission_id", sub.SubmissionID)

		if err := c.submitToServer(sub); err != nil {
			c.logger.Warn("Failed to submit cached submission",
				"submission_id", sub.SubmissionID,
				"error", err,
			)
			continue
		}

		// Remove from cache on success
		if err := c.cache.Remove(sub.SubmissionID); err != nil {
			c.logger.Warn("Failed to remove from cache",
				"submission_id", sub.SubmissionID,
				"error", err,
			)
		}
	}

	return nil
}

// calculateBackoff calculates exponential backoff duration with jitter
func (c *ComplianceClient) calculateBackoff(attempt int) time.Duration {
	backoff := c.config.Retry.InitialBackoff
	for i := 1; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * c.config.Retry.BackoffMultiplier)
		if backoff > c.config.Retry.MaxBackoff {
			backoff = c.config.Retry.MaxBackoff
			break
		}
	}

	// Add jitter (±25% randomness) to prevent thundering herd
	jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
	backoff = backoff - (backoff / 4) + jitter

	return backoff
}

// shouldRetry determines if an error is retryable based on error type
func (c *ComplianceClient) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are always retryable (connection refused, timeout, DNS, etc.)
	if isNetworkError(err) {
		c.logger.Debug("Network error detected, retrying", "error", errStr)
		return true
	}

	// Parse HTTP status code from error message
	statusCode := extractStatusCode(errStr)

	// Client errors (4xx) are NOT retryable (bad request, auth failure, etc.)
	if statusCode >= 400 && statusCode < 500 {
		c.logger.Warn("Client error detected, NOT retrying",
			"status_code", statusCode,
			"error", errStr,
		)
		return false
	}

	// Server errors (5xx) are retryable if configured
	if statusCode >= 500 && statusCode < 600 {
		c.logger.Debug("Server error detected, retrying",
			"status_code", statusCode,
			"error", errStr,
		)
		return c.config.Retry.RetryOnServerError
	}

	// Unknown errors - retry if configured
	c.logger.Debug("Unknown error type, using retry_on_server_error config",
		"error", errStr,
	)
	return c.config.Retry.RetryOnServerError
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error interface (includes timeout, temporary errors)
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	errStr := strings.ToLower(err.Error())

	// Common network error patterns
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"tls handshake timeout",
		"request failed",
		"dial tcp",
		"eof",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// extractStatusCode extracts HTTP status code from error message
func extractStatusCode(errStr string) int {
	// Error format from api client: "server error (500): message"
	// or "registration failed (401): message"

	// Find the pattern "(NNN)" where NNN is 3 digits
	start := strings.Index(errStr, "(")
	if start == -1 {
		return 0
	}

	end := strings.Index(errStr[start:], ")")
	if end == -1 {
		return 0
	}

	// Extract the substring between parentheses
	codeStr := errStr[start+1 : start+end]

	// Try to parse as integer
	var code int
	if _, err := fmt.Sscanf(codeStr, "%d", &code); err == nil {
		return code
	}

	return 0 // Unknown status
}
