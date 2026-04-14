#!/usr/bin/env sh
# Install lore from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/vicyap/lore/main/install.sh | sh
#
# Environment variables:
#   LORE_VERSION  - version to install (default: latest)
#   LORE_INSTALL  - install directory (default: /usr/local/bin, or ~/.local/bin if no write access)

set -e

REPO="vicyap/lore"
VERSION="${LORE_VERSION:-latest}"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "error: unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect arch
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "error: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Resolve latest version
if [ "$VERSION" = "latest" ]; then
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v//' | sed 's/".*//')"
    if [ -z "$VERSION" ]; then
        echo "error: could not determine latest version" >&2
        exit 1
    fi
fi

# Pick install dir
INSTALL_DIR="${LORE_INSTALL:-}"
if [ -z "$INSTALL_DIR" ]; then
    if [ -w /usr/local/bin ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${HOME}/.local/bin"
    fi
fi
mkdir -p "$INSTALL_DIR"

TARBALL="lore_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${TARBALL}"

echo "Installing lore v${VERSION} (${OS}/${ARCH}) to ${INSTALL_DIR}..."

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${TARBALL}"
tar xzf "${TMPDIR}/${TARBALL}" -C "$TMPDIR"
install -m 755 "${TMPDIR}/lore" "${INSTALL_DIR}/lore"

echo "Installed lore to ${INSTALL_DIR}/lore"

# Check if install dir is on PATH
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) echo "Note: ${INSTALL_DIR} is not in your PATH. Add it with:" >&2
       echo "  export PATH=\"${INSTALL_DIR}:\$PATH\"" >&2 ;;
esac
