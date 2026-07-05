package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"llama-admin/pkg/auth"
	"llama-admin/pkg/config"
	"llama-admin/pkg/database"
	"llama-admin/pkg/manager"
	"llama-admin/pkg/models"
)

type contextKey string

const (
	configKey contextKey = "config"
)

type Handler struct {
	Cfg               *config.AppConfig
	DB                *sql.DB
	Manager           manager.InstanceManager
	APIKeyStore       *database.APIKeyStore
	PermissionStore   *database.PermissionStore
	UserStore         *database.UserStore
	SessionStore      *database.SessionStore
	AllowedEmailStore *database.AllowedEmailStore
	ProviderRegistry  *auth.ProviderRegistry
	ModelManager      *models.Manager
}

func NewHandler(cfg *config.AppConfig, db *sql.DB, mgr manager.InstanceManager, registry *auth.ProviderRegistry, modelMgr *models.Manager) *Handler {
	return &Handler{
		Cfg:               cfg,
		DB:                db,
		Manager:           mgr,
		APIKeyStore:       database.NewAPIKeyStore(db),
		PermissionStore:   database.NewPermissionStore(db),
		UserStore:         database.NewUserStore(db),
		SessionStore:      database.NewSessionStore(db),
		AllowedEmailStore: database.NewAllowedEmailStore(db),
		ProviderRegistry:  registry,
		ModelManager:      modelMgr,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type User struct {
	ID             int64
	Provider       string
	ProviderUserID string
	Username       string
	Email          string
	AvatarURL      string
	CreatedAt      int64
	UpdatedAt      int64
}

func UserFromContext(ctx context.Context) *User {
	v := ctx.Value(userContextKey)
	if v == nil {
		return nil
	}
	return v.(*User)
}

type Session struct {
	ID        int64
	TokenHash string
	UserID    int64
	Provider  string
	ExpiresAt int64
	CreatedAt int64
	LastUsedAt *int64
}

func SessionFromContext(ctx context.Context) *Session {
	v := ctx.Value(sessionContextKey)
	if v == nil {
		return nil
	}
	return v.(*Session)
}
