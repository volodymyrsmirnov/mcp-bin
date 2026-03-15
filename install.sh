#!/bin/bash
set -euo pipefail

REPO="volodymyrsmirnov/mcp-bin"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="mcp-bin"

info() { printf '\033[1;34m%s\033[0m\n' "$*"; }
error() { printf '\033[1;31mError: %s\033[0m\n' "$*" >&2; exit 1; }

detect_asset() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Linux)
            case "$arch" in
                x86_64)  echo "mcp-bin-linux-amd64" ;;
                aarch64|arm64) echo "mcp-bin-linux-arm64" ;;
                *) error "Unsupported Linux architecture: $arch" ;;
            esac
            ;;
        Darwin)
            echo "mcp-bin-osx-universal"
            ;;
        *)
            error "Unsupported OS: $os"
            ;;
    esac
}

get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    if command -v curl &>/dev/null; then
        curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget &>/dev/null; then
        wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "curl or wget is required"
    fi
}

download() {
    local url="$1" dest="$2"
    if command -v curl &>/dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -qO "$dest" "$url"
    else
        error "curl or wget is required"
    fi
}

main() {
    local version="${1:-}"
    local asset download_url tmpfile

    asset="$(detect_asset)"
    info "Detected platform: $asset"

    if [ -z "$version" ]; then
        version="$(get_latest_version)"
    fi
    info "Installing mcp-bin $version"

    download_url="https://github.com/${REPO}/releases/download/${version}/${asset}"

    tmpfile="$(mktemp)"
    trap 'rm -f "$tmpfile"' EXIT

    info "Downloading $download_url"
    download "$download_url" "$tmpfile"

    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "$tmpfile"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$tmpfile" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "$tmpfile" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    # Remove quarantine attribute on macOS
    if [ "$(uname -s)" = "Darwin" ]; then
        info "Removing quarantine attribute"
        if [ -w "${INSTALL_DIR}/${BINARY_NAME}" ]; then
            xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null || true
        else
            sudo xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null || true
        fi
    fi

    info "mcp-bin $version installed successfully!"
    "${INSTALL_DIR}/${BINARY_NAME}" --help
}

main "$@"
