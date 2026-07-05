package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"llama-admin/pkg/auth"
	"llama-admin/pkg/database"
	"llama-admin/pkg/instance"
)

const (
	apiKeyContextKey   contextKey = "api_key"
	userContextKey     contextKey = "user"
	sessionContextKey  contextKey = "session"
)

type APIKeyContext struct {
	ID             int64
	Name           string
	PermissionMode string
}

type ManagementAuthMiddleware struct {
	sessionStore *database.SessionStore
	db           *sql.DB
}

func NewManagementAuthMiddleware(sessionStore *database.SessionStore, db *sql.DB) *ManagementAuthMiddleware {
	return &ManagementAuthMiddleware{sessionStore: sessionStore, db: db}
}

func (m *ManagementAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OPTIONS - pass through for CORS preflight
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract session token
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeOpenAIError(w, http.StatusUnauthorized, "missing_session_token", "authentication_error")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		tokenHash := auth.HashSessionToken(token)

		// Get session
		session, err := m.sessionStore.GetByHash(tokenHash)
		if err != nil {
			writeOpenAIError(w, http.StatusUnauthorized, "invalid_session_token", "authentication_error")
			return
		}

		// Check expiration
		if session.ExpiresAt < time.Now().Unix() {
			m.sessionStore.Delete(session.ID)
			writeOpenAIError(w, http.StatusUnauthorized, "session_expired", "authentication_error")
			return
		}

		// Touch session asynchronously
		go func() {
			m.sessionStore.Touch(session.ID)
		}()

		// Get user
		user, err := database.GetUserByID(m.db, session.UserID)
		if err != nil {
			writeOpenAIError(w, http.StatusInternalServerError, "internal_error", "internal_error")
			return
		}

		// Add to context
		ctx := context.WithValue(r.Context(), userContextKey, &User{
			ID:             user.ID,
			Provider:       user.Provider,
			ProviderUserID: user.ProviderUserID,
			Username:       user.Username,
			Email:          user.Email,
			AvatarURL:      user.AvatarURL,
			CreatedAt:      user.CreatedAt,
			UpdatedAt:      user.UpdatedAt,
		})
		ctx = context.WithValue(ctx, sessionContextKey, &Session{
			ID:         session.ID,
			TokenHash:  session.TokenHash,
			UserID:     session.UserID,
			Provider:   session.Provider,
			ExpiresAt:  session.ExpiresAt,
			CreatedAt:  session.CreatedAt,
			LastUsedAt: session.LastUsedAt,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type InferenceAuthMiddleware struct {
	store                *database.APIKeyStore
	permissionStore      *database.PermissionStore
	requireInferenceAuth bool
}

func NewInferenceAuthMiddleware(apiKeyStore *database.APIKeyStore, permissionStore *database.PermissionStore, requireAuth bool) *InferenceAuthMiddleware {
	return &InferenceAuthMiddleware{
		store:                apiKeyStore,
		permissionStore:      permissionStore,
		requireInferenceAuth: requireAuth,
	}
}

func (m *InferenceAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.requireInferenceAuth {
			next.ServeHTTP(w, r)
			return
		}

		// OPTIONS - pass through for CORS preflight
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract API key
		key := extractAPIKey(r)
		if key == "" {
			writeOpenAIError(w, http.StatusUnauthorized, "missing_api_key", "authentication_error")
			return
		}

		// Get active keys and verify
		activeKeys, err := m.store.GetActiveKeys()
		if err != nil {
			writeOpenAIError(w, http.StatusInternalServerError, "internal_error", "internal_error")
			return
		}

		var matchedKey *database.APIKey
		for _, k := range activeKeys {
			if err := auth.VerifyKey(key, k.KeyHash); err == nil {
				matchedKey = &k
				break
			}
		}

		if matchedKey == nil {
			writeOpenAIError(w, http.StatusUnauthorized, "invalid_api_key", "authentication_error")
			return
		}

		// Touch key asynchronously
		go func() {
			m.store.TouchKey(matchedKey.ID)
		}()

		// Add to context
		ctx := context.WithValue(r.Context(), apiKeyContextKey, &APIKeyContext{
			ID:             matchedKey.ID,
			Name:           matchedKey.Name,
			PermissionMode: matchedKey.PermissionMode,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractAPIKey(r *http.Request) string {
	// Try Authorization: Bearer <key>
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try X-API-Key header
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}

	// Try query parameter
	if key := r.URL.Query().Get("api_key"); key != "" {
		return key
	}

	return ""
}

func (h *Handler) proxyToInstance(inst *instance.Instance, w http.ResponseWriter, r *http.Request) {
	inst.Proxy().ServeHTTP(w, r)
}

func logError(err error) {
	if err != nil {
		log.Printf("error: %v", err)
	}
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.Method, r.RequestURI, time.Since(start), w.Header().Get("Content-Type"))
	})
}
