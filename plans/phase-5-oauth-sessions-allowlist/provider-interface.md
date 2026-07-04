# Phase 5 — Provider interface and reference implementation

This file specifies the OAuth `Provider` abstraction and the GitHub
reference implementation. See `plan.md` for the overall flow.

## Provider interface

```go
// pkg/auth/provider.go

type Provider interface {
    // Name is the stable identifier used in URL paths and the users table
    // (e.g. "github"). Must be lowercase, match [a-z0-9-]+.
    Name() string

    // SupportsDeviceFlow reports whether the provider implements Device
    // Flow (RFC 8628). llama-admin's login only uses Device Flow; a provider
    // returning false is listed by /auth/providers but its /device and
    // /token endpoints return 501.
    SupportsDeviceFlow() bool

    // InitiateDeviceFlow requests a device code from the provider. The
    // returned DeviceCode is forwarded to the CLI verbatim.
    InitiateDeviceFlow(ctx context.Context) (*DeviceCode, error)

    // ExchangeDeviceCode polls the provider's token endpoint with the
    // device_code. Implementations should return:
    //   - (accessToken, nil) on success
    //   - (ErrAuthorizationPending) while the user has not yet authorized
    //     (the server maps this to HTTP 408 so the CLI retries)
    //   - (ErrAuthorizationSlowDown) if the provider says to slow down
    //   - any other error is surfaced as-is
    ExchangeDeviceCode(ctx context.Context, deviceCode string) (string, error)

    // FetchUserInfo calls the provider's user info endpoint(s) and returns
    // a normalized UserInfo. VerifiedEmails MUST contain only emails the
    // provider has marked verified; the allowlist check depends on this.
    FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

type DeviceCode struct {
    DeviceCode      string `json:"device_code"`
    UserCode        string `json:"user_code"`
    VerificationURI string `json:"verification_uri"`
    ExpiresIn       int    `json:"expires_in"`
    Interval        int    `json:"interval"`
}

type UserInfo struct {
    Provider        string   `json:"provider"`        // == Provider.Name()
    ProviderUserID  string   `json:"provider_user_id"` // stringified ID
    Username        string   `json:"username"`
    Email           string   `json:"email"`            // primary email for display
    AvatarURL       string   `json:"avatar_url"`
    VerifiedEmails  []string `json:"verified_emails"`  // for allowlist check
}

var (
    ErrAuthorizationPending  = errors.New("authorization pending")
    ErrAuthorizationSlowDown = errors.New("authorization slow down")
)
```

## Registry

```go
type ProviderRegistry struct {
    providers map[string]Provider
}

func NewRegistry() *ProviderRegistry { ... }
func (r *ProviderRegistry) Register(p Provider) error  // reject duplicate name
func (r *ProviderRegistry) Get(name string) (Provider, bool)
func (r *ProviderRegistry) Names() []string
```

Construction in `cmd/server/main.go`:
```go
reg := auth.NewRegistry()
if cfg.Auth.Providers["github"].Enabled {
    reg.Register(github.New(cfg.Auth.Providers["github"]))
}
```
An unknown provider name in config is a startup error (so typos fail loudly).

## GitHub reference implementation

`pkg/auth/github/github.go`:

- Config: `ClientID`, `ClientSecret`, `Scopes` (default `["read:user"]`),
  and optional endpoint overrides (defaults shown):
  - `DeviceAuthorizationEndpoint` = `https://github.com/login/device/code`
  - `TokenEndpoint`               = `https://github.com/login/oauth/access_token`
  - `UserEndpoint`                = `https://api.github.com/user`
  - `UserEmailsEndpoint`          = `https://api.github.com/user/emails`
- HTTP client: a shared `*http.Client` with a 10s timeout.
- `Name()` returns `"github"`; `SupportsDeviceFlow()` returns `true`.
- `InitiateDeviceFlow`: `POST` form `{client_id, scope}` to the device
  endpoint with `Accept: application/json`. Map the response to `DeviceCode`.
  GitHub's response uses `user_code`, `device_code`, `verification_uri`,
  `expires_in`, `interval` — direct field mapping.
- `ExchangeDeviceCode`: `POST` form
  `{client_id, device_code, grant_type=urn:ietf:params:oauth:grant-type:device_code}`
  to the token endpoint with `Accept: application/json`. Parse:
  - `error == "authorization_pending"` → `ErrAuthorizationPending`
  - `error == "slow_down"`             → `ErrAuthorizationSlowDown`
  - `error == "expired_token"`         → a typed error the server maps to 410
  - `error == "access_denied"`         → a typed error the server maps to 403
  - else `access_token` on success.
  Note: GitHub Device Flow does not require `client_secret` for the token
  exchange, but we send it when configured for compatibility with other
  providers' conventions.
- `FetchUserInfo`: GET `/user` with `Authorization: Bearer <token>`. Map
  `id` (stringify), `login` (username), `avatar_url`. Then GET `/user/emails`
  and filter `verified == true`, collecting the `email` fields into
  `VerifiedEmails`. Set `Email` to the primary verified email if any,
  else the first verified email. If `/user/emails` returns 403 (missing
  `user:email` scope), `VerifiedEmails` is empty and login fails the
  allowlist check — surface this as a clear error rather than a generic 403.

## Adding a new provider

To add a provider (e.g. GitLab), implement `pkg/auth/<name>/<name>.go`
exposing `New(cfg auth.ProviderConfig) Provider`, then register it in
`cmd/server/main.go` when `cfg.Auth.Providers["<name>"].Enabled`. The HTTP
handlers, session issuance, allowlist, and CLI all work unchanged — the
provider only affects the device flow + user info normalization.

A later, fully-config-driven `generic` provider (endpoints + JSONPath field
mappings in YAML) is possible but deferred: each provider package keeps the
user-info parsing type-safe and tested.

## Error mapping (server side)

| Provider error                | HTTP status |
|-------------------------------|-------------|
| `ErrAuthorizationPending`     | 408         |
| `ErrAuthorizationSlowDown`    | 429         |
| `expired_token`               | 410 Gone    |
| `access_denied`               | 403         |
| provider HTTP 401/403         | 502 (the server's own token is bad) |
| any other                     | 500         |
