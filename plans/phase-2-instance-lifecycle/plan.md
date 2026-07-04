# Phase 2 — Instance lifecycle

## Goal

Spawn, supervise, and stop `llama-server` processes. Maintain a registry of
instances, allocate ports, persist state to SQLite, and expose a management
API for CRUD + lifecycle operations. No auth yet (Phase 4 gates inference,
Phase 5 gates management) — but every handler reads/writes through the
manager so wiring middleware later is trivial.

## Exit criteria

- `POST /api/v1/instances/{name}` with llama.cpp options creates an instance
  record and starts `llama-server`.
- The spawned process listens on an auto-allocated port in the configured
  range; health checks pass before the create call returns.
- `GET /api/v1/instances` lists instances; `GET /api/v1/instances/{name}`
  returns detail incl. status + port + pid.
- `POST .../start`, `.../stop`, `.../restart`, `DELETE ...` all work.
- `GET .../logs` returns the last N lines of the instance log file.
- On server restart, persisted instances are loaded; running instances with
  `auto_restart=true` come back up.
- Stopping the server (`SIGTERM`) stops all running instances.

## Files to create

```
pkg/backends/
    backend.go       # BackendType, Options (BackendOptions map), command/arg
                     #  building, docker detection, env building
    builder.go       # backendConstructors map (only llama_cpp for now)
    llama.go         # LlamaServerOptions: struct, BuildCommandArgs,
                     #  BuildDockerArgs, GetModel/Port/Host, Validate,
                     #  ParseCommand
    parser.go        # shared parsing helpers
pkg/instance/
    instance.go      # Instance struct + New/Start/Stop/Restart/WaitForHealthy
    options.go       # Options wrapper, validateAndApplyDefaults
    status.go        # Status type + state machine + onStatusChange callback
    process.go       # exec.Cmd management, health polling, pid tracking
    proxy.go         # httputil.ReverseProxy to instance host:port,
                     #  inflight counter, last-request time, shutdown guard
    logger.go        # rotating file writer (uses lumberjack-like behavior;
                     #  a thin wrapper around an existing rotation lib is fine)
    process_group_unix.go   # setpgid on Unix for clean child killing
    process_group_windows.go
pkg/manager/
    manager.go       # InstanceManager interface + impl, Shutdown, load on start
    registry.go      # in-memory map[string]*Instance, markRunning/markStopped
    ports.go         # portAllocator: allocate/free from [min,max]
    operations.go    # Create/Start/Stop/Restart/Delete/Update/List/Get/Logs
pkg/database/
    instances.go     # InstanceStore: Save, LoadAll, Delete, Get
pkg/server/
    handlers_instances.go   # all instance endpoints
```

## API contract

```
GET    /api/v1/instances                    -> [{id,name,status,created}]
POST   /api/v1/instances/{name}             body: Options  -> instance detail
GET    /api/v1/instances/{name}             -> instance detail
PUT    /api/v1/instances/{name}            body: Options  -> updated instance
DELETE /api/v1/instances/{name}             -> 204
POST   /api/v1/instances/{name}/start      -> instance detail (running)
POST   /api/v1/instances/{name}/stop       -> instance detail (stopped)
POST   /api/v1/instances/{name}/restart    -> instance detail
GET    /api/v1/instances/{name}/logs?lines=200 -> {logs: "..."}
```

Instance detail JSON:
```json
{
  "id": 1, "name": "tiny", "status": "running",
  "created": 1700000000,
  "options": {
    "backend_type": "llama_cpp",
    "backend_options": {"model": "/path/to/gguf", "port": 8000, "ctx_size": 8192}
  }
}
```

## Instance options (Phase 2 subset)

```go
type Options struct {
    BackendType     backends.BackendType
    BackendOptions  backends.BackendOptions   // unmarshaled into LlamaServerOptions
    DockerEnabled   *bool
    CommandOverride string
    Environment     map[string]string
    Nodes           map[string]struct{}       // kept for future; empty = local
    AutoRestart     *bool
    PresetIni       *string                   // router mode preset file
    // ... resource limits applied from global InstancesConfig defaults
}

type LlamaServerOptions struct {
    Model     string
    Host      string
    Port      int           // 0 = auto-allocate
    CtxSize   int
    NGpuLayers int
    // ... only the flags we wire up now; expand later
}
```

