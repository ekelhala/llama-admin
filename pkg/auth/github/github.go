package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"llama-admin/pkg/auth"
)

const (
	defaultDeviceEndpoint = "https://github.com/login/oauth/device/code"
	defaultTokenEndpoint  = "https://github.com/login/oauth/access_token"
	defaultUserEndpoint   = "https://api.github.com/user"
	defaultEmailsEndpoint = "https://api.github.com/user/emails"
)

type Provider struct {
	clientID                  string
	clientSecret              string
	scopes                    []string
	deviceEndpoint            string
	tokenEndpoint             string
	userEndpoint              string
	emailEndpoint             string
	httpClient                *http.Client
}

func New(clientID, clientSecret string, scopes []string) *Provider {
	p := &Provider{
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       scopes,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
	p.deviceEndpoint = defaultDeviceEndpoint
	p.tokenEndpoint = defaultTokenEndpoint
	p.userEndpoint = defaultUserEndpoint
	p.emailEndpoint = defaultEmailsEndpoint
	return p
}

func (p *Provider) Name() string { return "github" }

func (p *Provider) InitiateDeviceFlow(ctx context.Context) (*auth.DeviceCode, error) {
	values := url.Values{}
	values.Set("client_id", p.clientID)
	if len(p.scopes) > 0 {
		values.Set("scope", p.scopes[0])
		for _, s := range p.scopes[1:] {
			values.Set("scope", values.Get("scope")+" "+s)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.deviceEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device flow request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device flow failed (%d): %s", resp.StatusCode, string(body))
	}

	var dc auth.DeviceCode
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("parse device code: %w", err)
	}
	return &dc, nil
}

func (p *Provider) ExchangeDeviceCode(ctx context.Context, deviceCode string) (*auth.TokenResponse, error) {
	values := url.Values{}
	values.Set("client_id", p.clientID)
	values.Set("device_code", deviceCode)
	values.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, "POST", p.tokenEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Error           string `json:"error"`
		Description     string `json:"error_description"`
		AccessToken     string `json:"access_token"`
		TokenType       string `json:"token_type"`
		ExpiresIn       int    `json:"expires_in"`
		Scope           string `json:"scope"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if result.Error != "" {
		if result.Error == "authorization_pending" {
			return nil, fmt.Errorf("authorization_pending")
		}
		if result.Error == "slow_down" {
			return nil, fmt.Errorf("slow_down")
		}
		if result.Error == "expired_token" {
			return nil, fmt.Errorf("expired_token")
		}
		return nil, fmt.Errorf("token exchange failed: %s - %s", result.Error, result.Description)
	}

	return &auth.TokenResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		Scope:       result.Scope,
	}, nil
}

func (p *Provider) FetchUserInfo(ctx context.Context, accessToken string) (*auth.UserInfo, error) {
	// Fetch user info
	userReq, err := http.NewRequestWithContext(ctx, "GET", p.userEndpoint, nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+accessToken)
	userReq.Header.Set("Accept", "application/json")

	userResp, err := p.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	defer userResp.Body.Close()

	var user struct {
		ID       string `json:"id"`
		Login    string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	body, _ := io.ReadAll(userResp.Body)
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse user: %w", err)
	}

	// Fetch verified emails
	emailReq, err := http.NewRequestWithContext(ctx, "GET", p.emailEndpoint, nil)
	if err != nil {
		return nil, err
	}
	emailReq.Header.Set("Authorization", "Bearer "+accessToken)
	emailReq.Header.Set("Accept", "application/json")

	emailResp, err := p.httpClient.Do(emailReq)
	if err != nil {
		return nil, fmt.Errorf("fetch emails: %w", err)
	}
	defer emailResp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	body, _ = io.ReadAll(emailResp.Body)
	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, fmt.Errorf("parse emails: %w", err)
	}

	var verifiedEmails []string
	for _, e := range emails {
		if e.Verified {
			verifiedEmails = append(verifiedEmails, e.Email)
		}
	}

	return &auth.UserInfo{
		ProviderUserID: user.ID,
		Username:       user.Login,
		VerifiedEmails: verifiedEmails,
		AvatarURL:      user.AvatarURL,
	}, nil
}
