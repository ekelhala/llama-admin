package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llama-admin/pkg/auth"
	"llama-admin/pkg/auth/github"
	"llama-admin/pkg/config"
	"llama-admin/pkg/database"
	"llama-admin/pkg/manager"
	"llama-admin/pkg/models"
	"llama-admin/pkg/server"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	flag.Parse()

	if len(flag.Args()) > 0 && flag.Arg(0) == "--version" {
		fmt.Printf("llama-admin %s (commit: %s, built: %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	if err := config.LoadDotenv(); err != nil {
		log.Printf("warning: failed to load .env: %v", err)
	}

	cfgPath := config.ConfigPath()
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	cfg.Version = version
	cfg.Commit = commit
	cfg.BuildTime = buildTime

	if err := os.MkdirAll(config.DataDirPath(), 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(cfg.Database.Path); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Setup model manager (constructed before the instance manager so the
	// latter can resolve model references against the on-disk catalog at
	// start time).
	modelStore := database.NewModelStore(db)
	modelMgr := models.NewManager(cfg, version, modelStore)
	defer modelMgr.Close()

	mgr := manager.New(cfg, db, modelMgr)

	// Setup OAuth providers
	registry := auth.NewProviderRegistry()
	for name, pcfg := range cfg.Auth.Providers {
		if !pcfg.Enabled {
			continue
		}
		switch name {
		case "github":
			registry.Register(github.New(pcfg.ClientID, pcfg.ClientSecret, pcfg.Scopes))
		default:
			log.Fatalf("unknown OAuth provider: %s", name)
		}
	}

	h := server.NewHandler(cfg, db, mgr, registry, modelMgr)
	r := server.SetupRouter(h, cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	mgr.Shutdown()

	log.Println("server exited")
}
