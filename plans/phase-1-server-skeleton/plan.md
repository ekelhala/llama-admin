# Phase 1 — Server skeleton, config, and database

## Goal

Stand up the Go module, configuration loader, SQLite database with migration
runner, and a chi HTTP server skeleton with a single `GET /api/v1/version`
endpoint. Everything later phases build on is established here.

## Exit criteria

- `go run ./cmd/server` starts and prints a startup banner.
- `GET /api/v1/version` returns JSON with `version`, `commit`, `build_time`.
- `GET /healthz` returns `200 OK` (liveness).
- On first run, an SQLite database file is created and `001_initial_schema`
  is applied. Re-running the server does not re-apply it.
- `SIGTERM`/`SIGINT` triggers a graceful shutdown with a 30s deadline that
  closes the database.

## Files to create

```
go.mod
cmd/server/main.go
pkg/config/
    types.go         # all config structs (server, backends, instances,
                    #  database, auth — including providers, session,
                    #  allowed_emails)
    config.go        # LoadConfig(path) (*AppConfig, error)
    defaults.go      # platform-aware defaults (data_dir, logs_dir, db path)
    dotenv.go        # load .env into environment
    env.go           # LLAMA_ADMIN_* env var overrides
    expand.go        # ${VAR} and ${VAR:-default} placeholder expansion
    expand_test.go
pkg/database/
    database.go      # Open(cfg), Close(), wraps *sql.DB
    migrations.go    # RunMigrations(db) using embedded .sql files
    migrations/
        001_initial_schema.up.sql
        001_initial_schema.down.sql
pkg/server/
    routes.go        # SetupRouter(handler) *chi.Mux (CORS, /healthz,
                    #  /api/v1/version)
    handlers_system.go   # VersionHandler, HealthHandler
    middleware.go    # contextKey type + placeholder helpers
    handlers.go      # Handler struct + NewHandler + writeJSON/writeError
```

## Config type shape (defined now, used by later phases)

```go
type AppConfig struct {
    Server    ServerConfig
    Backends  BackendConfig      // only llama_cpp populated for now
    Instances InstancesConfig
    Database  DatabaseConfig
    Auth      AuthConfig         // providers + session + allowed_emails
    DataDir   string
    Version, Commit, BuildTime string
}

type AuthConfig struct {
    Session       SessionConfig
    Providers     map[string]ProviderConfig
    AllowedEmails []string       // config-seed allowlist
}

type SessionConfig struct {
    TTL time.Duration            // default 24h
}

type ProviderConfig struct {
    Enabled                  bool
    ClientID                 string
    ClientSecret             string
    Scopes                   []string
    DeviceAuthorizationEndpoint string  // optional override
    TokenEndpoint            string         // optional override
    UserEndpoint             string         // optional override
}
```

Defining the full `AuthConfig` now avoids a schema/config migration in
Phase 5 — the fields are just unused until then.

## Initial schema (`001_initial_schema.up.sql`)

Minimal — only what Phase 1 needs plus cheap-to-add-now tables that later
phases expect, to keep migrations stable:

```sql
-- instances: created empty in Phase 1, populated from Phase 2
CREATE TABLE IF NOT EXISTS instances (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'stopped'
                CHECK(status IN ('stopped','running','failed',
                                 'restarting','shutting_down')),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    options_json TEXT,
    owner_user_id INTEGER
);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);

-- api_keys: used from Phase 4
CREATE TABLE IF NOT EXISTS api_keys (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key_hash        TEXT NOT NULL,
    name            TEXT NOT NULL,
    user_id         INTEGER,
    permission_mode TEXT NOT NULL DEFAULT 'per_instance'
                    CHECK(permission_mode IN ('allow_all','per_instance')),
    expires_at      INTEGER,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    last_used_at    INTEGER
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);

-- key_permissions: per-instance scoping (Phase 4)
CREATE TABLE IF NOT EXISTS key_permissions (
    key_id      INTEGER NOT NULL,
    instance_id INTEGER NOT NULL,
    PRIMARY KEY (key_id, instance_id),
    FOREIGN KEY (key_id)      REFERENCES api_keys(id)   ON DELETE CASCADE,
    FOREIGN KEY (instance_id) REFERENCES instances(id)  ON DELETE CASCADE
);

-- users + sessions + allowed_emails: used from Phase 5
CREATE TABLE IF NOT EXISTS users (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    username         TEXT,
    email            TEXT,
    avatar_url       TEXT,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL,
    UNIQUE(provider, provider_user_id)
);

CREATE TABLE IF NOT EXISTS sessions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash   TEXT NOT NULL UNIQUE,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    expires_at    INTEGER NOT NULL,
    created_at    INTEGER NOT NULL,
    last_used_at  INTEGER
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

CREATE TABLE IF NOT EXISTS allowed_emails (
    email          TEXT PRIMARY KEY,
    added_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at     INTEGER NOT NULL
);
```

The down migration drops all six tables in reverse-dependency order.

## Implementation steps

1. `go mod init llama-admin`; add chi v5, chi/cors, go-sqlite3, golang-migrate,
   yaml.v3, golang.org/x/crypto (for Argon2id, used in Phase 4 but cheap to
   pin now).
2. `pkg/config/types.go`: define all structs above.
3. `pkg/config/defaults.go`: platform-dependent paths via
   `os.UserConfigDir`/`os.UserCacheDir` analog (e.g. `~/.local/share/llama-admin`).
4. `pkg/config/{dotenv,env,expand}.go`: `.env` auto-load, `LLAMA_ADMIN_*`
   overrides, `${VAR:-default}` expansion in YAML strings.
5. `pkg/config/config.go`: load YAML at env-provided path (default
   `./config.yaml`), apply defaults, apply env overrides, return `*AppConfig`.
6. `pkg/database/database.go`: `Open(cfg)` opens SQLite with
   `_busy_timeout=5000&_journal_mode=WAL`, sets connection pool limits.
7. `pkg/database/migrations.go`: embed `migrations/*.sql` via `//go:embed`,
   run with `golang-migrate`'s `embed` source.
8. `pkg/server/handlers.go`: `Handler` struct holding cfg + db (and later
   managers); `writeJSON`, `writeError` helpers.
9. `pkg/server/handlers_system.go`: `VersionHandler` reads `cfg.Version` etc.;
   `HealthHandler` returns `200`.
10. `pkg/server/routes.go`: chi router with `middleware.Logger`, CORS using
    `cfg.Server.AllowedOrigins`, `r.Get("/healthz", ...)`, and
    `r.Route("/api/v1", ...)` mounting `/version`. Leave an empty `r.Route("/v1", ...)`
    block as a placeholder with a `TODO` comment — Phase 3 fills it in.
11. `cmd/server/main.go`: parse `--version`, load config, create data dir,
    open + migrate DB, build handler + router, start `http.Server`,
    wait on SIGINT/SIGTERM, `server.Shutdown(30s)`, `db.Close()`.

## Notes

- Pin `go 1.24` in `go.mod`.
- Keep `main.go` small — wire dependencies only; no business logic.
- Tests for `expand.go` only this phase (the trickiest pure logic); HTTP
  and DB layers get tests in later phases when they have behavior worth
  exercising.
- Do **not** add management-auth middleware yet — Phase 5 does. `/api/v1/*`
  is open in Phases 1–4.
