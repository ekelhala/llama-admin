package server

import (
	"net/http"
)

func (h *Handler) VersionHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":    h.Cfg.Version,
		"commit":     h.Cfg.Commit,
		"build_time": h.Cfg.BuildTime,
	})
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
