# Refactor — Model & instance simplification

## Goal

Reduce the model/instance system to its essentials:

- A **model** is defined by exactly two things: an **alias** (user-chosen at
  registration time) and a **filename** on disk.
- An **instance** references a model by alias. At start time the alias is
  resolved to the filename and passed to `llama-server` as `--model=<path>`.
- All other instance parameters are **user-supplied key/value pairs** that are
  sanitized and forwarded to `llama-server` verbatim. No typed option structs,
  no backend abstraction layer, no fuzzy resolution.

## Current state (what's wrong)

### Three overlapping model identifiers

`ModelFileInfo` (scanner.go:10) carries **Name**, **Alias**, and **Path**:

| Field   | Meaning                                          | Source                |
|---------|--------------------------------------------------|-----------------------|
| `Name`  | relative path from cache dir (e.g. `repo/M.gguf`) | scanner              |
| `Alias` | auto-derived from filename stem                   | `assignAliases()`    |
| `Path`  | absolute filesystem path                          | scanner              |

The alias is **not user-configured** — it is synthesized by
`assignAliases`/`sanitizeAliasToken` (scanner.go:50-96) with collision
disambiguation by parent directory and numeric suffixes.

### Fuzzy resolution in two places

`ResolveModelArg` (resolve.go:21) matches an input against six candidates
per model (alias, name, bare filename, with/without extension, absolute
path), normalized via `NormalizeModelRef` (lowercase + hyphen→underscore
collapse). This runs:

1. **Client-side** — `internal/cmd/models_resolve.go` fetches the full
   catalog over HTTP and resolves the `--model` flag before POSTing.
2. **Server-side** — `manager.resolveInstanceModel` (operations.go:220)
   re-resolves just before launch to "correct" stale/wrong paths.

### Over-engineered options/backend layer

The `pkg/backends` package defines a `Backend` interface, a constructor
registry, a typed `LlamaServerOptions` struct, and a duplicate `Options`
struct — yet `instance.Start()` (instance.go:90-109) **ignores all of it**
and manually reads `BackendOptions["model"]`, `["ctx_size"]`,
`["n_gpu_layers"]` from the `map[string]any` with type assertions.

`instance.Options` (options.go:10) carries seven fields
(`BackendType`, `BackendOptions`, `DockerEnabled`, `CommandOverride`,
`Environment`, `Nodes`, `AutoRestart`, `PresetIni`) of which only
`BackendOptions` and `Environment` are actually used in `Start()`.

### Summary of call sites to change

| Symbol                              | Files                                                                 |
|-------------------------------------|-----------------------------------------------------------------------|
| `ResolveModelArg` / `NormalizeModelRef` | resolve.go, models_resolve.go, operations.go, resolve_test.go       |
| `assignAliases` / `sanitizeAliasToken`  | scanner.go, scanner_test.go                                          |
| `ModelFileInfo` (Name/Alias/Path)       | scanner.go, manager.go, resolve.go, handlers_models.go, models_resolve.go |
| `backends.*` (entire package)           | instance/options.go, instance/instance.go, handlers_openai.go, operations.go |
| `resolveInstanceModel`                  | operations.go, manager.go                                            |
| `resolveModelArg` (CLI)                 | internal/cmd/instances.go, models_resolve.go                         |

---

## Target state

### Model (DB-backed)

```go
type Model struct {
    ID        int64  `json:"id"`
    Alias     string `json:"alias"`      // user-chosen, unique
    Filename  string `json:"filename"`   // absolute path on disk
    SizeBytes int64  `json:"size_bytes"`
    CreatedAt int64  `json:"created_at"`
    UpdatedAt int64  `json:"updated_at"`
}
```

- Stored in a new `models` SQLite table.
- Alias is **unique** and **case-sensitive** (no normalization, no
  case-insensitive matching).
- Filename is validated to exist and end in `.gguf` at registration time.
- The scanner is retained **only** as a disk-listing helper for discovery
  (`models files`); it no longer assigns aliases and is **not** the source of
  truth for the model catalog.

### Instance options (flat)

