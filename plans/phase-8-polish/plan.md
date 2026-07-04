# Phase 8 — Polish

## Goal

Harden the system: log rotation, idle timeout, auto-restart resilience,
tests for the core packages, CI, and release tooling. Nothing user-facing
is added; the goal is production-readiness and maintainability.

## Exit criteria

- Log files rotate at a configurable size and (optionally) compress.
- Idle instances time out and stop automatically per config.
- Instances that crash are restarted up to `max_restarts` with backoff.
- `go test ./...` is green on Linux and macOS.
- A GitHub Actions workflow runs build + test on every push and tags
  cross-platform binaries on release.
- `goreleaser` config produces `llama-admin` (CLI) and `llama-admin-server`
  binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64.

## Sub-tracks

### 8.1 Log rotation

- `pkg/instance/logger.go` wraps a rotation library (or a small hand-rolled
  rotator) honoring:
  - `cfg.Instances.LogRotationEnabled` (default true)
  - `cfg.Instances.LogRotationMaxSize` (MB, default 100)
  - `cfg.Instances.LogRotationCompress` (default false)
- Rotated files: `<name>-<timestamp>.log[.gz]`.

### 8.2 Idle timeout + LRU eviction

- A lifecycle goroutine in `pkg/manager` (started in `manager.New`) wakes
  every `cfg.Instances.TimeoutCheckInterval` minutes (default 5).
- For each running instance with a non-zero idle timeout:
  - If `time.Since(lastRequestTime) > idleTimeout` and `inflight == 0`,
    stop it (`status -> stopped`).
- `lastRequestTime` is already maintained by `proxy.go` from Phase 2.
- LRU eviction: when `max_running_instances` is hit on a start request and
  `enable_lru_eviction` is true, evict the least-recently-used instance in
  the same group (if `group_limits` are set) or globally. Document the
  trade-off in the README.

### 8.3 Auto-restart on crash

- `process.go` (Phase 2) already detects unexpected exits and sets
  `failed`. Phase 8 adds a restart supervisor:
  - If `auto_restart == true` and restart count this lifetime
    `< max_restarts`, schedule a restart after `restart_delay` seconds.
  - Reset the counter on a clean stop.
- Persist `restart_count` in `options_json` so it survives server restarts
  only if desired (default: reset on server restart — simplest, avoids
  runaway loops after a host reboot).

### 8.4 Tests

Priority targets (pure logic, high value):
- `pkg/config`: env expansion, defaults, override precedence.
- `pkg/auth/hash.go`: round-trip + tamper detection.
- `pkg/auth/sessions.go`: token generation uniqueness + hash determinism.
- `pkg/manager/ports.go`: allocate/free/concurrency.
- `pkg/instance/status.go`: state machine transitions.
- `pkg/backends/llama.go`: arg building + validation.
- `pkg/auth/github/github.go`: error mapping using an `httptest` server
  stubbing GitHub's endpoints (pending, slow_down, expired, success, plus
  the `/user/emails` verified filtering).
- `pkg/server`: integration test with an in-memory stub `InstanceManager`
  covering the OpenAI proxy routing logic (no real `llama-server`).

Skip DB-heavy and exec-heavy tests except where they are cheap; the
Phase 1 schema-migration test (run against a temp SQLite file) is worth
keeping green.

### 8.5 CI

`.github/workflows/ci.yml`:
- Matrix: ubuntu-latest, macos-latest (skip cgo-SQLite quirks on windows
  for now, or use a pure-Go SQLite driver — see note below).
- `go vet`, `go test ./...`, `go build ./...`.
- `golangci-lint` with a minimal config (errcheck, govet, staticcheck,
  ineffassign).

### 8.6 Releases

`goreleaser.yml`:
- Two binaries: `llama-admin` (CLI, `./cmd/llama-admin`) and
  `llama-admin-server` (`./cmd/server`).
- ldflags `-X main.version=... -X main.commit=... -X main.buildTime=...`
  for both.
- Archives per OS/arch; checksums; homebrew tap optional.
- GitHub Actions release workflow on tag push.

### 8.7 Optional, time-permitting

- Docker backend support for llama.cpp (config-driven, mirroring
  llamactl's `docker:` block). Useful for GPU hosts without a local
  `llama-server` build.
- Swagger/OpenAPI generation + `/swagger/*` UI toggle
  (`cfg.Server.EnableSwagger`) for external integrators — useful since
  there is no web UI to explore the API.
- Persisting download-job state across restarts (resume job listing).
- A `generic` OAuth provider driven entirely by config (endpoints + JSON
  field paths), removing the need to write a Go package for simple RFC
  8628 providers.

## Notes

- Consider swapping `mattn/go-sqlite3` (cgo) for a pure-Go driver
  (`modernc.org/sqlite`) to simplify cross-compilation in goreleaser.
  Decision can be made in 8.6 when release builds are first attempted; the
  `database` package is the only touch point.
- All Phase 8 changes should be additive config knobs with sensible
  defaults so existing `config.yaml` files keep working.
