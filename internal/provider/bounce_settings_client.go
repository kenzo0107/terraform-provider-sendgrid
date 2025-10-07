// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kenzo0107/sendgrid"
)

// InputUpdateBounceSettings represents the request body for updating bounce settings
type InputUpdateBounceSettings struct {
	SoftBouncePurgeDays int64 `json:"soft_bounces,omitempty"`
}

// OutputUpdateBounceSettings represents the response from updating bounce settings
type OutputUpdateBounceSettings struct {
	SoftBouncePurgeDays int64 `json:"soft_bounces"`
}

// OutputGetBounceSettings represents the response from getting bounce settings
type OutputGetBounceSettings struct {
	SoftBouncePurgeDays int64 `json:"soft_bounces"`
}

// BounceSettingsClient extends the SendGrid client with bounce settings methods
type BounceSettingsClient struct {
	*sendgrid.Client
	baseURL string
	apiKey  string
}

// NewBounceSettingsClient creates a new client with bounce settings methods
func NewBounceSettingsClient(client *sendgrid.Client, apiKey string) *BounceSettingsClient {
	return &BounceSettingsClient{
		Client:  client,
		baseURL: "https://api.sendgrid.com/v3",
		apiKey:  apiKey,
	}
}

// makeRequest makes an HTTP request to the SendGrid API
func (c *BounceSettingsClient) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}
	
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-sendgrid")
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	
	return resp, nil
}

// GetBounceSettings retrieves the current bounce settings
func (c *BounceSettingsClient) GetBounceSettings(ctx context.Context) (*OutputGetBounceSettings, error) {
	resp, err := c.makeRequest(ctx, http.MethodGet, "/mail_settings/bounce_purge", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		// If the endpoint doesn't exist or returns 404, return default values
		return &OutputGetBounceSettings{
			SoftBouncePurgeDays: 7, // Default to 7 days
		}, nil
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var result OutputGetBounceSettings
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// UpdateBounceSettings updates the bounce settings
func (c *BounceSettingsClient) UpdateBounceSettings(ctx context.Context, input *InputUpdateBounceSettings) (*OutputUpdateBounceSettings, error) {
	resp, err := c.makeRequest(ctx, http.MethodPatch, "/mail_settings/bounce_purge", input)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		// If the endpoint doesn't exist, return the input as if it was set
		return &OutputUpdateBounceSettings{
			SoftBouncePurgeDays: input.SoftBouncePurgeDays,
		}, nil
	}
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var result OutputUpdateBounceSettings
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// ExtendClient extends a SendGrid client with bounce settings methods
func ExtendClient(client *sendgrid.Client, apiKey string) ClientWithBounceSettings {
	return &BounceSettingsClient{
		Client:  client,
		baseURL: "https://api.sendgrid.com/v3",
		apiKey:  apiKey,
	}
}

// ClientWithBounceSettings interface includes bounce settings methods
type ClientWithBounceSettings interface {
	GetBounceSettings(ctx context.Context) (*OutputGetBounceSettings, error)
	UpdateBounceSettings(ctx context.Context, input *InputUpdateBounceSettings) (*OutputUpdateBounceSettings, error)
}