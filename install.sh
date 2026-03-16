#!/bin/sh
set -e

# OpenBotKit installer
# curl -fsSL https://raw.githubusercontent.com/priyanshujain/openbotkit/master/install.sh | sh

REPO="priyanshujain/openbotkit"
INSTALL_DIR="${OBK_INSTALL_DIR:-$HOME/.local/bin}"
OBK_DIR="$HOME/.obk"

log()   { printf "\033[1;32m==> %s\033[0m\n" "$1"; }
warn()  { printf "\033[1;33m    %s\033[0m\n" "$1"; }
fatal() { printf "\033[1;31merror: %s\033[0m\n" "$1" >&2; exit 1; }

download() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        fatal "curl or wget required"
    fi
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        riscv64)       ARCH="riscv64" ;;
        *) fatal "unsupported architecture: $ARCH" ;;
    esac
    case "$OS" in
        darwin|linux) ;;
        *) fatal "unsupported OS: $OS" ;;
    esac
}

get_latest_version() {
    download "https://api.github.com/repos/${REPO}/releases/latest" /dev/stdout 2>/dev/null \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/'
}

install_binary() {
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        fatal "no release found — use 'sh install.sh --source' to build from source"
    fi

    log "Downloading obk ${VERSION} (${OS}/${ARCH})"

    ARCHIVE="openbotkit_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    download "$URL" "$tmpdir/$ARCHIVE"
    tar -xzf "$tmpdir/$ARCHIVE" -C "$tmpdir"

    mkdir -p "$INSTALL_DIR"
    mv "$tmpdir/obk" "$INSTALL_DIR/obk"
    chmod +x "$INSTALL_DIR/obk"
    ln -sf "$INSTALL_DIR/obk" "$INSTALL_DIR/openbotkit"
}

install_from_source() {
    command -v go >/dev/null 2>&1 || fatal "Go required for source install — https://go.dev/dl/"

    log "Building obk from source"
    go install "github.com/priyanshujain/openbotkit@latest"

    GOBIN="$(go env GOPATH)/bin"
    ln -sf "$GOBIN/openbotkit" "$GOBIN/obk"
    INSTALL_DIR="$GOBIN"
}

install_macos_helper() {
    [ "$OS" = "darwin" ] || return 0

    log "Installing macOS helper (Apple Contacts & Notes)"
    mkdir -p "$OBK_DIR/bin"

    # Try pre-built binary from release
    if [ -n "$VERSION" ]; then
        HELPER_URL="https://github.com/${REPO}/releases/download/${VERSION}/obkmacos-darwin-${ARCH}"
        if download "$HELPER_URL" "$OBK_DIR/bin/obkmacos" 2>/dev/null; then
            chmod +x "$OBK_DIR/bin/obkmacos"
            return 0
        fi
    fi

    # Build from source
    if command -v swiftc >/dev/null 2>&1; then
        # Check if we're running from a repo checkout
        SCRIPT_DIR="$(cd "$(dirname "$0")" 2>/dev/null && pwd || echo "")"
        if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/swift/obkmacos.swift" ]; then
            SWIFT_SRC="$SCRIPT_DIR/swift/obkmacos.swift"
        else
            SWIFT_SRC=$(mktemp)
            download "https://raw.githubusercontent.com/${REPO}/master/swift/obkmacos.swift" "$SWIFT_SRC"
            SWIFT_SRC_TMP="$SWIFT_SRC"
        fi
        swiftc -O -o "$OBK_DIR/bin/obkmacos" "$SWIFT_SRC"
        if [ -n "$SWIFT_SRC_TMP" ]; then rm -f "$SWIFT_SRC_TMP"; fi
    else
        warn "Xcode Command Line Tools not found — skipping Apple Contacts/Notes"
        warn "Install with: xcode-select --install"
        warn "Then re-run this script"
    fi
}

ensure_path() {
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) return 0 ;;
    esac

    SHELL_RC=""
    case "$(basename "$SHELL")" in
        zsh)  SHELL_RC="$HOME/.zshrc" ;;
        bash) SHELL_RC="$HOME/.bashrc" ;;
    esac

    if [ -n "$SHELL_RC" ] && [ -f "$SHELL_RC" ]; then
        if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
            printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$SHELL_RC"
        fi
    fi

    export PATH="$INSTALL_DIR:$PATH"
}

install_skills() {
    if command -v obk >/dev/null 2>&1; then
        log "Installing skills"
        obk update --skills-only 2>/dev/null || true
    fi
}

print_done() {
    echo ""
    log "OpenBotKit installed successfully!"
    echo ""
    echo "  Get started:"
    echo "    \$ obk setup"
    echo ""
}

main() {
    detect_platform

    if [ "$1" = "--source" ]; then
        install_from_source
    else
        install_binary
    fi

    install_macos_helper
    ensure_path
    install_skills
    print_done
}

main "$@"
