package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llama-admin/pkg/config"
	"llama-admin/pkg/database"
)

type Manager struct {
	downloader *Downloader
	registry   *JobRegistry
	cacheDir   string
	timeout    time.Duration
	version    string
	cfg        *config.AppConfig
	modelStore *database.ModelStore
	mu         sync.Mutex
}

func NewManager(cfg *config.AppConfig, version string, modelStore *database.ModelStore) *Manager {
	return &Manager{
		downloader: NewDownloader(cfg.Backends.LlamaCpp.CacheDir, version, cfg.Backends.LlamaCpp.DownloadTimeout),
		registry:   NewJobRegistry(),
		cacheDir:   cfg.Backends.LlamaCpp.CacheDir,
		timeout:    cfg.Backends.LlamaCpp.DownloadTimeout,
		version:    version,
		cfg:        cfg,
		modelStore: modelStore,
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

func (m *Manager) RegisterModel(alias, filename string, sizeBytes int64) (*database.Model, error) {
	mu := &database.Model{
		Alias:     alias,
		Filename:  filename,
		SizeBytes: sizeBytes,
	}
	if err := m.modelStore.Save(mu); err != nil {
		return nil, fmt.Errorf("register model: %w", err)
	}
	return mu, nil
}

func (m *Manager) GetModel(alias string) (*database.Model, error) {
	return m.modelStore.GetByAlias(alias)
}

func (m *Manager) ListModels() ([]*database.Model, error) {
	return m.modelStore.List()
}

func (m *Manager) DeleteModel(alias string) error {
	return m.modelStore.Delete(alias)
}

func (m *Manager) ResolveAlias(alias string) (string, error) {
	mod, err := m.modelStore.GetByAlias(alias)
	if err != nil {
		return "", err
	}
	return mod.Filename, nil
}

func (m *Manager) ListFiles() ([]ModelFileInfo, error) {
	return ScanModelFiles(m.cacheDir)
}

func (m *Manager) Close() {
	// Stop all running jobs
	for _, job := range m.registry.List() {
		if job.Cancel != nil && (job.Status == StatusDownloading || job.Status == StatusPending) {
			job.Cancel()
		}
	}
}
