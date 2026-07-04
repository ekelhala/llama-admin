# llama-admin

Control server for managing `llama.cpp` instances and routing OpenAI-compatible
requests, administered through a CLI that authenticates users via OAuth
(GitHub by default, extensible to other providers).

## Overview

llama-admin is a small Go server you deploy on your inference host. It:

- Spawns and supervises `llama-server` processes (one or more instances)
- Exposes an OpenAI-compatible API (`/v1/*`) protected by API keys, routing
  requests to instances based on the request `model` field
- Provides a management API (`/api/v1/*`) protected by OAuth-issued session
  tokens
- Ships a CLI (`llama-admin`) that authenticates the user via OAuth device
  flow and manages instances, models, and API keys

## Status

Pre-implementation. The build is organized into phases tracked under
[`plans/`](plans). See [`plans/README.md`](plans/README.md) for the roadmap.

## Design decisions

- **Server + CLI only** (no web UI) — all management happens through the CLI.
- **llama.cpp backend only** initially (multi-backend support deferred).
- **Local instances only** (remote-node support deferred).
- **Generalized OAuth** via a `Provider` interface; GitHub is the reference
  implementation. Adding a provider is a small Go package + a config entry.
- **Two auth mechanisms:**
  - Session tokens (OAuth device flow) protect **management** endpoints.
  - API keys (Argon2id-hashed, per-instance scoping) protect **inference**
    endpoints, so external OpenAI clients need no OAuth.
- **Email allowlist** gates who can log in. Seeded from config and extended
  at runtime via the API.

## Tech stack

- Go 1.24+
- [chi](https://github.com/go-chi/chi) router
- SQLite (via `mattn/go-sqlite3`)
- [cobra](https://github.com/spf13/cobra) for the CLI
- Argon2id for key/token hashing

## License

MIT
