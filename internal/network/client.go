package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client handles HTTP communication with external services
type Client struct {
	httpClient  *http.Client
	baseURL     string
	retryCount  int
	retryDelay  time.Duration
}

// ClientConfig holds configuration for the HTTP client
type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	RetryCount  int
	RetryDelay  time.Duration
}

// NewClient creates a new HTTP client instance
func NewClient(config ClientConfig) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL:    config.BaseURL,
		retryCount: config.RetryCount,
		retryDelay: config.RetryDelay,
	}
}

// formatUID formats a UID into the required URL format
func formatUID(uid string) string {
	return fmt.Sprintf("https://nfc.cursive.team/tap?uid=%s", uid)
}

// ValidateUIDs sends UIDs to the Cursive server for validation
func (c *Client) ValidateUIDs(ctx context.Context, uids []string) (*ValidationResponse, error) {
	// Format UIDs according to the specified format
	formattedUIDs := make([]string, len(uids))
	for i, uid := range uids {
		formattedUIDs[i] = formatUID(uid)
	}

	// Extract raw UIDs for the payload
	rawUIDs := make([]string, len(uids))
	for i, uid := range uids {
		rawUIDs[i] = uid
	}

	payload := ValidationRequest{
		UIDs: rawUIDs,
	}

	var response ValidationResponse
	err := c.doWithRetry(ctx, "POST", "/api/validate_uids", payload, &response)
	if err != nil {
		return nil, fmt.Errorf("validate UIDs request failed: %w", err)
	}

	return &response, nil
}

// doWithRetry performs an HTTP request with retry logic
func (c *Client) doWithRetry(ctx context.Context, method, path string, payload, response interface{}) error {
	var lastErr error
	
	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay):
			}
		}

		err := c.do(ctx, method, path, payload, response)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// do performs a single HTTP request
func (c *Client) do(ctx context.Context, method, path string, payload, response interface{}) error {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// ValidationRequest represents the payload for UID validation
type ValidationRequest struct {
	UIDs []string `json:"uids"`
}

// ValidationResponse represents the response from the Cursive server
type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Accounts []string `json:"accounts,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}
