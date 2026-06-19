#!/usr/bin/env bash
set -e

# https://github.com/Kartik-2239/lightcode/releases/download/v1.2.3/lightcode-darwin-arm64

REPO="Kartik-2239/lightcode"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
GO_URL="github.com/${REPO}/cmd/lightcode@"
RELEASES_URL="https://github.com/Kartik-2239/lightcode/releases/download/"
AARCH=$(uname -m)

LATEST_TAG=$(curl -fsSL https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name"' | cut -d '"' -f 4)

echo $OS
echo $AARCH
echo $LATEST_TAG


is_go_installed() {
    command -v go >/dev/null 2>&1
}

run_go_install() {
    go install ${GO_URL}${LATEST_TAG}
}

install_binary() {
    curl -fL -o lightcode "${RELEASES_URL}${LATEST_TAG}/lightcode-${OS}-${AARCH}"
}

if [ "$OS" = "linux" ]; then
    if is_go_installed; then
        run_go_install
    else
        install_binary
    fi
fi

if [ "$OS" = "darwin" ]; then
    if ! is_go_installed; then
        run_go_install
    else
        install_binary
    fi
fi

if [ "$OS" = "windows" ]; then
    if is_go_installed; then
        run_go_install
    else
        install_binary
    fi
fi
