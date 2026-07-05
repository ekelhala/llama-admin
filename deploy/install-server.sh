#!/bin/sh
# llama-admin server installer
#
# Downloads the latest build of the llama-admin server binary from the
# rolling "latest" GitHub release and provisions the system for running it
# as a systemd service. Picks the right asset (linux/amd64 or linux/arm64)
# based on `uname -m`.
#
# The server is shipped as a cgo (glibc-linked) binary and runs on standard
# Linux hosts (workstations, Raspberry Pi 4/5, Graviton, Ampere). It will
# NOT run inside Termux on Android — for that, build from source inside
# Termux itself.
#
# Provisioned:
#   - creates a dedicated "llama-admin" system user and group
#   - installs the server binary into $PREFIX/bin
#   - installs the example config and systemd unit
#   - creates $DATA_DIR (database, instance logs, model cache)
#   - enables the llama-admin service (does not start it — edit config first)
#
# Usage:
#   curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-server.sh | sh
#
# Or, to inspect first:
#   curl -fsSL -o install-server.sh https://github.com/ekelhala/llama-admin/releases/download/latest/install-server.sh
#   sh install-server.sh
#
# Env vars (optional):
#   PREFIX       install prefix for the binary         (default: /usr/local)
#   SYSCONF      config file directory                (default: /etc)
#   UNITDIR      systemd unit directory               (default: /etc/systemd/system)
#   DATA_DIR     data directory                       (default: /var/lib/llama-admin)
#   RELEASE_TAG  which release tag to pull from       (default: latest)
#
set -eu

OWNER=ekelhala
REPO=llama-admin
USER_NAME=llama-admin
GROUP_NAME=llama-admin

PREFIX="${PREFIX:-/usr/local}"
SYSCONF="${SYSCONF:-/etc}"
UNITDIR="${UNITDIR:-/etc/systemd/system}"
DATA_DIR="${DATA_DIR:-/var/lib/llama-admin}"
RELEASE_TAG="${RELEASE_TAG:-latest}"

DOWNLOAD_BASE="https://github.com/${OWNER}/${REPO}/releases/download/${RELEASE_TAG}"
INSTALLER_URL="${DOWNLOAD_BASE}/install-server.sh"

bail() {
    echo "error: $*" >&2
    exit 1
}

log() {
    echo "==> $*"
}

# --- platform check -------------------------------------------------------

[ "$(uname -s)" = "Linux" ] || bail "this installer only supports Linux."

arch="$(uname -m)"
case "$arch" in
    x86_64|amd64)   asset_arch=amd64 ;;
    aarch64|arm64)  asset_arch=arm64 ;;
    *) bail "no prebuilt binary for architecture '$arch'; only linux/amd64 and linux/arm64 are published. Build from source instead: make build"
esac

# --- root / sudo ----------------------------------------------------------

if [ "$(id -u)" -ne 0 ]; then
    # Re-exec ourselves with sudo if available. When the script is piped
    # (`curl | sh`), $0 is the interpreter ("sh"), not a file we can re-exec,
    # so re-fetch the installer and pipe it into sudo instead.
    if command -v sudo >/dev/null 2>&1; then
        log "not running as root; re-execing with sudo"
        if [ -r "$0" ] && [ "$0" != "sh" ] && [ "$0" != "-sh" ] && [ "$0" != "/bin/sh" ] && [ "$0" != "/usr/bin/sh" ]; then
            exec sudo -E sh "$0" "$@"
        else
            exec curl -fsSL "$INSTALLER_URL" | sudo -E sh
        fi
    fi
    bail "must be run as root (or via sudo)"
fi

# --- tools ----------------------------------------------------------------

need() {
    command -v "$1" >/dev/null 2>&1 || bail "required command not found: $1"
}
need curl
need install
need useradd 2>/dev/null || need adduser 2>/dev/null

# --- download helper ------------------------------------------------------

fetch() {
    # fetch <url> <dest>
    log "downloading $1"
    curl -fsSL -o "$2" "$1" || bail "failed to download $1"
}

# --- create system user/group --------------------------------------------

if ! getent group "$GROUP_NAME" >/dev/null 2>&1; then
    log "creating group $GROUP_NAME"
    groupadd --system "$GROUP_NAME" 2>/dev/null || addgroup --system "$GROUP_NAME"
