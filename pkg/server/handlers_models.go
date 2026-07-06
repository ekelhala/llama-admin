package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateDownloadJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID   string `json:"repo_id"`
		Filename string `json:"filename"`
		Revision string `json:"revision"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RepoID == "" || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "repo_id and filename are required")
		return
	}

	job, err := h.ModelManager.StartDownload(req.RepoID, req.Filename, req.Revision)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"job_id": job.ID,
		"status": string(job.Status),
	})
}

func (h *Handler) ListDownloadJobs(w http.ResponseWriter, r *http.Request) {
	jobs := h.ModelManager.ListJobs()

	result := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, map[string]any{
			"job_id":   job.ID,
			"repo_id":  job.RepoID,
			"filename": job.Filename,
			"status":   string(job.Status),
			"progress": job.Progress,
			"error":    job.Error,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetDownloadJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	job, ok := h.ModelManager.GetJob(id)
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"job_id":   job.ID,
		"repo_id":  job.RepoID,
		"filename": job.Filename,
		"revision": job.Revision,
		"status":   string(job.Status),
		"progress": job.Progress,
		"error":    job.Error,
	})
}

func (h *Handler) CancelDownloadJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.ModelManager.CancelJob(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterModel registers a model alias -> filename mapping.
func (h *Handler) RegisterModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Alias    string `json:"alias"`
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Alias == "" || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "alias and filename are required")
		return
	}

	// Validate file exists and ends with .gguf
	if !strings.HasSuffix(req.Filename, ".gguf") {
		writeError(w, http.StatusBadRequest, "filename must end with .gguf")
		return
	}

	if _, err := os.Stat(req.Filename); err != nil {
		writeError(w, http.StatusBadRequest, "file not found: "+req.Filename)
		return
	}

	info, err := os.Stat(req.Filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	model, err := h.ModelManager.RegisterModel(req.Alias, req.Filename, info.Size())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         model.ID,
		"alias":      model.Alias,
		"filename":   model.Filename,
		"size_bytes": model.SizeBytes,
		"created_at": model.CreatedAt,
		"updated_at": model.UpdatedAt,
	})
}

// GetModel returns a model by alias.
func (h *Handler) GetModel(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")

	model, err := h.ModelManager.GetModel(alias)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         model.ID,
		"alias":      model.Alias,
		"filename":   model.Filename,
		"size_bytes": model.SizeBytes,
		"created_at": model.CreatedAt,
		"updated_at": model.UpdatedAt,
	})
}

// DeleteModel deletes a model by alias.
func (h *Handler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")

	if err := h.ModelManager.DeleteModel(alias); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListModels returns the DB-backed model catalog.
func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.ModelManager.ListModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]any, 0, len(models))
	for _, m := range models {
		result = append(result, map[string]any{
			"id":         m.ID,
			"alias":      m.Alias,
			"filename":   m.Filename,
			"size_bytes": m.SizeBytes,
			"created_at": m.CreatedAt,
			"updated_at": m.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// ListModelFiles returns raw disk-scanned GGUF files.
func (h *Handler) ListModelFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.ModelManager.ListFiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]any, 0, len(files))
	for _, f := range files {
		result = append(result, map[string]any{
			"filename":   f.Name,
			"path":       f.Path,
			"size_bytes": f.SizeBytes,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
