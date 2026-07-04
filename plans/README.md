# Implementation Plan

llama-admin is built in eight phases. Each phase lives in its own folder and
has an exit criterion that lets you verify it works before moving on.

| Phase | Folder | Summary | Exit criterion |
|-------|--------|---------|----------------|
| 1 | [`phase-1-server-skeleton/`](phase-1-server-skeleton/plan.md) | Go module, config, SQLite, chi router skeleton | `go run ./cmd/server` starts and serves `/api/v1/version` |
| 2 | [`phase-2-instance-lifecycle/`](phase-2-instance-lifecycle/plan.md) | llama.cpp process spawning, registry, port allocation, management API | Create an instance via curl, it spawns `llama-server` and proxies to it |
| 3 | [`phase-3-openai-proxy/`](phase-3-openai-proxy/plan.md) | `/v1/*` proxy with routing by `model` field, streaming, on-demand start | `curl /v1/chat/completions` returns a completion |
| 4 | [`phase-4-api-key-auth/`](phase-4-api-key-auth/plan.md) | Argon2id API keys, per-instance permissions, inference auth middleware | `/v1/*` requires a valid API key; keys scoped per-instance |
| 5 | [`phase-5-oauth-sessions-allowlist/`](phase-5-oauth-sessions-allowlist/plan.md) | `Provider` interface, GitHub provider, session tokens, email allowlist | CLI obtains a session token; management endpoints require it |
| 6 | [`phase-6-cli/`](phase-6-cli/plan.md) | cobra CLI: auth, instances, models, api-keys, config | Full CLI workflow works end-to-end |
| 7 | [`phase-7-model-management/`](phase-7-model-management/plan.md) | HuggingFace downloader, model scanner, download jobs | Download a model from HF and reference it in an instance |
| 8 | [`phase-8-polish/`](phase-8-polish/plan.md) | Log rotation, idle timeout, auto-restart, tests, CI, releases | All quality gates pass; release binaries build |

## Cross-cutting decisions

These were settled during planning and apply across phases:

- **Backend scope:** llama.cpp only initially.
- **Deployment scope:** local instances only initially.
- **Interface:** CLI only (no web UI).
- **OAuth flow:** Device Flow (RFC 8628); the server proxies the flow so the
  CLI never ships provider credentials.
- **Auth split:** session tokens (OAuth) for management; API keys for
  inference. They never overlap.
- **Allowlist gating:** config-seeded + DB-extendable; checks only verified
  emails; any logged-in user can manage the allowlist via the API.
- **Instance options:** stored as a single JSON blob in SQLite (schema
  evolves without migrations); Go structs handle (un)marshaling.

## Suggested file layout (created starting in Phase 1)

```
llama-admin/
├── cmd/
│   ├── server/main.go
│   └── llama-admin/main.go          # Phase 6
├── pkg/                              # server packages
│   ├── config/
│   ├── database/
│   ├── auth/
│   │   └── github/                   # Phase 5 reference provider
│   ├── backends/
│   ├── instance/
│   ├── manager/
│   ├── models/                      # Phase 7
│   └── server/
├── internal/                         # CLI-only packages (Phase 6)
│   ├── config/
│   ├── client/
│   ├── auth/
│   └── cmd/
├── plans/
└── go.mod
```
