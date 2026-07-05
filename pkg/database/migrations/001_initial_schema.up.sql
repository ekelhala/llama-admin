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
