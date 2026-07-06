package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"llama-admin/pkg/config"
)

func SetupRouter(h *Handler, cfg *config.AppConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Server.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(Logger)

	r.Get("/healthz", h.HealthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/version", h.VersionHandler)

		// Public auth routes (no session required)
		r.Get("/auth/providers", h.ListProviders)
		r.Route("/auth/{provider}/device", func(r chi.Router) {
			r.Post("/", h.InitiateDeviceFlow)
		})
		r.Route("/auth/{provider}/token", func(r chi.Router) {
			r.Post("/", h.ExchangeDeviceCode)
		})

		// Management auth middleware for remaining routes
		mgmtAuth := NewManagementAuthMiddleware(h.SessionStore, h.DB)
		r.With(mgmtAuth.Handler).Route("/auth", func(r chi.Router) {
			r.Get("/session", h.GetSession)
			r.Delete("/session", h.DeleteSession)
			r.Get("/allowed-emails", h.ListAllowedEmails)
			r.Post("/allowed-emails", h.AddAllowedEmail)
			r.Route("/allowed-emails/{email}", func(r chi.Router) {
				r.Delete("/", h.RemoveAllowedEmail)
			})
		})

		r.Route("/instances", func(r chi.Router) {
			r.Get("/", h.ListInstances)
			r.Post("/{name}", h.CreateInstance)
			r.Get("/{name}", h.GetInstance)
			r.Put("/{name}", h.UpdateInstance)
			r.Delete("/{name}", h.DeleteInstance)
			r.Post("/{name}/start", h.StartInstance)
			r.Post("/{name}/stop", h.StopInstance)
			r.Post("/{name}/restart", h.RestartInstance)
			r.Get("/{name}/logs", h.GetInstanceLogs)
		})

		r.Route("/auth/keys", func(r chi.Router) {
			r.Post("/", h.CreateAPIKey)
			r.Get("/", h.ListAPIKeys)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetAPIKey)
				r.Delete("/", h.DeleteAPIKey)
				r.Route("/permissions", func(r chi.Router) {
					r.Get("/", h.GetKeyPermissions)
					r.Post("/", h.GrantKeyPermission)
					r.Delete("/{iid}", h.RevokeKeyPermission)
				})
			})
		})

		r.Route("/models", func(r chi.Router) {
			r.Post("/download", h.CreateDownloadJob)
			r.Get("/download/jobs", h.ListDownloadJobs)
			r.Get("/download/jobs/{id}", h.GetDownloadJob)
			r.Delete("/download/jobs/{id}", h.CancelDownloadJob)
			r.Get("/", h.ListModels)
			r.Post("/", h.RegisterModel)
			r.Get("/files", h.ListModelFiles)
			r.Route("/{alias}", func(r chi.Router) {
				r.Get("/", h.GetModel)
				r.Delete("/", h.DeleteModel)
			})
		})
	})

	// Phase 3 - OpenAI-compatible proxy
	authMiddleware := NewInferenceAuthMiddleware(h.APIKeyStore, h.PermissionStore, true)
	r.Route("/v1", func(r chi.Router) {
		r.Use(authMiddleware.Handler)
		r.Get("/models", h.OpenAIListInstances)
		r.Post("/*", h.OpenAIProxy)
	})

	log.Println("router configured")
	return r
}