fi

if ! getent passwd "$USER_NAME" >/dev/null 2>&1; then
    log "creating system user $USER_NAME"
    useradd --system \
        --no-create-home \
        --shell /usr/sbin/nologin \
        --gid "$GROUP_NAME" \
        --home-dir "$DATA_DIR" \
        "$USER_NAME" 2>/dev/null || \
    adduser --system --no-create-home --shell /usr/sbin/nologin \
        --ingroup "$GROUP_NAME" --home "$DATA_DIR" "$USER_NAME"
fi

# --- install binary -------------------------------------------------------

log "installing server binary into $PREFIX/bin"
install -d -m 0755 "$PREFIX/bin"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fetch "${DOWNLOAD_BASE}/llama-admin-server-${asset_arch}" "${tmp}/llama-admin-server"
install -m 0755 "${tmp}/llama-admin-server" "$PREFIX/bin/llama-admin-server"

# --- install config -------------------------------------------------------

log "installing config into $SYSCONF/llama-admin"
install -d -m 0755 "$SYSCONF/llama-admin"
if [ -f "$SYSCONF/llama-admin/config.yaml" ]; then
    echo "    $SYSCONF/llama-admin/config.yaml already exists; leaving it alone"
else
    fetch "${DOWNLOAD_BASE}/config.example.yaml" "${tmp}/config.example.yaml"
    install -m 0644 "${tmp}/config.example.yaml" "$SYSCONF/llama-admin/config.example.yaml"
    # Drop a working config.yaml pointing at the right data dir so the service
    # can start without manual editing (OAuth still needs filling in).
    sed "s|^dataDir:.*|dataDir: $DATA_DIR|" \
        "$SYSCONF/llama-admin/config.example.yaml" \
        > "$SYSCONF/llama-admin/config.yaml"
    chmod 0640 "$SYSCONF/llama-admin/config.yaml"
    chown "root:${GROUP_NAME}" "$SYSCONF/llama-admin/config.yaml"
    echo "    created $SYSCONF/llama-admin/config.yaml (edit OAuth credentials here)"
fi

# --- data directory -------------------------------------------------------

log "creating data directory $DATA_DIR"
install -d -m 0755 "$DATA_DIR"
install -d -m 0755 "$DATA_DIR/data"
install -d -m 0755 "$DATA_DIR/models"
chown -R "${USER_NAME}:${GROUP_NAME}" "$DATA_DIR"

# --- systemd unit ---------------------------------------------------------

log "installing systemd unit into $UNITDIR"
fetch "${DOWNLOAD_BASE}/llama-admin.service" "${tmp}/llama-admin.service"
# Rewrite the unit's ExecStart / paths to match PREFIX and SYSCONF.
sed -i \
    -e "s|ExecStart=.*|ExecStart=$PREFIX/bin/llama-admin-server|" \
    -e "s|Environment=LLAMA_ADMIN_CONFIG_PATH=.*|Environment=LLAMA_ADMIN_CONFIG_PATH=$SYSCONF/llama-admin/config.yaml|" \
    -e "s|Environment=LLAMA_ADMIN_DATA_DIR=.*|Environment=LLAMA_ADMIN_DATA_DIR=$DATA_DIR|" \
    -e "s|WorkingDirectory=.*|WorkingDirectory=$DATA_DIR|" \
    -e "s|ReadWritePaths=.*|ReadWritePaths=$DATA_DIR|" \
    "${tmp}/llama-admin.service"
install -m 0644 "${tmp}/llama-admin.service" "$UNITDIR/llama-admin.service"

log "reloading systemd"
systemctl daemon-reload
log "enabling llama-admin.service (not starting it yet)"
systemctl enable llama-admin.service 2>/dev/null || true

cat <<EOF

  llama-admin server installed.

  Binary:          $PREFIX/bin/llama-admin-server
  Config:          $SYSCONF/llama-admin/config.yaml
  Data directory:  $DATA_DIR
  Systemd unit:    $UNITDIR/llama-admin.service

  Next steps:

    1. Edit $SYSCONF/llama-admin/config.yaml and enable OAuth (set
       providers.github.enabled: true and fill in client id/secret).
    2. Start the service:
         systemctl start llama-admin
         journalctl -u llama-admin -f

EOF