```go
type Options struct {
    ModelAlias  string            `json:"model_alias"`
    Params      map[string]string `json:"params"`        // llama-server flags
    Env         map[string]string `json:"env,omitempty"`  // extra env vars
    AutoRestart *bool             `json:"auto_restart,omitempty"`
}
```

- `ModelAlias` is resolved to a filename via the model store at **start time
  only**. It is stored as-is in the DB.
- `Params` is a flat `map[string]string`. Keys are llama-server flag names
  (e.g. `"ctx-size"`, `"n-gpu-layers"`). Values are raw strings. An empty
  value emits a bare flag (`--flash-attn`).
- No `BackendType` (only llama-server exists), no `DockerEnabled`, no `Nodes`,
  no `PresetIni`, no `CommandOverride`.
- No `BackendOptions` map-of-any, no type assertions, no typed
  `LlamaServerOptions` struct.

### Param sanitization

A single function in `pkg/instance/sanitize.go`:

```go
// SanitizeParams validates user-supplied llama-server parameters.
// Keys must match ^[a-z][a-z0-9-]*$ and not be in the blocked set
// (model, host, port — those are managed by llama-admin itself).
// Returns a sorted, deterministic slice of "key=value" or "key" pairs.
func SanitizeParams(params map[string]string) ([]string, error)
```

- **Allowlist vs. free-form:** free-form with a regex on keys (no allowlist of
  specific flag names). This avoids needing to track llama-server's evolving
  flag set. The regex + blocked-set is the minimum viable sanitization.
- `model`, `host`, `port` are rejected because llama-admin sets those itself.

### Instance start flow

```
StartInstance(name)
  → load instance options from DB
  → resolve opts.ModelAlias → model.Filename via ModelStore
  → args = ["--model=" + filename, "--host=" + host, "--port=" + port]
  → args += SanitizeParams(opts.Params)
  → env = os.Environ() + cfg env + opts.Env
  → process.start(binaryPath, args, env)
```

No fuzzy resolution, no re-derivation, no catalog walk. One DB lookup.

---

## API contract

### Models

```
POST   /api/v1/models              body: {alias, filename}     -> 201 model
GET    /api/v1/models                                           -> [model]
GET    /api/v1/models/{alias}                                   -> model
DELETE /api/v1/models/{alias}                                   -> 204
GET    /api/v1/models/files                                     -> [{filename, size_bytes}]
```

- `POST /models` registers an alias → filename mapping. Validates that the
  file exists and is `.gguf`.
- `GET /models/files` is the raw disk scan (replaces the old `GET /models`
  scanner output). No aliases, just filenames + sizes. Useful for deciding
  what to register.
- Download endpoints (`POST /models/download`, jobs) are unchanged.

### Instances

```
POST   /api/v1/instances/{name}   body: Options                -> instance detail
PUT    /api/v1/instances/{name}   body: Options                -> instance detail
```

Request body (new shape):
```json
{
  "model_alias": "qwen3-9b",
  "params": {"ctx-size": "8192", "n-gpu-layers": "99", "flash-attn": ""},
  "auto_restart": false
}
```

### OpenAI proxy

`/v1/*` requests use the `model` field as the **instance name** only. The
instance's configured model is already loaded by `llama-server`. No
`instance/model` splitting, no body rewriting.

---

## Database migration

`pkg/database/migrations/002_models.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS models (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    alias       TEXT NOT NULL UNIQUE,
    filename    TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_models_alias ON models(alias);
```

`002_models.down.sql`:
```sql
DROP TABLE IF EXISTS models;
```

The `instances` table is unchanged — `options_json` still holds the serialized
`Options` struct (which evolves in shape but stays a JSON blob, per the
cross-cutting decision in plans/README.md).

---

## Files to delete

| File                          | Reason                                                |
|-------------------------------|-------------------------------------------------------|
| `pkg/backends/backend.go`     | Backend abstraction unused; single backend only        |
| `pkg/backends/builder.go`    | Constructor registry unnecessary                       |
| `pkg/backends/llama.go`      | Typed options struct + BuildCommandArgs never called  |
| `pkg/backends/parser.go`     | `SplitInstanceModel`/`ValidateInstanceName` move out  |
| `pkg/models/resolve.go`      | Fuzzy resolution replaced by DB lookup                 |
| `pkg/models/resolve_test.go` | Tests for deleted code                                 |
| `internal/cmd/models_resolve.go` | Client-side resolution removed                     |

