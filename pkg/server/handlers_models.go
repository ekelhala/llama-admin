package server

import (
	"encoding/json"
	"net/http"

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
			"job_id":     job.ID,
			"repo_id":    job.RepoID,
			"filename":   job.Filename,
			"status":     string(job.Status),
			"progress":   job.Progress,
			"error":      job.Error,
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

func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.ModelManager.ListModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]any, 0, len(models))
	for _, m := range models {
		result = append(result, map[string]any{
			"name":        m.Name,
			"path":        m.Path,
			"size_bytes":  m.SizeBytes,
			"source":      m.Source,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
