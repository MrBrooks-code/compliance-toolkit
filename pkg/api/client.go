package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a client for the Compliance Toolkit API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// ClientOption configures a Client
type ClientOption func(*Client)

// WithTimeout sets a custom HTTP timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithInsecureSkipVerify disables TLS certificate verification (for testing only!)
func WithInsecureSkipVerify() ClientOption {
	return func(c *Client) {
		transport := c.httpClient.Transport.(*http.Transport)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

// WithTLSConfig sets a custom TLS configuration
func WithTLSConfig(config *tls.Config) ClientOption {
	return func(c *Client) {
		transport := c.httpClient.Transport.(*http.Transport)
		transport.TLSClientConfig = config
	}
}

// NewClient creates a new API client
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	client := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Submit submits a compliance report to the server
func (c *Client) Submit(submission *ComplianceSubmission) (*SubmissionResponse, error) {
	// Validate before submitting
	if err := submission.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(submission)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submission: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/v1/compliance/submit", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var submissionResp SubmissionResponse
	if err := json.Unmarshal(body, &submissionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &submissionResp, nil
}

// Register registers a new client with the server
func (c *Client) Register(registration *ClientRegistration) error {
	jsonData, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("failed to marshal registration: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/clients/register", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetStatus retrieves the status of a submission
func (c *Client) GetStatus(submissionID string) (*SubmissionSummary, error) {
	url := fmt.Sprintf("%s/api/v1/compliance/status/%s", c.baseURL, submissionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(body))
	}

	var summary SubmissionSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &summary, nil
}

// Ping checks if the server is reachable
func (c *Client) Ping() error {
	url := fmt.Sprintf("%s/api/v1/health", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}
