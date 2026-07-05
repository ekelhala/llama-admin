package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListAllowedEmails(w http.ResponseWriter, r *http.Request) {
	dbEmails, err := h.AllowedEmailStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]any, 0)

	// Config-seeded emails
	for _, email := range h.Cfg.Auth.AllowedEmails {
		result = append(result, map[string]any{
			"email":  email,
			"source": "config",
		})
	}

	// DB emails
	for _, e := range dbEmails {
		result = append(result, map[string]any{
			"email":  e.Email,
			"source": "api",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"emails": result})
}

func (h *Handler) AddAllowedEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	user := UserFromContext(r.Context())
	userID := (*int64)(nil)
	if user != nil {
		userID = &user.ID
	}

	if err := h.AllowedEmailStore.Add(req.Email, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) RemoveAllowedEmail(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")
	// URL decode
	email, _ = url.PathUnescape(email)

	// Check if config-sourced
	for _, e := range h.Cfg.Auth.AllowedEmails {
		if strings.EqualFold(e, email) {
			writeError(w, http.StatusConflict, "cannot remove config-sourced email")
			return
		}
	}

	if err := h.AllowedEmailStore.Remove(email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