## Files to create

| File                                  | Purpose                                      |
|---------------------------------------|----------------------------------------------|
| `pkg/database/models.go`              | `ModelStore`: Save/Get/GetByAlias/List/Delete |
| `pkg/database/migrations/002_models.up.sql`   | models table                          |
| `pkg/database/migrations/002_models.down.sql` | drop models table                     |
| `pkg/instance/sanitize.go`            | `SanitizeParams`                             |
| `pkg/instance/sanitize_test.go`        | Tests for sanitization                       |

## Files to modify

### `pkg/models/scanner.go`
- Strip `Alias` field from `ModelFileInfo`. Keep `Name` (relative path),
  `Path`, `SizeBytes`, `Source`.
- Delete `assignAliases` and `sanitizeAliasToken`.
- Rename to `ScanModelFiles` to clarify it lists files, not models.

### `pkg/models/manager.go`
- Add `ModelStore` dependency.
- Add methods: `RegisterModel(alias, filename)`, `GetModel(alias)`,
  `ListModels()`, `DeleteModel(alias)`, `ResolveAlias(alias) → filename`.
- Keep download/job methods unchanged.
- `ListModels()` reads from DB, not scanner. `ListFiles()` calls the scanner.

### `pkg/instance/options.go`
- Replace the 7-field `Options` with the flat struct (ModelAlias + Params +
  Env + AutoRestart).
- Simplify `MarshalJSON`/`UnmarshalJSON` — the new struct serializes
  naturally; custom marshalers can be deleted entirely.
- Delete `ValidateAndApplyDefaults` — replace with `Validate()` that checks
  `ModelAlias != ""` and calls `SanitizeParams`.

### `pkg/instance/instance.go`
- `Start()`: resolve alias → filename via injected `ModelStore` (or a
  resolver closure), build args with `SanitizeParams`, call `process.start`.
- Delete the manual `BackendOptions["model"]` / `["ctx_size"]` /
  `["n_gpu_layers"]` type-assertion block (instance.go:90-109).

### `pkg/manager/operations.go`
- Delete `resolveInstanceModel` (the re-resolution at start time).
- `CreateInstance`/`UpdateInstance`: call `opts.Validate()` (no backend
  constructor lookup).
- `StartInstance`/`RestartInstance`: no model resolution call (moved into
  `instance.Start`).

### `pkg/manager/manager.go`
- Inject `ModelStore` (or keep `models.Manager` and use its `ResolveAlias`).
- `loadInstances`: remove the `BackendOptions["port"]` type-assertion
  (manager.go:72-78) — port is now managed by the port allocator only, not
  stored in params.

### `pkg/server/handlers_models.go`
- `ListModels` → reads from DB.
- Add `RegisterModel`, `GetModel`, `DeleteModel` handlers.
- Add `ListModelFiles` handler (raw disk scan).
- Keep download/job handlers.

### `pkg/server/handlers_instances.go`
- `CreateInstance`/`UpdateInstance`: decode new `Options` shape.

### `pkg/server/handlers_openai.go`
- Remove `backends.SplitInstanceModel` call and `instance/model` splitting.
- Remove body rewriting (lines 117-140). The `model` field = instance name;
  proxy the request body through unchanged.
- Inline a simple instance-name validation (or use `validation.IsValidInstanceName`).

### `pkg/server/routes.go`
- Add model CRUD routes: `POST /models`, `GET /models/{alias}`,
  `DELETE /models/{alias}`, `GET /models/files`.

### `internal/cmd/instances.go`
- `instancesCreateCmd`: replace `--model`/`--ctx-size`/`--gpu-layers` flags
  with `--model-alias` and `--param` (repeatable: `--param ctx-size=8192`).
- Delete the `resolveModelArg` call — the alias is sent as-is; the server
  resolves it.

### `internal/cmd/models.go`
- Add `models register <alias> <filename>` command.
- Add `models get <alias>` and `models delete <alias>` commands.
- `models list` reads from the new DB-backed endpoint.
- Add `models files` command (raw disk listing).

