#!/usr/bin/env bash
#
# Lightcode installer.
#
# Downloads the latest prebuilt binary from GitHub Releases and puts it on your
# PATH. Falls back to `go install` for platforms that have no prebuilt asset.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Kartik-2239/lightcode/main/install.sh | bash
#
#   # pin a version
#   curl -fsSL https://raw.githubusercontent.com/Kartik-2239/lightcode/main/install.sh | bash -s -- --version v1.2.3
#
#   # pick the install dir
#   curl -fsSL ... | bash -s -- --bin-dir ~/.bin
#   # or via env:
#   LIGHTCODE_BIN=~/.bin curl -fsSL ... | bash
#
set -euo pipefail

REPO="Kartik-2239/lightcode"
INSTALL_NAME="lightcode"
VERSION=""
BIN_DIR_OVERRIDE="${LIGHTCODE_BIN:-}"

log()  { printf '%s\n' "$*"; }
note() { printf '\033[36m=>\033[0m %s\n' "$*"; }
warn() { printf '\033[33mwarn:\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[31merror:\033[0m %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

# --- args ---
while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      [ $# -ge 2 ] || die "--version needs a tag (e.g. v1.2.3)"
      VERSION="$2"; shift 2;;
    --bin-dir)
      [ $# -ge 2 ] || die "--bin-dir needs a path"
      BIN_DIR_OVERRIDE="$2"; shift 2;;
    --help|-h)
      log "Lightcode installer"
      log "  --version <tag>  Install a specific release tag (default: latest)"
      log "  --bin-dir <path> Install directory (default: \$HOME/.local/bin or /usr/local/bin)"
      exit 0;;
    *) die "unknown option: $1 (try --help)";;
  esac
done

need_cmd() { command -v "$1" >/dev/null 2>&1; }

# --- detect platform ---
raw_os="$(uname -s)"
raw_arch="$(uname -m)"

case "$raw_os" in
  Darwin)                              os="darwin" ;;
  Linux)                               os="linux" ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT)     os="windows" ;;
  *) die "unsupported OS: $raw_os" ;;
esac

case "$raw_arch" in
  x86_64|amd64)   arch="amd64" ;;
  aarch64|arm64)  arch="arm64" ;;
  *) die "unsupported architecture: $raw_arch" ;;
esac

ext=""
[ "$os" = "windows" ] && ext=".exe"
asset="${INSTALL_NAME}-${os}-${arch}${ext}"

# --- resolve download URL ---
if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
  download_base="https://github.com/${REPO}/releases/latest/download"
else
  download_base="https://github.com/${REPO}/releases/download/${VERSION}"
fi

# --- temp workspace ---
tmpdir="$(mktemp -d 2>/dev/null || mktemp -d -t lightcode)"
trap 'rm -rf "$tmpdir"' EXIT

# --- download + checksum-verify (if checksums.txt exists for the release) ---
download_asset() {
  local url="$1"
  note "Downloading $asset"
  curl -fSL -o "$tmpdir/$asset" "$url" || die "download failed: $url"

  if curl -fsSL "$download_base/checksums.txt" -o "$tmpdir/checksums.txt" 2>/dev/null; then
    local expected actual
    expected="$(grep -F "$asset" "$tmpdir/checksums.txt" | awk '{print $1}' || true)"
    if [ -z "$expected" ]; then
      warn "no checksum entry for $asset; skipping verification"
      return
    fi
    if need_cmd sha256sum; then
      actual="$(sha256sum "$tmpdir/$asset" | awk '{print $1}')"
    elif need_cmd shasum; then
      actual="$(shasum -a 256 "$tmpdir/$asset" | awk '{print $1}')"
    else
      warn "no sha256 tool found; skipping verification"
      return
    fi
    [ "$actual" = "$expected" ] || die "checksum mismatch for $asset
  expected: $expected
  got:      $actual"
    note "Checksum verified"
  else
    warn "no checksums.txt for this release; skipping verification"
  fi
}

# --- pick install dir: override > ~/.local/bin > /usr/local/bin (sudo) ---
choose_bindir() {
  if [ -n "$BIN_DIR_OVERRIDE" ]; then
    printf '%s\n' "$BIN_DIR_OVERRIDE"; return
  fi
  local candidate="$HOME/.local/bin"
  if { [ -d "$candidate" ] || mkdir -p "$candidate" 2>/dev/null; } && [ -w "$candidate" ]; then
    printf '%s\n' "$candidate"; return
  fi
  printf '%s\n' "/usr/local/bin"
}

download_asset "$download_base/$asset"
binpath="$tmpdir/$asset"

# --- place binary ---
bindir="$(choose_bindir)"
target="$bindir/$INSTALL_NAME$ext"

# create the dir without sudo when possible
if [ ! -d "$bindir" ]; then
  if mkdir -p "$bindir" 2>/dev/null; then
    :
  else
    warn "$bindir could not be created; using sudo"
    sudo mkdir -p "$bindir"
  fi
fi

if [ -w "$bindir" ]; then
  mv "$binpath" "$target"
else
  warn "$bindir is not writable; using sudo"
  sudo mv "$binpath" "$target"
fi
chmod +x "$target" 2>/dev/null || sudo chmod +x "$target"
note "Installed $INSTALL_NAME to $target"

# --- PATH check ---
case ":$PATH:" in
  *":$bindir:"*) ;;
  *)
    warn "$bindir is not in your PATH"
    log "  Add this to your shell config (~/.bashrc, ~/.zshrc):"
    log "    export PATH=\"$bindir:\$PATH\""
    ;;
esac

# --- Linux runtime dep check: the CGO binary links libX11 dynamically ---
if [ "$os" = "linux" ] && need_cmd ldd; then
  ldd_out="$(ldd "$target" 2>/dev/null || true)"
  if printf '%s\n' "$ldd_out" | grep -q 'libX11'; then
    if printf '%s\n' "$ldd_out" | grep -q 'libX11.*=> not found'; then
      warn "libX11 is not installed; clipboard features need it."
      log "  Debian/Ubuntu:  sudo apt install libx11-6"
      log "  Fedora/RHEL:    sudo dnf install libX11"
      log "  Arch:           sudo pacman -S libx11"
      log "  openSUSE:       sudo zypper install libX11"
    fi
  fi
fi

# --- verify ---
if [ -x "$target" ]; then
  note "Verifying:"
  "$target" --version 2>/dev/null || warn "could not run 'lightcode --version' (missing runtime libs?)"
fi

log ""
note "Done. Run: lightcode"
