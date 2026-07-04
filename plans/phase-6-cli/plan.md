# Phase 6 — CLI

## Goal

Ship a `llama-admin` CLI (cobra) that authenticates the user via OAuth
Device Flow and manages the server. The CLI is the only management
interface (no web UI).

## Exit criteria

- `llama-admin auth login [--provider github] [--server URL]` completes the
  device flow and stores a session token locally.
- `llama-admin auth status` shows the current user + server.
- `llama-admin auth logout` revokes the session and clears local state.
- `llama-admin instances list|get|create|start|stop|restart|delete|logs`
  work end-to-end against the server.
- `llama-admin api-keys create|list|delete|grant|revoke` work.
- `llama-admin allowed-emails add|list|remove` work.
- `llama-admin config set|show` manage the local config file.
- `llama-admin models list` works (download arrives in Phase 7).
- All commands fail cleanly when not logged in, with a hint to run
  `auth login`.

## Files to create

```
cmd/llama-admin/main.go             # cobra root, wires deps, version flag
internal/config/
    config.go                       # load/save ~/.config/llama-admin/config.json
    types.go
internal/client/
    client.go                       # typed HTTP client: base URL, bearer token,
                                    #  do/Get/Post/Delete + error decoding
internal/auth/
    device.go                       # client-side device flow: poll server's
                                    #  /auth/{p}/device + /auth/{p}/token
internal/cmd/
    root.go                         # root command, --server, --config flags
    auth.go                         # login/status/logout
    instances.go                    # instance subcommands
    api_keys.go                     # api-keys subcommands
    allowed_emails.go               # allowed-emails subcommands
    models.go                       # models list (download in Phase 7)
    config.go                       # config set/show
    format.go                       # shared table/JSON output helpers
```

## Local config

`~/.config/llama-admin/config.json`:
```json
{
  "server_url": "http://localhost:8080",
  "session_token": "las-...",
  "provider": "github"
}
```
- `--server` flag overrides `server_url` per invocation; `config set
  server_url` persists it.
- `session_token` is written only by `auth login` and read by every
  command that talks to the server. Permissions: `0600`.
- `--output json|table` flag on list commands; default `table`.

## Command tree

```
llama-admin
├── --version
├── auth
│   ├── login   [--provider github] [--server URL]
│   ├── status
│   └── logout
├── instances
│   ├── list
│   ├── get <name>
│   ├── create <name> --model <path> [--ctx-size N] [--gpu-layers N] ...
│   ├── start <name>
│   ├── stop <name>
│   ├── restart <name>
│   ├── delete <name>
│   └── logs <name> [--lines N]
├── api-keys
│   ├── create <name> [--mode allow_all|per_instance] [--expires <dur>]
│   ├── list
│   ├── delete <id>
│   ├── grant <id> --instance <name>
│   └── revoke <id> --instance <name>
├── allowed-emails
│   ├── list
│   ├── add <email>
│   └── remove <email>
├── models
│   └── list                       # download added in Phase 7
└── config
    ├── set <key> <value>
    └── show
```

## Device flow client

`internal/auth/device.go`:

1. If `--provider` is empty, `GET /api/v1/auth/providers`; if exactly one
   is enabled, use it; otherwise prompt the user to pick (or error in
   non-interactive mode).
2. `POST /api/v1/auth/{provider}/device` → print the user code and
   `verification_uri`, instruct the user to open the URL.
3. Poll `POST /api/v1/auth/{provider}/token` with `{device_code}`:
   - 408 → wait `interval` seconds, retry (until `expires_in`).
   - 429 → back off (double the interval, capped).
   - 2xx → success: persist `session_token` + `provider` + `server_url`
     to the local config, print a success message + user info.
   - 4xx/5xx other → fail with the server's error message.

## HTTP client

`internal/client/client.go`:
- `New(serverURL, sessionToken)` returns a `*Client`.
- `do(method, path, body)` injects `Authorization: Bearer <sessionToken>`
  (or no header for the public auth endpoints), sends the request, decodes
  the OpenAI-shaped error envelope on non-2xx into a typed `APIError`
  containing `Status`, `Message`, `Type`.
- Typed helpers: `ListInstances()`, `CreateInstance(...)`, `StartInstance(name)`,
  `ListAPIKeys()`, `CreateAPIKey(...)`, `AddAllowedEmail(email)`, etc. These
  keep command code thin and make the API surface easy to test.

## Implementation steps

1. `cmd/llama-admin/main.go`: minimal cobra root + `--version` reading
   ldflags-injected vars (mirror the server's pattern).
2. `internal/config`: load + save with `0600` perms; XDG-aware default path.
3. `internal/client`: base implementation + `APIError`; cover the auth
   endpoints first so `auth login` can be built.
4. `internal/auth/device.go`: the flow above.
5. `internal/cmd/auth.go`: `login`, `status` (`GET /auth/session`), `logout`
   (`DELETE /auth/session` + clear local token).
6. `internal/cmd/instances.go`: wire each subcommand to the client; flag
   parsing for `create` mirrors the server's `Options` (backend options
   nested under `backend_options`).
7. `internal/cmd/api_keys.go`, `allowed_emails.go`, `config.go`,
   `models.go`.
8. `internal/cmd/format.go`: simple `text/tabwriter` table renderer +
   `--output json` passthrough.
9. Manual end-to-end test: `auth login` → `instances create tiny --model
   /path/gguf` → `instances start tiny` → `api-keys create` → use the key
   against `/v1/chat/completions` → `auth logout`.

## Notes

- The CLI **never** handles provider credentials directly — it always
  goes through the server's `/auth/{provider}/*` endpoints. This keeps
  the OAuth App secret server-side and lets the CLI work over SSH without
  a browser callback.
- `api-keys create` prints the plaintext key once, with a clear warning
  that it will not be shown again.
- `instances create` flags should map 1:1 to the most common llama.cpp
  options for Phase 2; exotic flags can be passed via a generic
  `--backend-option key=value` repeatable flag that injects into
  `backend_options`. This keeps the CLI future-proof as backend options grow.
- The CLI's session token is long-lived per `cfg.Auth.Session.TTL`. If we
  later want short-lived tokens + refresh, the server can issue a refresh
  token alongside; out of scope here.
