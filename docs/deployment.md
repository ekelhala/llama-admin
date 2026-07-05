# Deployment

llama-admin is designed to run as a systemd service on a Linux host that has
access to a `llama-server` binary and your model files.

## One-line install

The rolling `latest` release on GitHub ships an installer that provisions
everything (system user, binary, config, data directory and systemd unit):

```sh
curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-server.sh | sh
```

By default it installs:

- Binary: `/usr/local/bin/llama-admin-server`
- Config: `/etc/llama-admin/config.yaml`
- Data dir: `/var/lib/llama-admin`
- Systemd unit: `/etc/systemd/system/llama-admin.service`

Override these with `PREFIX`, `SYSCONF`, `UNITDIR` and `DATA_DIR`:

```sh
curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-server.sh \
  | PREFIX=/opt/llama-admin DATA_DIR=/data/llama-admin sh
```

The installer enables but does not start the service — edit the config first
(see [configuration.md](configuration.md)):

```sh
sudo systemctl start llama-admin
journalctl -u llama-admin -f
```

## Manual install

If you prefer to build from source:

```sh
git clone https://github.com/ekelhala/llama-admin.git
cd llama-admin
make build               # produces bin/llama-admin-server and bin/llama-admin
sudo make install-systemd  # installs binaries, config and unit
```

`make install-systemd` honours `PREFIX`, `SYSCONF`, `UNITDIR`, `DATA_DIR`
and `DESTDIR` (for packaging).

## systemd

The shipped unit (`deploy/llama-admin.service`) runs as a dedicated
`llama-admin` system user with hardening enabled:

- `NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`
- `ReadWritePaths=/var/lib/llama-admin` (the data dir)
- `ProtectKernelTunables`, `ProtectKernelModules`, `ProtectControlGroups`
- `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX`
- `CapabilityBoundingSet=` (no capabilities)
- `Restart=on-failure`

To keep OAuth secrets out of the unit file, use an `EnvironmentFile=`:

```ini
# /etc/llama-admin/env
LLAMA_ADMIN_GITHUB_CLIENT_ID=your_client_id
LLAMA_ADMIN_GITHUB_CLIENT_SECRET=your_client_secret
```

and uncomment the `EnvironmentFile=` line in the installed unit, then
`systemctl daemon-reload && systemctl restart llama-admin`.

## CLI install

To manage a llama-admin server from your workstation, install just the CLI:

```sh
curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-cli.sh | sh
```

This puts `llama-admin` in `$PREFIX/bin` (default `/usr/local/bin`) and
creates nothing else. Point it at your server:

```sh
llama-admin config set-server https://your-host:8080
llama-admin auth login
```

## Upgrading

Re-running the server installer replaces the binary and the systemd unit but
leaves `/etc/llama-admin/config.yaml` and the data directory untouched:

```sh
curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-server.sh | sh
sudo systemctl restart llama-admin
```

The rolling `latest` release is rebuilt on every push to `main`; treat it as
a development build. Tagged releases will be added as the project matures.
