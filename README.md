# llama-admin

Control server for managing `llama.cpp` instances and routing OpenAI-compatible
requests, administered through a CLI that authenticates users via OAuth
(GitHub by default, extensible to other providers).

llama-admin is a small Go server you deploy on your inference host. It:

- Spawns and supervises `llama-server` processes (one or more instances)
- Exposes an OpenAI-compatible API (`/v1/*`) protected by API keys, routing
  requests to instances based on the request `model` field
- Provides a management API (`/api/v1/*`) protected by OAuth-issued session
  tokens
- Ships a CLI (`llama-admin`) that authenticates the user via OAuth device
  flow and manages instances, models, and API keys

## Install

### Server (on the inference host)

```sh
curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-server.sh | sh
```

This provisions a `llama-admin` system user, installs the binary into
`/usr/local/bin`, drops the config at `/etc/llama-admin/config.yaml`, creates
`/var/lib/llama-admin`, and enables (but does not start) the systemd service.
Edit the config, then:

```sh
sudo systemctl start llama-admin
journalctl -u llama-admin -f
```

### CLI (on your workstation)

```sh
curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-cli.sh | sh
```

Then point it at your server and log in:

```sh
llama-admin config set-server https://your-host:8080
llama-admin auth login
```

## Build from source

```sh
git clone https://github.com/ekelhala/llama-admin.git
cd llama-admin
make build
```

See the [Makefile](Makefile) for `run`, `test`, `install` and
`install-systemd` targets.

## Documentation

- [Architecture & design](docs/architecture.md)
- [Configuration reference](docs/configuration.md)
- [Deployment guide](docs/deployment.md)
- [Implementation roadmap](plans/README.md)

## License

MIT
