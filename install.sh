#!/bin/sh
set -eu

# OpenBotKit installer
# curl -fsSL https://raw.githubusercontent.com/73ai/openbotkit/master/install.sh | sh

REPO="73ai/openbotkit"
INSTALL_DIR="${OBK_INSTALL_DIR:-$HOME/.local/bin}"
OBK_DIR="$HOME/.obk"
VERSION=""
SWIFT_SRC_TMP=""

log()   { printf "\033[1;32m==> %s\033[0m\n" "$1"; }
warn()  { printf "\033[1;33m    %s\033[0m\n" "$1"; }
fatal() { printf "\033[1;31merror: %s\033[0m\n" "$1" >&2; exit 1; }

check_cmd() { command -v "$1" >/dev/null 2>&1; }

need_cmd() {
    if ! check_cmd "$1"; then
        fatal "need '$1' (command not found)"
    fi
}

download() {
    if check_cmd curl; then
        curl -fsSL -o "$2" "$1"
    elif check_cmd wget; then
        wget -qO "$2" "$1"
    else
        fatal "need 'curl' or 'wget' (neither found)"
    fi
}

detect_platform() {
    need_cmd uname
    need_cmd mktemp
    need_cmd chmod
    need_cmd tar

    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    # Detect Rosetta 2: uname -m lies under translation
    if [ "$OS" = "darwin" ] && [ "$ARCH" = "x86_64" ]; then
        if sysctl hw.optional.arm64 2>/dev/null | grep -q ': 1'; then
            ARCH="arm64"
        fi
    fi

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
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
        fatal "no release found — install from source instead:
    git clone https://github.com/${REPO} && cd openbotkit && make install
  requires: go (https://go.dev/dl/)"
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
    if ! check_cmd go; then
        fatal "go is required to build from source — install it first: https://go.dev/dl/"
    fi

    log "Building obk from source"
    go install "github.com/${REPO}@latest"

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

    # Build from source if swiftc available
    if check_cmd swiftc; then
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
        log "Xcode Command Line Tools required for Apple Contacts/Notes"
        xcode-select --install 2>/dev/null || true
        warn "Complete the Xcode install dialog, then re-run this script"
    fi
}

# Create an env script that can be sourced from shell rc files
write_env_script() {
    mkdir -p "$OBK_DIR"
    cat > "$OBK_DIR/env" <<ENVEOF
# OpenBotKit PATH setup (sourced by shell rc)
case ":\${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) export PATH="${INSTALL_DIR}:\$PATH" ;;
esac
ENVEOF
}

ensure_path() {
    write_env_script

    SOURCE_LINE=". \"$OBK_DIR/env\""

    for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile"; do
        if [ -f "$rc" ]; then
            if ! grep -q "$OBK_DIR/env" "$rc" 2>/dev/null; then
                printf '\n# OpenBotKit\n%s\n' "$SOURCE_LINE" >> "$rc"
            fi
        fi
    done

    # Also write GITHUB_PATH for CI environments
    if [ -n "${GITHUB_PATH:-}" ]; then
        echo "$INSTALL_DIR" >> "$GITHUB_PATH"
    fi

    export PATH="$INSTALL_DIR:$PATH"
}

install_skills() {
    if check_cmd obk; then
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
    if ! echo ":$PATH:" | grep -q ":$INSTALL_DIR:"; then
        warn "Restart your shell or run: source $OBK_DIR/env"
    fi
}

main() {
    detect_platform

    if [ "${1:-}" = "--source" ]; then
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
