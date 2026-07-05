package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"llama-admin/pkg/instance"
)

func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.Manager.ListInstances()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type instanceSummary struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		CreatedAt int64  `json:"created"`
	}

	result := make([]instanceSummary, 0, len(instances))
	for _, inst := range instances {
		result = append(result, instanceSummary{
			ID:        inst.ID,
			Name:      inst.Name,
			Status:    string(inst.Status()),
			CreatedAt: inst.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	inst, err := h.Manager.GetInstance(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.instanceToJSON(inst))
}

func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var opts instance.Options
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	inst, err := h.Manager.CreateInstance(name, &opts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, h.instanceToJSON(inst))
}

func (h *Handler) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var opts instance.Options
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	inst, err := h.Manager.UpdateInstance(name, &opts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.instanceToJSON(inst))
}

func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.Manager.DeleteInstance(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	inst, err := h.Manager.StartInstance(name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.instanceToJSON(inst))
}

func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	inst, err := h.Manager.StopInstance(name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.instanceToJSON(inst))
}

func (h *Handler) RestartInstance(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	inst, err := h.Manager.RestartInstance(name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.instanceToJSON(inst))
}

func (h *Handler) GetInstanceLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	lines := 200
	if l := r.URL.Query().Get("lines"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			lines = n
		}
	}

	logs, err := h.Manager.GetInstanceLogs(name, lines)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"logs": logs})
}

func (h *Handler) instanceToJSON(inst *instance.Instance) map[string]any {
	result := map[string]any{
		"id":        inst.ID,
		"name":      inst.Name,
		"status":    string(inst.Status()),
		"created":   inst.CreatedAt,
		"updated":   inst.UpdatedAt,
		"host":      inst.Host,
		"port":      inst.Port,
		"pid":       inst.PID,
		"options":   inst.Opts,
	}
	if inst.OwnerUserID != nil {
		result["owner_user_id"] = *inst.OwnerUserID
	}
	return result
}
