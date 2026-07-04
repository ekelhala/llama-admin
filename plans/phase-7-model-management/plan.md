# Phase 7 — Model management

## Goal

Download GGUF models from HuggingFace and scan local directories for
existing models, so instances can reference models by path or by a
friendly name. Download jobs run in the background and are queryable.

## Exit criteria

- `POST /api/v1/models/download` with `{repo_id, filename, revision?}`
  starts a background download and returns a job id.
- `GET /api/v1/models/download/jobs` lists jobs; `GET .../jobs/{id}`
  returns progress + status + error.
- `GET /api/v1/models` lists models found in the configured cache directory
  (scanned for `.gguf` files) plus any downloaded by llama-admin.
- `llama-admin models download <repo_id> [<filename>]` streams progress to
  the terminal; `llama-admin models list` lists local models.
- An instance created with `--model <downloaded path>` (from Phase 2)
  starts and serves it.

## Files to create / modify

```
pkg/models/
    manager.go          # Manager: owns downloader + scanner + job registry
    downloader.go       # HuggingFace download with resume (Range requests),
                        #  SHA-256 verify against HF-provided checksum
    scanner.go          # walk cfg.Backends.LlamaCpp.CacheDir for *.gguf
    job.go              # DownloadJob: id, status, progress, error, cancel
    progress_tracker.go # atomic bytes downloaded / total + ETA
pkg/server/
    handlers_models.go  # download, list, job endpoints
pkg/database/
    models.go           # (optional) persist job metadata across restarts
internal/cmd/
    models.go           # extend Phase 6's models commands
```

## API contract

```
POST   /api/v1/models/download       body:{repo_id, filename, revision?}
                                   -> {job_id, status:"pending"}
GET    /api/v1/models/download/jobs  -> [{job_id, repo_id, filename, status,
                                          progress, error?}]
GET    /api/v1/models/download/jobs/{id} -> job detail
DELETE /api/v1/models/download/jobs/{id} -> cancel a running job (204)
GET    /api/v1/models               -> [{name, path, size_bytes, source}]
```

`progress` is `{"bytes_downloaded":..., "bytes_total":..., "percent":...}`.

## Downloader

- Resolve the file URL via
  `https://huggingface.co/{repo_id}/resolve/{revision}/{filename}` (default
  revision `main`).
- Stream to `<cache_dir>/<sanitized_repo>/<filename>` writing to a `.part`
  file first, renaming on completion.
- Resume using the existing `.part` size via `Range: bytes=<n>-`.
- Verify the final file against the ETag/SHA from HF response headers when
  available; on mismatch, delete and re-download once.
- Use `User-Agent: llama-admin/<version>` (HF rate-limits anonymous
  downloads; a `HF_TOKEN` env var can be wired in Phase 8 for private repos).
- Respect `cfg.Backends.LlamaCpp.DownloadTimeout`.

## Scanner

- Walk `cfg.Backends.LlamaCpp.CacheDir` (default
  `<data_dir>/models`) for `*.gguf` files.
- Return `{name, path, size_bytes, source: "scan"|"download"}`.
- Cheap to run on every `GET /models` for small directories; for large
  libraries, cache results with a directory mtime check.

## Jobs

- In-memory registry keyed by job id (uuid or counter).
- Status: `pending | downloading | verifying | completed | failed | cancelled`.
- A `context.CancelFunc` per job lets `DELETE .../jobs/{id}` stop the
  download.
- On server restart, in-progress jobs are lost (acceptable for downloads;
  the `.part` file lets the user retry and resume). Persisting job state
  across restarts is a Phase 8 nicety.

## Implementation steps

1. `pkg/models/progress_tracker.go`: atomic counters + a `Snapshot()`.
2. `pkg/models/job.go`: `DownloadJob` struct + lifecycle methods.
3. `pkg/models/downloader.go`: HTTP GET with Range resume, stream to
   `.part`, rename, verify.
4. `pkg/models/scanner.go`: `filepath.WalkDir` collecting `.gguf` entries.
5. `pkg/models/manager.go`: `NewManager(cacheDir, timeout, version)`,
   `StartDownload(repo, file, rev)`, `ListJobs()`, `GetJob(id)`,
   `CancelJob(id)`, `ListModels()`. A `Close()` stops background work on
   server shutdown.
6. `pkg/server/handlers_models.go`: the four endpoints.
7. `pkg/server/routes.go`: mount under `/api/v1/models`.
8. `internal/cmd/models.go`: `download` (polls job endpoint with a
   progress bar), `list`.
9. `cmd/server/main.go`: construct `models.NewManager(...)` and pass to
   the handler; call `Close()` on shutdown.

## Notes

- The download path under `<cache_dir>/<sanitized_repo>/<filename>` matches
  HuggingFace's own layout, so models pulled by `huggingface-cli` show up
  in the scanner too.
- Multi-file Safetensors downloads are out of scope (llama.cpp GGUF is a
  single file). vLLM/Safetensors support would come with multi-backend
  work (not currently planned).
- `GET /v1/models` (the OpenAI endpoint from Phase 3) still lists
  **instances**, not raw model files. The Phase 7 `/api/v1/models` endpoint
  lists **files**. They are deliberately separate concepts. A later
  enhancement (router-mode enumeration, deferred) would have `/v1/models`
  also include per-instance loaded models.
