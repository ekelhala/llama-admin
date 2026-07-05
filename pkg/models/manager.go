package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llama-admin/pkg/config"
)

type Manager struct {
	downloader *Downloader
	registry   *JobRegistry
	cacheDir   string
	timeout    time.Duration
	version    string
	cfg        *config.AppConfig
	mu         sync.Mutex
}

func NewManager(cfg *config.AppConfig, version string) *Manager {
	return &Manager{
		downloader: NewDownloader(cfg.Backends.LlamaCpp.CacheDir, version, cfg.Backends.LlamaCpp.DownloadTimeout),
		registry:   NewJobRegistry(),
		cacheDir:   cfg.Backends.LlamaCpp.CacheDir,
		timeout:    cfg.Backends.LlamaCpp.DownloadTimeout,
		version:    version,
		cfg:        cfg,
	}
}

func (m *Manager) StartDownload(repoID, filename, revision string) (*DownloadJob, error) {
	job := m.registry.Create(repoID, filename, revision)

	ctx, cancel := context.WithCancel(context.Background())
	job.Cancel = cancel

	go func() {
		tracker := &ProgressTracker{}
		if err := m.downloader.StartDownload(ctx, job, tracker); err != nil {
			// Error already set by downloader
		}
	}()

	return job, nil
}

func (m *Manager) ListJobs() []*DownloadJob {
	return m.registry.List()
}

func (m *Manager) GetJob(id string) (*DownloadJob, bool) {
	return m.registry.Get(id)
}

func (m *Manager) CancelJob(id string) error {
	if !m.registry.Cancel(id) {
		return fmt.Errorf("job %q not found", id)
	}
	return nil
}

func (m *Manager) ListModels() ([]ModelFileInfo, error) {
	return ScanModels(m.cacheDir)
}

func (m *Manager) Close() {
	// Stop all running jobs
	for _, job := range m.registry.List() {
		if job.Cancel != nil && (job.Status == StatusDownloading || job.Status == StatusPending) {
			job.Cancel()
		}
	}
}
