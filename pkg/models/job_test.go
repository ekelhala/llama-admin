package models

import (
	"context"
	"testing"
	"time"
)

func TestJobRegistry_CreateAssignsSequentialIDs(t *testing.T) {
	r := NewJobRegistry()

	j1 := r.Create("owner/model-a", "model-a-Q4.gguf", "main")
	j2 := r.Create("owner/model-b", "model-b-Q4.gguf", "main")

	if j1.ID == "" || j2.ID == "" {
		t.Fatalf("expected non-empty job IDs: %q %q", j1.ID, j2.ID)
	}
	if j1.ID == j2.ID {
		t.Fatalf("expected distinct IDs, both = %q", j1.ID)
	}

	if j1.Status != StatusPending {
		t.Errorf("new job status = %q, want pending", j1.Status)
	}
	if j1.RepoID != "owner/model-a" {
		t.Errorf("RepoID = %q, want owner/model-a", j1.RepoID)
	}
	if j1.Filename != "model-a-Q4.gguf" {
		t.Errorf("Filename = %q, want model-a-Q4.gguf", j1.Filename)
	}
	if j1.Revision != "main" {
		t.Errorf("Revision = %q, want main", j1.Revision)
	}
	if j1.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestJobRegistry_Get(t *testing.T) {
	r := NewJobRegistry()
	j := r.Create("repo", "file.gguf", "main")

	got, ok := r.Get(j.ID)
	if !ok {
		t.Fatalf("Get(%q) returned ok=false", j.ID)
	}
	if got != j {
		t.Fatal("Get returned a different job pointer")
	}

	if _, ok := r.Get("does-not-exist"); ok {
		t.Fatal("Get for unknown id returned ok=true")
	}
}

func TestJobRegistry_ListReturnsAllJobs(t *testing.T) {
	r := NewJobRegistry()
	r.Create("a", "a.gguf", "main")
	r.Create("b", "b.gguf", "main")
	r.Create("c", "c.gguf", "main")

	got := r.List()
	if len(got) != 3 {
		t.Fatalf("List returned %d jobs, want 3", len(got))
	}

	seen := map[string]bool{}
	for _, j := range got {
		seen[j.ID] = true
	}
	if len(seen) != 3 {
		t.Fatalf("List returned duplicate IDs: %v", got)
	}
}

func TestJobRegistry_Cancel(t *testing.T) {
	r := NewJobRegistry()
	j := r.Create("repo", "file.gguf", "main")

	cancelled := false
	j.Cancel = func() {
		cancelled = true
	}

	if !r.Cancel(j.ID) {
		t.Fatal("Cancel returned false for existing job")
	}
	if j.Status != StatusCancelled {
		t.Errorf("status after Cancel = %q, want cancelled", j.Status)
	}
	if !cancelled {
		t.Error("Cancel function was not invoked")
	}
	if !j.UpdatedAt.After(j.CreatedAt) {
		t.Error("UpdatedAt not bumped after Cancel")
	}
}

func TestJobRegistry_CancelUnknownJob(t *testing.T) {
	r := NewJobRegistry()
	if r.Cancel("nope") {
		t.Fatal("expected false cancelling unknown job")
	}
}

func TestDownloadJob_SetStatusBumpsUpdatedAt(t *testing.T) {
	j := &DownloadJob{Status: StatusPending, UpdatedAt: time.Now()}
	prev := j.UpdatedAt

	time.Sleep(time.Millisecond)
	j.SetStatus(StatusDownloading)
	if j.Status != StatusDownloading {
		t.Fatalf("status = %q, want downloading", j.Status)
	}
	if !j.UpdatedAt.After(prev) {
		t.Error("UpdatedAt not bumped by SetStatus")
	}
}

func TestDownloadJob_SetError(t *testing.T) {
	j := &DownloadJob{Status: StatusDownloading}
	j.SetError(context.DeadlineExceeded)

	if j.Status != StatusFailed {
		t.Errorf("status after SetError = %q, want failed", j.Status)
	}
	if j.Error == "" {
		t.Error("Error message is empty after SetError")
	}
}

func TestDownloadJob_UpdateProgress(t *testing.T) {
	tracker := &ProgressTracker{}
	tracker.SetTotal(1000)
	tracker.Add(500)

	j := &DownloadJob{UpdatedAt: time.Now()}
	time.Sleep(time.Millisecond)
	j.UpdateProgress(tracker)

	if j.Progress.BytesDownloaded != 500 {
		t.Errorf("BytesDownloaded = %d, want 500", j.Progress.BytesDownloaded)
	}
	if j.Progress.BytesTotal != 1000 {
		t.Errorf("BytesTotal = %d, want 1000", j.Progress.BytesTotal)
	}
	if j.Progress.Percent != 50.0 {
		t.Errorf("Percent = %v, want 50", j.Progress.Percent)
	}
}
