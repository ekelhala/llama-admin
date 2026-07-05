package auth

import (
	"context"

	"llama-admin/pkg/config"
)

type Provider interface {
	Name() string
	InitiateDeviceFlow(ctx context.Context) (*DeviceCode, error)
	ExchangeDeviceCode(ctx context.Context, deviceCode string) (*TokenResponse, error)
	FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

type DeviceCode struct {
	DeviceCode           string    `json:"device_code"`
	UserCode             string    `json:"user_code"`
	VerificationURI      string    `json:"verification_uri"`
	VerificationURIHTML  string    `json:"verification_uri_html,omitempty"`
	ExpiresIn            int       `json:"expires_in"`
	Interval             int       `json:"interval"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
}

type UserInfo struct {
	ProviderUserID string   `json:"provider_user_id"`
	Username       string   `json:"username"`
	Email          string   `json:"email"`
	VerifiedEmails []string `json:"verified_emails"`
	AvatarURL      string   `json:"avatar_url"`
}

type ProviderRegistry struct {
	providers map[string]Provider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

func (r *ProviderRegistry) Register(p Provider) {
	r.providers[p.Name()] = p
}

func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

func (r *ProviderRegistry) ListEnabled(cfgProviders map[string]config.ProviderConfig) []Provider {
	var result []Provider
	for name, p := range r.providers {
		if cfg, ok := cfgProviders[name]; ok && cfg.Enabled {
			result = append(result, p)
		}
	}
	return result
}

func (r *ProviderRegistry) AllEnabled(cfgProviders map[string]config.ProviderConfig) map[string]Provider {
	result := make(map[string]Provider)
	for name, p := range r.providers {
		if cfg, ok := cfgProviders[name]; ok && cfg.Enabled {
			result[name] = p
		}
	}
	return result
}
