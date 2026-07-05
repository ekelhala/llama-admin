---
title: "Configuration reference"
weight: 20
lead: "YAML file, env vars and overrides."
---

llama-admin loads its configuration from a YAML file (default `config.yaml`
in the working directory; override with `LLAMA_ADMIN_CONFIG_PATH`) and
overlays environment variables on top.

A fully commented reference file ships at
[`config.example.yaml`](https://github.com/ekelhala/llama-admin/blob/main/config.example.yaml).
Copy it to `config.yaml` and edit to taste:

```sh
cp config.example.yaml config.yaml
```

## Precedence

Values are resolved in this order (later wins):

1. Built-in defaults (see `pkg/config/defaults.go`)
2. The YAML config file
3. Environment variables prefixed with `LLAMA_ADMIN_`
4. Path placeholders of the form `${VAR}` or `${VAR:-default}` are expanded
   against the process environment in path fields (see
   `pkg/config/expand.go`).

## Sections

### `server`

| Field           | Type     | Default     | Description                                    |
|-----------------|----------|-------------|------------------------------------------------|
| `host`          | string   | `127.0.0.1` | Bind address. Use `0.0.0.0` for all interfaces.|
| `port`          | int      | `8080`      | Listen port.                                    |
| `allowedOrigins`| []string | `["*"]`     | CORS allowed origins.                          |
| `enableSwagger` | bool     | `false`     | Expose Swagger UI at `/swagger`.               |

Env: `LLAMA_ADMIN_SERVER_HOST`, `LLAMA_ADMIN_SERVER_PORT`,
`LLAMA_ADMIN_SERVER_ALLOWED_ORIGINS` (comma-separated),
`LLAMA_ADMIN_SERVER_ENABLE_SWAGGER`.

### `backends.llamaCpp`

| Field             | Type     | Default | Description                                              |
|-------------------|----------|---------|----------------------------------------------------------|
| `binaryPath`      | string   | `""`    | Path to `llama-server`. Empty → look up on `PATH`.       |
| `cacheDir`        | string   | `""`    | Where downloaded HuggingFace models are cached.          |
| `downloadTimeout` | duration | `10m`   | Per-download timeout for HuggingFace model downloads.    |

Env: `LLAMA_ADMIN_BACKEND_LLAMACPP_BINARY_PATH`,
`LLAMA_ADMIN_BACKEND_LLAMACPP_CACHE_DIR`,
`LLAMA_ADMIN_BACKEND_LLAMACPP_DOWNLOAD_TIMEOUT`.

### `instances`

| Field                   | Type             | Default      | Description                                              |
|-------------------------|------------------|--------------|----------------------------------------------------------|
| `portRange.min`/`.max`  | int              | `8100`/`9000`| Port range allocated to spawned `llama-server` instances.|
| `onDemandStartTimeout`  | duration         | `30s`        | Time to wait for an on-demand instance to become ready.  |
| `logRotationEnabled`    | bool             | `true`       | Enable per-instance log rotation.                        |
| `logRotationMaxSize`    | int (bytes)      | `52428800`   | Max log file size before rotation (50 MiB).              |
| `logRotationCompress`   | bool             | `true`       | Compress rotated logs.                                   |
| `timeoutCheckInterval`  | duration         | `10s`        | How often to scan running instances for idle timeouts.   |
| `enableLRUEviction`     | bool             | `false`      | Evict least-recently-used idle instances at capacity.    |
| `maxRunningInstances`   | int              | `10`         | Hard cap on concurrent running instances.                |
| `groupLimits`           | map[string]int   | `{}`         | Optional per-group concurrency caps.                    |

Env (prefixed `LLAMA_ADMIN_INSTANCES_`): `PORT_RANGE_MIN`, `PORT_RANGE_MAX`,
`ON_DEMAND_START_TIMEOUT`, `LOG_ROTATION_ENABLED`, `LOG_ROTATION_MAX_SIZE`,
`LOG_ROTATION_COMPRESS`, `TIMEOUT_CHECK_INTERVAL`, `ENABLE_LRU_EVICTION`,
`MAX_RUNNING_INSTANCES`.

### `database`

| Field | Type   | Default | Description                                                          |
|-------|--------|---------|----------------------------------------------------------------------|
| `path`| string | `""`    | SQLite database path. Defaults to `<dataDir>/data/llama-admin.db`. |

Env: `LLAMA_ADMIN_DATABASE_PATH`.

### `auth`

| Field                | Type            | Default | Description                                            |
|----------------------|-----------------|---------|--------------------------------------------------------|
| `session.ttl`        | duration        | `24h`   | Time-to-live for OAuth-issued management session tokens.|
| `providers`          | map             | `{}`    | OAuth providers. Key is the provider name (`github`).  |
| `providers.<name>.enabled`           | bool     | `false` | Activate the provider.                                 |
| `providers.<name>.clientId`           | string   | `""`    | OAuth App client ID.                                  |
| `providers.<name>.clientSecret`       | string   | `""`    | OAuth App client secret.                              |
| `providers.<name>.scopes`             | []string | `[]`    | OAuth scopes to request.                              |
| `providers.<name>.deviceAuthorizationEndpoint` | string | provider default | Override for GitHub Enterprise Server.     |
| `providers.<name>.tokenEndpoint`              | string | provider default | Override for GitHub Enterprise Server.     |
| `providers.<name>.userEndpoint`               | string | provider default | Override for GitHub Enterprise Server.     |
| `allowedEmails`      | []string        | `[]`    | Email allowlist seeded at startup. Only verified matching emails may log in. Extendable at runtime via the API. |

Env: `LLAMA_ADMIN_AUTH_SESSION_TTL`, `LLAMA_ADMIN_AUTH_ALLOWED_EMAILS`
(comma-separated).

### Top-level

| Field     | Type   | Default | Description                                                                 |
|-----------|--------|---------|-----------------------------------------------------------------------------|
| `dataDir` | string | `${XDG_CONFIG_HOME}/llama-admin` or `~/.config/llama-admin` | Where the database, instance logs and model cache live. |

Env: `LLAMA_ADMIN_DATA_DIR`.

## GitHub OAuth setup

For the GitHub provider, create an OAuth App at
<https://github.com/settings/developers>. Device flow does not require a
redirect URL. Set the client id/secret in the `providers.github` section
(or via `LLAMA_ADMIN_GITHUB_CLIENT_ID` / `LLAMA_ADMIN_GITHUB_CLIENT_SECRET`)
and `enabled: true`.

For GitHub Enterprise Server, set the `deviceAuthorizationEndpoint`,
`tokenEndpoint` and `userEndpoint` fields to the enterprise equivalents.
