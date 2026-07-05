package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string `json:"name"`
		PermissionMode string `json:"permission_mode"`
		ExpiresAt      *int64 `json:"expires_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	key, err := h.APIKeyStore.Create(req.Name, nil, req.PermissionMode, req.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":             key.ID,
		"key":            key.Key,
		"name":           key.Name,
		"permission_mode": key.PermissionMode,
		"created_at":     key.CreatedAt,
	})
}

func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.APIKeyStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		result = append(result, map[string]any{
			"id":              k.ID,
			"name":            k.Name,
			"permission_mode": k.PermissionMode,
			"expires_at":      k.ExpiresAt,
			"created_at":      k.CreatedAt,
			"last_used_at":    k.LastUsedAt,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	key, err := h.APIKeyStore.Get(int64(id))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              key.ID,
		"name":            key.Name,
		"permission_mode": key.PermissionMode,
		"expires_at":      key.ExpiresAt,
		"created_at":      key.CreatedAt,
		"last_used_at":    key.LastUsedAt,
	})
}

func (h *Handler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	if err := h.APIKeyStore.Delete(int64(id)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetKeyPermissions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	perms, err := h.PermissionStore.List(int64(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		result = append(result, map[string]any{
			"instance_id":    p.InstanceID,
			"instance_name":  p.InstanceName,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GrantKeyPermission(w http.ResponseWriter, r *http.Request) {
	keyID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	var req struct {
		InstanceID int64 `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.PermissionStore.Grant(int64(keyID), req.InstanceID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) RevokeKeyPermission(w http.ResponseWriter, r *http.Request) {
	keyID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	instanceID, err := strconv.Atoi(chi.URLParam(r, "iid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}

	if err := h.PermissionStore.Revoke(int64(keyID), int64(instanceID)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
