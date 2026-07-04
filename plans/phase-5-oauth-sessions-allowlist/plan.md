# Phase 5 — OAuth providers, sessions, and email allowlist

## Goal

Introduce a generalized OAuth `Provider` abstraction with GitHub as the
reference implementation. Users log in via Device Flow (RFC 8628); the
server validates the provider token, upserts a user, checks the email
allowlist, and issues a **session token** the CLI uses for management.
Any logged-in user can extend the allowlist at runtime.

After this phase, all `/api/v1/*` management endpoints (including the
key-management routes from Phase 4) require a valid session token. `/v1/*`
still uses API keys.

## Exit criteria

- `GET /api/v1/auth/providers` lists enabled providers.
- `POST /api/v1/auth/{provider}/device` returns a device code + verification
  URL (proxied from the provider).
- `POST /api/v1/auth/{provider}/token` with the device code: the server
  polls the provider, exchanges for an access token, fetches user info,
  verifies a **verified** email is in the allowlist, upserts the user,
  issues a session token, returns `{session_token, expires_at, user}`.
- `GET /api/v1/auth/session` returns the current user (for `auth status`).
- `DELETE /api/v1/auth/session` revokes the current session (logout).
- `GET /api/v1/auth/allowed-emails` lists config + DB entries, tagged by source.
- `POST /api/v1/auth/allowed-emails` adds an email to the DB allowlist.
- `DELETE /api/v1/auth/allowed-emails/{email}` removes a DB entry (config
  entries are not removable via API).
- All `/api/v1/*` routes except the `/auth/{provider}/*` and
  `/auth/providers` endpoints require a valid session token.
- A login attempt whose verified emails are all outside the allowlist
  returns 403 and creates no user/session.

This phase is split across two files for readability:
- `plan.md` — this file: overview, API, data model, flow.
- `provider-interface.md` — the `Provider` abstraction, GitHub reference
  impl, and how to add a new provider.

## Files to create / modify

```
pkg/auth/
    provider.go          # Provider interface + DeviceCode + UserInfo + registry
    sessions.go         # session token generation (32 bytes -> "las-<hex>"),
                        #  hash (SHA-256; tokens are high-entropy so Argon2 is
                        #  unnecessary), issue/verify/revoke
    github/
        github.go       # reference Provider implementation
pkg/database/
    users.go            # UpsertUser, GetUser, GetUserByProviderID
    sessions.go         # CreateSession, GetSessionByHash, TouchSession,
                        #   DeleteSession, DeleteExpired
    allowed_emails.go   # ListAllowedEmails, AddAllowedEmail, RemoveAllowedEmail
pkg/server/
    handlers_auth_oauth.go   # /auth/providers, /auth/{p}/device,
                             #   /auth/{p}/token, /auth/session
    handlers_auth_allowlist.go   # /auth/allowed-emails CRUD
    middleware.go            # ManagementAuthMiddleware (session tokens)
    routes.go               # mount auth routes; gate /api/v1/* with sessions
```

## API contract

```
GET    /api/v1/auth/providers
       -> {providers: [{name:"github", device_flow:true}]}

POST   /api/v1/auth/{provider}/device
       -> {device_code, user_code, verification_uri, expires_in, interval}

POST   /api/v1/auth/{provider}/token   body: {device_code}
       -> {session_token, expires_at, user:{id,username,email,avatar_url,provider}}

GET    /api/v1/auth/session             (session required)
       -> {user:{...}, expires_at}

DELETE /api/v1/auth/session             (session required)
       -> 204

GET    /api/v1/auth/allowed-emails      (session required)
       -> {emails:[{email, source:"config"|"api", added_by?}]}

POST   /api/v1/auth/allowed-emails      body:{email}  (session required)
       -> 201

DELETE /api/v1/auth/allowed-emails/{email}  (session required)
       -> 204 (409 if the email is config-sourced)
```

The `/auth/{provider}/device`, `/auth/{provider}/token`, and
`/auth/providers` endpoints are **public** (no session). Everything else
under `/api/v1/*` is gated by `ManagementAuthMiddleware`.

## Data model

Tables `users`, `sessions`, `allowed_emails` were created in Phase 1.

- Users are unique on `(provider, provider_user_id)` so multiple providers
  coexist.
- Session tokens are random 32-byte hex with an `las-` prefix. Stored hashed
  with SHA-256 (sufficient because tokens are 256-bit random; Argon2id is
  only needed for low-entropy human-chosen secrets).
