package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	innerclient "llama-admin/internal/client"
)

type DeviceFlowResult struct {
	SessionToken string
	ExpiresAt    int64
	User         map[string]any
}

func CompleteDeviceFlow(c *innerclient.Client, provider string) (*DeviceFlowResult, error) {
	// Get providers if no provider specified
	if provider == "" {
		data, err := c.Get("/api/v1/auth/providers")
		if err != nil {
			return nil, fmt.Errorf("get providers: %w", err)
		}
		var resp struct {
			Providers []map[string]any `json:"providers"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse providers: %w", err)
		}
		if len(resp.Providers) == 0 {
			return nil, fmt.Errorf("no providers available")
		}
		provider = resp.Providers[0]["name"].(string)
	}

	// Initiate device flow
	data, err := c.Post("/api/v1/auth/"+provider+"/device", nil)
	if err != nil {
		return nil, fmt.Errorf("initiate device flow: %w", err)
	}
	var dc struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parse device code: %w", err)
	}

	fmt.Printf("\nPlease visit:\n  %s\n\nAnd enter the code: %s\n\n", dc.VerificationURI, dc.UserCode)

	// Poll for token
	started := time.Now()
	timeout := time.Duration(dc.ExpiresIn) * time.Second
	interval := time.Duration(dc.Interval) * time.Second
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}

	for time.Since(started) < timeout {
		time.Sleep(interval)

		resp, err := http.Post(c.BaseURL+"/api/v1/auth/"+provider+"/token", "application/json",
			strings.NewReader(`{"device_code":"`+dc.DeviceCode+`"}`))
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			SessionToken string                 `json:"session_token"`
			ExpiresAt    int64                  `json:"expires_at"`
			User         map[string]any         `json:"user"`
			Error        string                 `json:"error"`
			ErrorDesc    string                 `json:"error_description"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		if result.Error == "authorization_pending" || result.Error == "slow_down" {
			continue
		}
		if result.Error == "expired_token" {
			return nil, fmt.Errorf("device flow expired")
		}
		if result.Error != "" {
			return nil, fmt.Errorf("device flow failed: %s", result.ErrorDesc)
		}

		return &DeviceFlowResult{
			SessionToken: result.SessionToken,
			ExpiresAt:    result.ExpiresAt,
			User:         result.User,
		}, nil
	}

	return nil, fmt.Errorf("device flow timed out")
}
