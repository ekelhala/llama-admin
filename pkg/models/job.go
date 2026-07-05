package models

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type DownloadStatus string

const (
	StatusPending    DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusVerifying  DownloadStatus = "verifying"
	StatusCompleted  DownloadStatus = "completed"
	StatusFailed     DownloadStatus = "failed"
	StatusCancelled  DownloadStatus = "cancelled"
)

type DownloadJob struct {
	ID         string            `json:"job_id"`
	RepoID     string            `json:"repo_id"`
	Filename   string            `json:"filename"`
	Revision   string            `json:"revision"`
	Status     DownloadStatus    `json:"status"`
	Error      string            `json:"error,omitempty"`
	Progress   ProgressSnapshot  `json:"progress"`
	Cancel     context.CancelFunc `json:"-"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type ProgressSnapshot struct {
	BytesDownloaded int64   `json:"bytes_downloaded"`
	BytesTotal      int64   `json:"bytes_total"`
	Percent         float64 `json:"percent"`
}

func (j *DownloadJob) UpdateProgress(tracker *ProgressTracker) {
	d, t, p := tracker.Snapshot()
	j.Progress = ProgressSnapshot{
		BytesDownloaded: d,
		BytesTotal:      t,
		Percent:         p,
	}
	j.UpdatedAt = time.Now()
}

func (j *DownloadJob) SetStatus(s DownloadStatus) {
	j.Status = s
	j.UpdatedAt = time.Now()
}

func (j *DownloadJob) SetError(err error) {
	j.Error = err.Error()
	j.SetStatus(StatusFailed)
}

type JobRegistry struct {
	mu      sync.RWMutex
	jobs    map[string]*DownloadJob
	counter int
}

func NewJobRegistry() *JobRegistry {
	return &JobRegistry{
		jobs: make(map[string]*DownloadJob),
	}
}

func (r *JobRegistry) Create(repoID, filename, revision string) *DownloadJob {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counter++
	job := &DownloadJob{
		ID:       fmt.Sprintf("job-%d", r.counter),
		RepoID:   repoID,
		Filename: filename,
		Revision: revision,
		Status:   StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.jobs[job.ID] = job
	return job
}

func (r *JobRegistry) Get(id string) (*DownloadJob, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.jobs[id]
	return job, ok
}

func (r *JobRegistry) List() []*DownloadJob {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*DownloadJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		result = append(result, job)
	}
	return result
}

func (r *JobRegistry) Cancel(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[id]
	if !ok || job.Cancel == nil {
		return false
	}
	job.Cancel()
	job.SetStatus(StatusCancelled)
	return true
}
