package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"llama-admin/pkg/auth"
)

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.ProviderRegistry.ListEnabled(h.Cfg.Auth.Providers)

	result := make([]map[string]any, 0, len(providers))
	for _, p := range providers {
		result = append(result, map[string]any{
			"name":        p.Name(),
			"device_flow": true,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"providers": result})
}

func (h *Handler) InitiateDeviceFlow(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := h.ProviderRegistry.Get(providerName)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("provider %q not found", providerName))
		return
	}

	dc, err := provider.InitiateDeviceFlow(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dc)
}

func (h *Handler) ExchangeDeviceCode(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := h.ProviderRegistry.Get(providerName)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("provider %q not found", providerName))
		return
	}

	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Poll for token
	var tokenResp *auth.TokenResponse
	pollInterval := 2
	timeout := 600 // 10 minutes
	started := time.Now()

	for time.Since(started).Seconds() < float64(timeout) {
		tr, err := provider.ExchangeDeviceCode(r.Context(), req.DeviceCode)
		if err != nil {
			if strings.Contains(err.Error(), "authorization_pending") {
				time.Sleep(time.Duration(pollInterval) * time.Second)
				continue
			}
			if strings.Contains(err.Error(), "slow_down") {
				pollInterval *= 2
				time.Sleep(time.Duration(pollInterval) * time.Second)
				continue
			}
			if strings.Contains(err.Error(), "expired_token") {
				writeOpenAIError(w, http.StatusBadRequest, "device_code_expired", "invalid_request_error")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tokenResp = tr
		break
	}

	if tokenResp == nil {
		writeOpenAIError(w, http.StatusBadRequest, "device_code_expired", "invalid_request_error")
		return
	}

	// Fetch user info
	userInfo, err := provider.FetchUserInfo(r.Context(), tokenResp.AccessToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check allowlist
	effectiveAllowlist := make(map[string]struct{})
	for _, e := range h.Cfg.Auth.AllowedEmails {
		effectiveAllowlist[e] = struct{}{}
	}
	// Add DB entries
	dbEmails, _ := h.AllowedEmailStore.List()
	for _, e := range dbEmails {
		effectiveAllowlist[e.Email] = struct{}{}
	}

	allowed := false
	for _, email := range userInfo.VerifiedEmails {
		if _, ok := effectiveAllowlist[email]; ok {
			allowed = true
			userInfo.Email = email
			break
		}
	}

	if !allowed {
		writeOpenAIError(w, http.StatusForbidden, "email_not_allowed", "forbidden")
		return
	}

	// Upsert user
	user, err := h.UserStore.Upsert(providerName, userInfo.ProviderUserID, userInfo.Username, userInfo.Email, userInfo.AvatarURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create session
	token, err := auth.GenerateSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tokenHash := auth.HashSessionToken(token)
	session, err := h.SessionStore.Create(user.ID, providerName, tokenHash, h.Cfg.Auth.Session.TTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_token": token,
		"expires_at":    session.ExpiresAt,
		"user": map[string]any{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"avatar_url":     user.AvatarURL,
			"provider":       user.Provider,
		},
	})
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeOpenAIError(w, http.StatusUnauthorized, "not_authenticated", "authentication_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"avatar_url": user.AvatarURL,
			"provider":   user.Provider,
		},
	})
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	session := SessionFromContext(r.Context())
	if session != nil {
		h.SessionStore.Delete(session.ID)
	}

	w.WriteHeader(http.StatusNoContent)
}