- `sessions.provider` records which provider issued the session.
- `allowed_emails.email` is the PRIMARY KEY. Config-seed entries are loaded
  into memory only; they are never inserted into the table, so the table
  holds only API-added entries. `ListAllowedEmails` merges the in-memory
  config set with the table rows.

## Login flow (device flow)

1. CLI calls `POST /api/v1/auth/{provider}/device`. Server calls
   `provider.InitiateDeviceFlow(ctx)` which hits the provider's device
   authorization endpoint and returns `{device_code, user_code,
   verification_uri, expires_in, interval}`. Server forwards these to the
   CLI.
2. CLI displays the user code + URL, then polls
   `POST /api/v1/auth/{provider}/token` with `{device_code}` at the
   provider's `interval`.
3. Server calls `provider.ExchangeDeviceCode(ctx, device_code)`. If the
   provider says `authorization_pending`, server returns 408 so the CLI
   retries. On success the server gets a provider access token.
4. Server calls `provider.FetchUserInfo(ctx, accessToken)` → `UserInfo`
   with `VerifiedEmails []string` (the github provider calls `/user/emails`
   and filters `verified: true`).
5. Allowlist check: is the intersection of `VerifiedEmails` and the
   effective allowlist (config ∪ DB) non-empty?
   - No → 403 `email_not_allowed`; no user upsert, no session.
   - Yes → `UpsertUser(provider, provider_user_id, ...)`, pick the first
     matching verified email as the user's `email`.
6. Issue session: generate `las-<hex>`, hash with SHA-256, insert row with
   `expires_at = now + cfg.Auth.Session.TTL`, return plaintext token + user.

## Middleware behavior

`ManagementAuthMiddleware` (applied to `/api/v1/*` except the public auth
routes):

1. `OPTIONS` → pass through.
2. Extract `Authorization: Bearer <token>` (no `X-API-Key` here — keep
   inference keys and sessions visually distinct). Missing → 401.
3. Hash the token (SHA-256), `GetSessionByHash`; not found → 401.
4. `expires_at < now` → 401 `session_expired` (and delete the row).
5. Set `*User` in context; asynchronously `TouchSession(id)`.
6. Continue.

Handlers that need the user read it from context via a helper
(`UserFromContext(ctx) *User`). The allowlist-management handlers use this
to record `added_by_user_id`.

## Implementation steps

1. `pkg/auth/provider.go`: define `Provider`, `DeviceCode`, `UserInfo`,
   `ProviderRegistry` (a `map[string]Provider` with `Get(name)`).
2. `pkg/auth/github/github.go`: implement `Provider` using GitHub's endpoints
   (see `provider-interface.md` for details).
3. `pkg/auth/sessions.go`: `GenerateSessionToken()`, `HashSessionToken()`.
4. `pkg/database/{users,sessions,allowed_emails}.go`: data access.
5. `pkg/server/handlers_auth_oauth.go`: providers list, device initiation,
   token exchange (server-side polling, returning 408 on pending), session
   get/delete.
6. `pkg/server/handlers_auth_allowlist.go`: list (merge config + DB), add,
   remove (reject config-sourced deletes with 409).
7. `pkg/server/middleware.go`: add `ManagementAuthMiddleware`; refactor the
   existing `apiKeyContextKey` to add `userContextKey`.
8. `pkg/server/routes.go`: mount auth routes; apply
   `ManagementAuthMiddleware` to the `/api/v1` group **after** the public
   auth subroutes are carved out (chi lets you `r.With(mw).Route(...)` for
   the gated subset, leaving the public routes unwrapped).
9. Wire provider registry construction in `cmd/server/main.go`: read
   `cfg.Auth.Providers`, instantiate the `github` provider for any
   `enabled: true` entry, register it. Unknown provider names in config
   are a startup error.
10. Manual test: run the device flow by hand with `curl`, confirm a session
    token is issued and `GET /api/v1/auth/session` returns the user.

## Notes

- Server-side polling in step 3 keeps the client simple: the CLI just
  retries its single `POST .../token` call until it gets a non-408 answer.
  The server must respect the provider's `interval` and stop polling after
  `expires_in`.
- Config-seed `allowed_emails` are loaded into a `map[string]struct{}`
  at startup. The effective set is recomputed on each login by unioning
  with the DB table.
- On first run with an empty allowlist, the operator logs in by adding
  their email to `config.yaml` and restarting; afterwards they can invite
  others via the API. Document this in the CLI help in Phase 6.
- Do not implement token refresh of the provider access token. Sessions
  are first-class in llama-admin; the provider token is only used during
  the login exchange and is not stored. If we later want to call provider
  APIs on the user's behalf, store + refresh the provider token in a
  separate table — out of scope here.
