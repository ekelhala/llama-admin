# Phase 3 — OpenAI-compatible proxy

## Goal

Expose `/v1/*` endpoints that look like the OpenAI API to external clients.
Requests are routed to a running `llama-server` instance based on the
`model` field in the request body, with on-demand start and full streaming
(SSE) support.

## Exit criteria

- `GET /v1/models` returns instances in the OpenAI list format.
- `POST /v1/chat/completions` (and `/v1/completions`, `/v1/embeddings`) with
  `{"model": "<instance>"}` proxies to that instance and returns its
  response verbatim, including streamed `text/event-stream` responses.
- `{"model": "<instance>/<model>"}` sets the inner `model` to `<model>`
  before proxying (used by llama.cpp router mode).
- A request to a stopped instance with on-demand start enabled waits for
  it to become healthy, then proxies; otherwise it returns 503.
- A request to an instance in `shutting_down` returns 503 immediately.

## Files to create / modify

```
pkg/server/
    handlers_openai.go     # OpenAIListInstances, OpenAIProxy
pkg/server/routes.go      # mount /v1/* (no auth yet — Phase 4 adds it)
```

## Endpoints

```
GET  /v1/models                    -> OpenAIListInstancesResponse
POST /v1/chat/completions
POST /v1/completions
POST /v1/embeddings
POST /v1/rerank
POST /v1/reranking
POST /v1/*                         -> OpenAIProxy (catch-all)
```

The catch-all `POST /v1/*` handles every path under `/v1` so llama.cpp
quirks (e.g. `/v1/messages`) work without per-route additions.

## Response shapes

`/v1/models`:
```json
{
  "object": "list",
  "data": [
    {"id": "tiny", "object": "model", "created": 1700000000, "owned_by": "llama-admin"}
  ]
}
```

For llama.cpp router-mode instances (Phase 7 adds model listing; Phase 3
just lists the instance name as a single entry — a TODO notes the
`<instance>/<model>` enumeration that Phase 7 will fill in).

## Routing logic

`OpenAIProxy` (in `handlers_openai.go`):

1. Read the entire body into bytes (we must inspect it before proxying).
2. `json.Unmarshal` into `map[string]any`; require non-empty `model`.
3. Split on first `/`: `instanceName = before`, `modelName = after` (if any).
   If no `/`, `instanceName = model`.
4. `validation.ValidateInstanceName(instanceName)` (reject path traversal
   and odd characters).
5. `manager.GetInstance(instanceName)`; 400/404 on not found.
6. If `status == shutting_down` → 503 `instance_shutting_down`.
7. If not running:
   - If on-demand start enabled → `ensureInstanceRunning(inst)`
     (waits for healthy, returns 500 on timeout).
   - Else → 503 `instance_not_running`.
8. Resolve the final inner `model`: if the request had no `/`, set it to
   the instance's configured backend `model` (so llama.cpp gets a real
   model id). If it had `/`, keep the suffix.
9. Re-marshal the body with the updated `model`, replace `r.Body` +
   `r.ContentLength`.
10. `inst.ServeHTTP(w, r)` — delegates to `instance.proxy.serveHTTP`,
    which uses `httputil.ReverseProxy` and tracks inflight + last-request.

## Streaming support

`httputil.ReverseProxy` already streams the response body. The critical
pieces:

- Do **not** buffer the response in the handler; let the reverse proxy
  flush through.
- Ensure `http.Server` is not wrapped in anything that disables flush.
- `proxy.go` (from Phase 2) sets `FlushInterval: -1` on the reverse proxy
  so SSE chunks flush immediately to the client.
- Disable response compression on the proxied path (or let the upstream
  decide); do not wrap `/v1/*` in a compressing middleware.

A manual test against `llama-server` with `stream: true` confirms chunks
arrive progressively.

## Implementation steps

1. `pkg/server/handlers_openai.go`: `OpenAIListInstancesResponse` +
   `OpenAIInstance` structs; `OpenAIListInstances` lists all instances and
   maps each to one entry (router-mode enumeration deferred to Phase 7).
2. `OpenAIProxy` as described above; reuses
   `instance.ServeHTTP` from Phase 2.
3. `pkg/server/routes.go`: add `r.Route("/v1", func(r chi.Router) {
   r.Get("/models", ...); r.Post("/*", ...) })` — no middleware in this phase.
4. Manual test: with a tiny GGUF and a running instance, run

       curl -N localhost:8080/v1/chat/completions \
         -H 'Content-Type: application/json' \
         -d '{"model":"tiny","messages":[{"role":"user","content":"hi"}],"stream":true}'

   and confirm SSE chunks stream.

## Notes

- `ensureInstanceRunning` lives on the handler struct; it just calls
  `manager.StartInstance` then `inst.WaitForHealthy(timeout)`.
- Error envelope matches OpenAI's shape where reasonable:
  `{"error":{"message":"...","type":"..."}}`. Phase 4 reuses the same
  envelope for auth failures on `/v1/*`.
- Do not implement Anthropic-style `/v1/messages` specially; the catch-all
  proxies it. llama.cpp handles the format if configured.
