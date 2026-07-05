package models

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Downloader struct {
	cacheDir  string
	timeout   time.Duration
	version   string
	httpClient *http.Client
}

func NewDownloader(cacheDir, version string, timeout time.Duration) *Downloader {
	return &Downloader{
		cacheDir:  cacheDir,
		timeout:   timeout,
		version:   version,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (d *Downloader) StartDownload(ctx context.Context, job *DownloadJob, tracker *ProgressTracker) error {
	job.SetStatus(StatusDownloading)

	repoID := job.RepoID
	filename := job.Filename
	revision := job.Revision
	if revision == "" {
		revision = "main"
	}

	// Build URL
	baseURL := fmt.Sprintf("https://huggingface.co/%s/resolve/%s/%s", repoID, revision, filename)

	// Build local path
	sanitizedRepo := strings.ReplaceAll(repoID, "/", "--")
	localDir := filepath.Join(d.cacheDir, sanitizedRepo)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		job.SetError(fmt.Errorf("create dir: %w", err))
		return err
	}

	partPath := localDir + ".part"
	finalPath := filepath.Join(localDir, filename)

	// Resume support
	var startOffset int64
	if info, err := os.Stat(partPath); err == nil {
		startOffset = info.Size()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		job.SetError(err)
		return err
	}
	req.Header.Set("User-Agent", "llama-admin/"+d.version)
	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		job.SetError(err)
		return err
	}
	defer resp.Body.Close()

	// Handle partial content (resume)
	if resp.StatusCode == http.StatusPartialContent && startOffset > 0 {
		// Good, resuming
	} else if resp.StatusCode != http.StatusOK {
		job.SetError(fmt.Errorf("unexpected status: %d", resp.StatusCode))
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Get total size
	totalSize := int64(resp.ContentLength)
	if totalSize > 0 {
		tracker.SetTotal(totalSize)
		if startOffset > 0 {
			tracker.downloaded.Add(startOffset)
		}
	}

	// Open file
	f, err := os.OpenFile(partPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		job.SetError(err)
		return err
	}
	defer f.Close()

	// Stream download
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				job.SetError(writeErr)
				return writeErr
			}
			tracker.Add(int64(n))
			job.UpdateProgress(tracker)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			job.SetError(err)
			return err
		}
	}

	// Verify and rename
	if err := f.Close(); err != nil {
		job.SetError(err)
		return err
	}

	if err := os.Rename(partPath, finalPath); err != nil {
		job.SetError(err)
		return err
	}

	job.SetStatus(StatusCompleted)
	return nil
}
