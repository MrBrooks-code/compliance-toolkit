package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"
)

// TestErrorClassification tests the error classification logic
func TestErrorClassification(t *testing.T) {
	// Create a test client
	config := DefaultClientConfig()
	config.Retry.RetryOnServerError = true

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := &ComplianceClient{
		config: config,
		logger: logger,
	}

	tests := []struct {
		name          string
		err           error
		shouldRetry   bool
		description   string
	}{
		{
			name:        "nil error",
			err:         nil,
			shouldRetry: false,
			description: "nil errors should not retry",
		},
		{
			name:        "network connection refused",
			err:         errors.New("request failed: dial tcp: connection refused"),
			shouldRetry: true,
			description: "network errors should always retry",
		},
		{
			name:        "network timeout",
			err:         &net.DNSError{Err: "i/o timeout", IsTimeout: true},
			shouldRetry: true,
			description: "timeout errors should retry",
		},
		{
			name:        "400 bad request",
			err:         fmt.Errorf("server error (400): bad request"),
			shouldRetry: false,
			description: "4xx client errors should NOT retry",
		},
		{
			name:        "401 unauthorized",
			err:         fmt.Errorf("server error (401): unauthorized"),
			shouldRetry: false,
			description: "401 auth errors should NOT retry",
		},
		{
			name:        "404 not found",
			err:         fmt.Errorf("server error (404): not found"),
			shouldRetry: false,
			description: "404 errors should NOT retry",
		},
		{
			name:        "500 internal server error",
			err:         fmt.Errorf("server error (500): internal server error"),
			shouldRetry: true,
			description: "5xx server errors should retry when configured",
		},
		{
			name:        "503 service unavailable",
			err:         fmt.Errorf("server error (503): service unavailable"),
			shouldRetry: true,
			description: "503 errors should retry",
		},
		{
			name:        "unknown error",
			err:         errors.New("some random error"),
			shouldRetry: true,
			description: "unknown errors follow retry_on_server_error config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.shouldRetry(tt.err)
			if result != tt.shouldRetry {
				t.Errorf("shouldRetry() = %v, want %v for error: %v (%s)",
					result, tt.shouldRetry, tt.err, tt.description)
			}
		})
	}
}

// TestStatusCodeExtraction tests extracting status codes from error messages
func TestStatusCodeExtraction(t *testing.T) {
	tests := []struct {
		errStr       string
		expectedCode int
	}{
		{"server error (500): internal error", 500},
		{"server error (401): unauthorized", 401},
		{"registration failed (404): not found", 404},
		{"some error without code", 0},
		{"server error (abc): invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.errStr, func(t *testing.T) {
			code := extractStatusCode(tt.errStr)
			if code != tt.expectedCode {
				t.Errorf("extractStatusCode(%q) = %d, want %d",
					tt.errStr, code, tt.expectedCode)
			}
		})
	}
}

// TestBackoffJitter tests that backoff includes jitter
func TestBackoffJitter(t *testing.T) {
	config := DefaultClientConfig()
	config.Retry.InitialBackoff = 1 * time.Second
	config.Retry.BackoffMultiplier = 2.0
	config.Retry.MaxBackoff = 10 * time.Second

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := &ComplianceClient{
		config: config,
		logger: logger,
	}

	// Calculate backoff multiple times and ensure we get different values (due to jitter)
	backoffs := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		backoff := client.calculateBackoff(2) // Attempt 2
		backoffs[backoff] = true
	}

	// Should have at least 2 different backoff values due to jitter
	if len(backoffs) < 2 {
		t.Errorf("Expected jitter in backoff calculation, got same value %d times", len(backoffs))
	}

	// All backoffs should be reasonable (between 0.75x and 1.25x of base)
	baseBackoff := 2 * time.Second // 1s * 2.0
	for backoff := range backoffs {
		minBackoff := time.Duration(float64(baseBackoff) * 0.75)
		maxBackoff := time.Duration(float64(baseBackoff) * 1.25)
		if backoff < minBackoff || backoff > maxBackoff {
			t.Errorf("Backoff %v outside expected range [%v, %v]",
				backoff, minBackoff, maxBackoff)
		}
	}
}

// TestNetworkErrorDetection tests network error detection
func TestNetworkErrorDetection(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isNetwork bool
	}{
		{
			name:      "connection refused",
			err:       errors.New("dial tcp: connection refused"),
			isNetwork: true,
		},
		{
			name:      "dns error",
			err:       errors.New("no such host"),
			isNetwork: true,
		},
		{
			name:      "timeout",
			err:       &net.DNSError{Err: "timeout", IsTimeout: true},
			isNetwork: true,
		},
		{
			name:      "eof",
			err:       errors.New("EOF"),
			isNetwork: true,
		},
		{
			name:      "server error",
			err:       errors.New("server error (500): internal error"),
			isNetwork: false,
		},
		{
			name:      "nil error",
			err:       nil,
			isNetwork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.isNetwork {
				t.Errorf("isNetworkError(%v) = %v, want %v",
					tt.err, result, tt.isNetwork)
			}
		})
	}
}
