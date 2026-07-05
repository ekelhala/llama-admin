package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
	"llama-admin/pkg/auth"
)

type Provider struct {
	config oauth2.Config
}

func New(clientID, clientSecret string, scopes []string) *Provider {
	return &Provider{
		config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     githuboauth.Endpoint,
			Scopes:       scopes,
		},
	}
}

func (p *Provider) Name() string { return "github" }

func (p *Provider) InitiateDeviceFlow(ctx context.Context) (*auth.DeviceCode, error) {
	log.Printf("github.InitiateDeviceFlow: client_id=%q scopes=%v", p.config.ClientID, p.config.Scopes)
	da, err := p.config.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("device flow: %w", err)
	}
	expiresIn := 0
	if !da.Expiry.IsZero() {
		expiresIn = int(time.Until(da.Expiry).Seconds())
	}
	return &auth.DeviceCode{
		DeviceCode:          da.DeviceCode,
		UserCode:            da.UserCode,
		VerificationURI:     da.VerificationURI,
		VerificationURIHTML: da.VerificationURIComplete,
		ExpiresIn:           expiresIn,
		Interval:            int(da.Interval),
	}, nil
}

func (p *Provider) ExchangeDeviceCode(ctx context.Context, deviceCode string) (*auth.TokenResponse, error) {
	da := &oauth2.DeviceAuthResponse{DeviceCode: deviceCode}
	tok, err := p.config.DeviceAccessToken(ctx, da)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	expiresIn := 0
	if !tok.Expiry.IsZero() {
		expiresIn = int(time.Until(tok.Expiry).Seconds())
	}
	return &auth.TokenResponse{
		AccessToken: tok.AccessToken,
		TokenType:   tok.TokenType,
		ExpiresIn:   expiresIn,
	}, nil
}

func (p *Provider) FetchUserInfo(ctx context.Context, accessToken string) (*auth.UserInfo, error) {
	client := p.config.Client(ctx, &oauth2.Token{AccessToken: accessToken, TokenType: "Bearer"})

	userReq, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	defer userReq.Body.Close()

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := decodeResponse(userReq, &user); err != nil {
		return nil, fmt.Errorf("parse user: %w", err)
	}

	emailReq, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, fmt.Errorf("fetch emails: %w", err)
	}
	defer emailReq.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Verified bool   `json:"verified"`
	}
	if err := decodeResponse(emailReq, &emails); err != nil {
		return nil, fmt.Errorf("parse emails: %w", err)
	}

	var verifiedEmails []string
	for _, e := range emails {
		if e.Verified {
			verifiedEmails = append(verifiedEmails, e.Email)
		}
	}

	return &auth.UserInfo{
		ProviderUserID: fmt.Sprintf("%d", user.ID),
		Username:       user.Login,
		VerifiedEmails: verifiedEmails,
		AvatarURL:      user.AvatarURL,
	}, nil
}

func decodeResponse(resp *http.Response, v any) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