## Status state machine

```
stopped --start--> restarting --healthy--> running
running --stop----> shutting_down --process exits--> stopped
running --crash--> failed --restart--> restarting
any    --delete--> (removed)
```

`process.go` watches the child; on unexpected exit it sets `failed` and, if
`AutoRestart` and below `MaxRestarts`, schedules a restart after
`RestartDelay`.

## Port allocator

- In-memory `map[int]string` (port -> instance name).
- On load, mark every persisted instance's port as allocated.
- `allocate(name)` returns lowest free port in range; `allocateSpecific`
  for restoring persisted ports; `free(port)` on stop/delete.
- Port range comes from `cfg.Instances.PortRange` (default `[8000,9000]`).

## Persistence

`InstanceStore` serializes the full `Options` as `options_json` (matching
the API shape) so the DB row round-trips the same struct the API uses. On
startup, `manager.loadInstances()` reads all rows, reconstructs `Instance`
objects, restores their `ID`/`Created`/`Status`, allocates their persisted
port, and registers them. Auto-restart instances are started in a goroutine.

## Implementation steps

1. `pkg/backends/backend.go` + `builder.go`: `BackendType` constants,
   `backend` interface, constructor map, `Options` with custom
   (Un)MarshalJSON that dispatches on `BackendType`.
2. `pkg/backends/llama.go`: `LlamaServerOptions` with the flags we support
   now; `BuildCommandArgs` emits `--model`, `--port`, `--host`, `--ctx-size`,
   `--n-gpu-layers`. `Validate` requires `Model`.
3. `pkg/instance/status.go`: atomic status with notify callback.
4. `pkg/instance/process.go`: `start()` builds `exec.Cmd` (command + args +
   env + stdout/stderr to logger), sets process group, starts, polls
   `http://host:port/health` until 200 or timeout. `stop()` sends SIGTERM,
   waits, escalates to SIGKILL after a grace period; kills the process
   group on Unix.
5. `pkg/instance/logger.go`: open `logs_dir/<name>.log`, write lines;
   rotation is a Phase 8 concern but the interface accommodates it.
6. `pkg/instance/proxy.go`: `httputil.ReverseProxy` with `Director` rewriting
   the target to `host:port`; an `inflight` atomic counter incremented in
   `serveHTTP` and decremented on return; `lastRequestTime` updated each
   request; reject new requests when status is `shutting_down`.
7. `pkg/instance/instance.go`: `New` validates options, creates status +
   options + proxy + logger + process; `Start/Stop/Restart/WaitForHealthy`
   delegate to `process`; `MarshalJSON`/`UnmarshalJSON` for persistence.
8. `pkg/manager/ports.go`, `registry.go`, `operations.go`, `manager.go`:
   per-instance mutex map for safe concurrent ops; `Shutdown` stops all
   running instances concurrently and waits.
9. `pkg/database/instances.go`: `Save` upserts by name; `LoadAll` returns
   all rows; `Delete`; `Get`.
10. `pkg/server/handlers_instances.go`: handlers using the
    `InstanceManager` interface; consistent error envelope via `writeError`.
11. `pkg/server/routes.go`: mount the instance routes under `/api/v1/instances`.
12. `cmd/server/main.go`: construct `manager.New(cfg, db)` and pass to
    `NewHandler`; on shutdown call `manager.Shutdown()`.

## Notes

- Per-instance locking via `sync.Map` of `*sync.Mutex` (mirrors llamactl)
  lets two different instances start concurrently while serializing ops on
  the same one.
- `WaitForHealthy` poll interval 200ms, default timeout from
  `cfg.Instances.OnDemandStartTimeout` (default 120s).
- Do not implement idle timeout / LRU eviction here — that is Phase 8.
  Leave `ShouldTimeout`/`GetInflightRequests` accessible (proxy already
  tracks them) so Phase 8 can layer on top without changes.
