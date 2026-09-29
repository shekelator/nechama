#!/usr/bin/env bash
#
# install.sh — download and install the latest nechama release from GitHub.
#
# Downloads the archive for the current platform, extracts the binary, and
# installs it to $INSTALL_DIR (default: ~/.local/bin). On macOS it also
# ad-hoc re-signs the binary and clears the quarantine xattr so macOS Sequoia
# Gatekeeper does not kill it on launch — see the README for why both steps
# are required.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/shekelator/nechama/main/scripts/install.sh | bash
#
#   # install into a specific directory:
#   curl -fsSL .../install.sh | INSTALL_DIR=/usr/local/bin bash
#
#   # install a specific tag:
#   curl -fsSL .../install.sh | NECHAMA_VERSION=v0.4.3 bash

set -euo pipefail

REPO="shekelator/nechama"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# --- detect platform -------------------------------------------------------

case "$(uname -s)" in
  Darwin)                  os="macOS"  ;;
  Linux)                   os="linux"  ;;
  MINGW*|MSYS*|CYGWIN*)     os="windows" ;;
  *) echo "install.sh: unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "install.sh: unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

# --- resolve version -------------------------------------------------------

if [ -n "${NECHAMA_VERSION:-}" ]; then
  tag="$NECHAMA_VERSION"
else
  echo "Fetching latest release of $REPO..."
  if ! tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
            | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]*)".*/\1/'); then
    echo "install.sh: could not reach GitHub API" >&2; exit 1
  fi
  if [ -z "$tag" ]; then
    echo "install.sh: could not determine latest release tag" >&2; exit 1
  fi
fi
echo "Installing nechama $tag ($os/$arch)"

# --- download --------------------------------------------------------------

version="${tag#v}"  # strip leading 'v' to match archive naming (0.4.3, not v0.4.3)
if [ "$os" = "windows" ]; then
  asset="nechama_${version}_${os}_${arch}.zip"
else
  asset="nechama_${version}_${os}_${arch}.tar.gz"
fi
url="https://github.com/$REPO/releases/download/$tag/$asset"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $url"
curl -fsSL --output "$tmp/$asset" "$url"

# --- extract ---------------------------------------------------------------

case "$asset" in
  *.tar.gz) tar -xzf "$tmp/$asset" -C "$tmp" ;;
  *.zip)    unzip -q "$tmp/$asset" -d "$tmp" ;;
esac

bin="$(find "$tmp" -type f -name nechama -print -quit)"
if [ -z "$bin" ] || [ "$os" = "windows" ]; then
  # On Windows the executable is nechama.exe.
  bin="$(find "$tmp" -type f -name 'nechama.exe' -print -quit)"
fi
if [ -z "$bin" ]; then
  echo "install.sh: nechama binary not found in archive" >&2; exit 1
fi

# --- macOS: re-sign + clear quarantine -------------------------------------
#
# The release binary is ad-hoc signed (so the kernel runs it on Apple Silicon)
# but not notarized, so Sequoia Gatekeeper kills a browser-downloaded copy on
# launch. Re-signing locally produces a new signature Gatekeeper has no
# rejected assessment for, and clearing quarantine stops the launch-time
# assessment. Both steps are required; either alone still results in
# "zsh: killed".

if [ "$os" = "macOS" ]; then
  echo "Re-signing binary and clearing quarantine (macOS Sequoia workaround)..."
  codesign --force --sign - "$bin" >/dev/null 2>&1 || {
    echo "install.sh: codesign failed — install Xcode Command Line Tools:" >&2
    echo "    xcode-select --install" >&2
    exit 1
  }
  xattr -d com.apple.quarantine "$bin" 2>/dev/null || true
fi

# --- install ---------------------------------------------------------------

mkdir -p "$INSTALL_DIR"
if [ "$os" = "windows" ]; then
  target="$INSTALL_DIR/nechama.exe"
else
  target="$INSTALL_DIR/nechama"
fi
install -m 0755 "$bin" "$target"

echo
echo "Installed: $target"
if command -v "$target" >/dev/null 2>&1; then
  "$target" version
else
  echo "Note: $INSTALL_DIR is not on your PATH. Add it:"
  echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
fi