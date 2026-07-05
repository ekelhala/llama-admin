#!/bin/sh
# llama-admin CLI installer
#
# Downloads the latest build of the llama-admin CLI binary from the rolling
# "latest" GitHub release and installs it into $PREFIX/bin. Picks the right
# asset (linux/amd64 or linux/arm64) based on `uname -m`.
#
# The CLI is shipped as a static (CGO_ENABLED=0) binary, so the arm64
# build also runs inside Termux on Android phones.
#
# Intended for workstations/admin machines that talk to a llama-admin server
# running elsewhere — no system user, systemd unit or config is created.
#
# Usage:
#   curl -fsSL https://github.com/ekelhala/llama-admin/releases/download/latest/install-cli.sh | sh
#
# Or, to inspect first:
#   curl -fsSL -o install-cli.sh https://github.com/ekelhala/llama-admin/releases/download/latest/install-cli.sh
#   sh install-cli.sh
#
# Env vars (optional):
#   PREFIX       install prefix for the binary         (default: /usr/local)
#   RELEASE_TAG  which release tag to pull from       (default: latest)
#
set -eu

OWNER=ekelhala
REPO=llama-admin

PREFIX="${PREFIX:-/usr/local}"
RELEASE_TAG="${RELEASE_TAG:-latest}"

DOWNLOAD_BASE="https://github.com/${OWNER}/${REPO}/releases/download/${RELEASE_TAG}"
INSTALLER_URL="${DOWNLOAD_BASE}/install-cli.sh"

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

DOWNLOAD_URL="${DOWNLOAD_BASE}/llama-admin-${asset_arch}"

# --- tools ----------------------------------------------------------------

need() {
    command -v "$1" >/dev/null 2>&1 || bail "required command not found: $1"
}
need curl

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

# --- download + install ---------------------------------------------------

log "installing llama-admin CLI into $PREFIX/bin"
install -d -m 0755 "$PREFIX/bin"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
log "downloading $DOWNLOAD_URL"
curl -fsSL -o "${tmp}/llama-admin" "$DOWNLOAD_URL" || bail "failed to download llama-admin"
install -m 0755 "${tmp}/llama-admin" "$PREFIX/bin/llama-admin"

cat <<EOF

  llama-admin CLI installed.

  Binary:  $PREFIX/bin/llama-admin

  Next steps:

    1. Point the CLI at your llama-admin server:
         llama-admin config set-server https://your-host:8080
    2. Log in via OAuth:
         llama-admin auth login
    3. Run \`llama-admin --help\` to see available commands.

EOF
