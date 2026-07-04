# Phase 4 — API key auth for inference

## Goal

Protect `/v1/*` with API keys. Keys are created via the management API,
hashed with Argon2id at rest, and optionally scoped to a subset of
instances. External OpenAI clients authenticate with
`Authorization: Bearer <key>` (or `X-API-Key`).

This phase does **not** protect `/api/v1/*` management endpoints — that
is Phase 5. Management stays open through Phase 4.

## Exit criteria

- `POST /api/v1/auth/keys` creates a key and returns the plaintext key once.
- `GET /api/v1/auth/keys` lists keys (never returns plaintext).
- `DELETE /api/v1/auth/keys/{id}` revokes.
- `GET/POST/DELETE /api/v1/auth/keys/{id}/permissions` manage per-instance
  scoping.
- All `/v1/*` requests require a valid key; missing/invalid/expired → 401
  in the OpenAI error envelope.
- A key with `permission_mode=per_instance` and a permission grant for
  instance `tiny` can hit `/v1/chat/completions` with `model=tiny` but
  gets 403 for any other instance. `allow_all` keys bypass the check.
- `last_used_at` is updated asynchronously after successful auth.

## Files to create / modify

```
pkg/auth/
    hash.go           # Argon2id HashKey/VerifyKey (params: t=1, m=64MiB, p=4)
    key.go            # APIKey, KeyPermission, PermissionMode constants,
                      #  GenerateKey(prefix) -> "la-<64 hex>"
pkg/database/
    apikeys.go        # CreateKey, ListKeys, GetKey, DeleteKey, TouchKey,
    permissions.go    #   GetActiveKeys, HasPermission, Grant, Revoke, List
pkg/server/
    handlers_auth.go  # API key + permission CRUD handlers
    middleware.go     # APIAuthMiddleware: InferenceAuthMiddleware
    routes.go         # mount /api/v1/auth/keys/* ; wrap /v1/* with middleware
```

## API contract

```
POST   /api/v1/auth/keys            body: {name, permission_mode, expires_at?}
                                   -> {id, key, name, permission_mode, created_at}
GET    /api/v1/auth/keys            -> [{id, name, permission_mode, expires_at, created_at, last_used_at}]
GET    /api/v1/auth/keys/{id}       -> key detail (no plaintext)
DELETE /api/v1/auth/keys/{id}       -> 204
GET    /api/v1/auth/keys/{id}/permissions        -> [{instance_id, instance_name}]
POST   /api/v1/auth/keys/{id}/permissions        body: {instance_id} -> 201
DELETE /api/v1/auth/keys/{id}/permissions/{iid}  -> 204
```

## Data model

Uses the `api_keys` and `key_permissions` tables created in Phase 1.

- `key_hash` stores the Argon2id encoded hash (`$argon2id$v=19$m=...$salt$hash`).
- `user_id` is NULL for now (no OAuth users until Phase 5); once sessions
  exist, the creating user is recorded.
- `permission_mode`: `allow_all` or `per_instance`.

## Middleware behavior

`InferenceAuthMiddleware` (applied to `/v1/*` only):

1. `OPTIONS` → pass through (CORS preflight).
2. Extract key from `Authorization: Bearer ...` → else `X-API-Key` →
   else `?api_key=` → else 401 `missing_api_key`.
3. Load `GetActiveKeys` (non-expired). For each, `auth.VerifyKey(provided, hash)`.
   On match, set the `*auth.APIKey` in the request context and asynchronously
   `TouchKey(id)`.
4. If none match → 401 `invalid_api_key`.
5. In `OpenAIProxy` (Phase 3) after resolving the instance, call
   `authMiddleware.CheckInstancePermission(ctx, inst.ID)`:
   - No APIKey in context → allow (would only happen if management key auth
     were wired, which Phase 5 adds; for now this branch is unused).
   - `allow_all` → allow.
   - `per_instance` → require a row in `key_permissions`; else 403
     `permission_denied`.
6. Auth failures return OpenAI-shaped errors so SDKs parse them cleanly:

```json
{"error":{"message":"Missing API key","type":"authentication_error"}}
```

## Implementation steps

1. `pkg/auth/hash.go`: Argon2id with the params above; constant-time verify.
2. `pkg/auth/key.go`: `GenerateKey("la")` → `crypto/rand` 32 bytes → hex.
3. `pkg/database/apikeys.go`: CRUD; `GetActiveKeys` filters `expires_at IS NULL
   OR expires_at > now`; `TouchKey` updates `last_used_at` (best-effort,
   errors only logged).
4. `pkg/database/permissions.go`: grant/revoke/list; `HasPermission(keyID, instID)`.
5. `pkg/server/handlers_auth.go`: key CRUD (always hash on create, return
   plaintext once in the `key` field, never persist it); permission handlers
   resolve `instance_id` → name for the list response.
6. `pkg/server/middleware.go`: `APIAuthMiddleware` with `authStore` +
   `requireInferenceAuth` flag from config; `InferenceAuthMiddleware()`;
   `CheckInstancePermission(ctx, instanceID)`.
7. `pkg/server/routes.go`: mount `/api/v1/auth/keys` routes (still under
   the open `/api/v1` group); conditionally wrap `/v1` with the inference
   middleware when `cfg.Auth.RequireInferenceAuth` (default true).
8. Manual test: create a key, use it against `/v1/chat/completions`,
   confirm 401 without it and 403 for a non-granted instance.

## Notes

- The plaintext key is shown **exactly once** at creation; the DB never
  stores it. Document this in the CLI output in Phase 6.
- Argon2id is intentionally heavier than SHA; that is fine because there are
  few keys and `GetActiveKeys` is cached per request. If key counts grow
  large, a key-id prefix + lookup-by-id can be added later (the `la-<hex>`
  format has room for an embedded id if we split it as `la-<id>-<secret>`).
  Out of scope for now.
- Session tokens (Phase 5) use the same `Authorization: Bearer` header on
  `/api/v1/*`. The two middlewares are scoped to disjoint route trees, so
  there is no ambiguity.