### `cmd/server/main.go`
- Construct `ModelStore` and pass to `models.Manager` (or inject directly).
- No other wiring changes.

### Tests to update
- `pkg/models/scanner_test.go` — remove alias/disambiguation tests; keep
  file-listing tests.
- `pkg/instance/options_test.go` — update for new `Options` shape.
- Add `pkg/instance/sanitize_test.go`.
- Add `pkg/database/models_test.go` (ModelStore round-trip).

---

## Implementation steps

Each step is independently testable and leaves the build green.

### Step 1 — Database layer
1. Write `002_models.up.sql` / `.down.sql`.
2. Write `pkg/database/models.go` (`ModelStore`: Save, GetByAlias, List, Delete).
3. Add `Model` struct to `pkg/models` (or a shared package) — the DB store and
   the manager both need it.

**Verify:** `go test ./pkg/database/...` passes; migration applies cleanly.

### Step 2 — Model manager
1. Add `ModelStore` to `models.Manager`.
2. Add `RegisterModel`, `GetModel`, `ListModels` (DB-backed), `DeleteModel`,
   `ResolveAlias`.
3. Strip aliases from `scanner.go`; rename `ScanModels` → `ScanModelFiles`.
   Delete `assignAliases` / `sanitizeAliasToken`.
4. Delete `resolve.go` and `resolve_test.go`.

**Verify:** `go test ./pkg/models/...` passes.

### Step 3 — Instance sanitize + options
1. Write `pkg/instance/sanitize.go` with `SanitizeParams`.
2. Write `pkg/instance/sanitize_test.go`.
3. Rewrite `pkg/instance/options.go` to the flat struct. Delete custom
   marshalers (use struct tags).
4. Update `options_test.go`.
5. Delete `pkg/backends/` entirely.

**Verify:** `go build ./...` — some callers will break; fix them in step 4.

### Step 4 — Instance start
1. Rewrite `instance.Start()` to: resolve alias via injected resolver,
   build args with `SanitizeParams`, spawn process.
2. Inject a `ModelResolver` func or `*database.ModelStore` into `Instance`
   (pass via `New`).

**Verify:** `go test ./pkg/instance/...` passes.

### Step 5 — Manager wiring
1. Remove `resolveInstanceModel` from `operations.go`.
2. Update `CreateInstance`/`UpdateInstance` to call `opts.Validate()`.
3. Fix `loadInstances` port restoration (remove `BackendOptions["port"]`).
4. Inject `ModelStore`/resolver into instances created by the manager.

**Verify:** `go build ./...` passes; `go test ./...` passes.

### Step 6 — Server handlers + routes
1. Add model CRUD handlers to `handlers_models.go`.
2. Add `ListModelFiles` handler.
3. Update `handlers_instances.go` for new `Options` decode.
4. Simplify `handlers_openai.go` (remove splitting/rewriting).
5. Update `routes.go` with new model routes.

**Verify:** `go build ./...` passes.

### Step 7 — CLI
1. Update `instances.go` create command (`--model-alias`, `--param`).
2. Delete `internal/cmd/models_resolve.go`.
3. Add `models register`, `models get`, `models delete`, `models files` to
   `models.go`.

**Verify:** `go build ./...` passes; manual `llama-admin models register`
and `llama-admin instances create` smoke test.

### Step 8 — Cleanup
1. Remove now-unused `backends` imports everywhere.
2. Run `go vet ./...` and `go test ./...`.
3. Update `config.example.yaml` if any config keys changed (none expected).

---

## Decisions to confirm

1. **Param values for boolean flags:** propose empty string = bare flag
   (`"flash-attn": ""` → `--flash-attn`). Alternative: `"true"`/`"false"`.
2. **Model filename storage:** absolute path (simple, current scanner
   behavior) vs. relative-to-cacheDir (portable). Propose absolute for
   simplicity.
3. **OpenAI proxy `model` field:** instance name only (no `/` splitting).
   The request body is proxied through unchanged. llama-server's own model
   field in responses will reflect its loaded model, which may differ from
   the request — acceptable since each instance serves exactly one model.
4. **Scanner retention:** keep as `GET /models/files` for disk discovery,
   but it is no longer the source of truth for the model catalog.
