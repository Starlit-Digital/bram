#!/usr/bin/env bash
set -euo pipefail

APP_NAME="bram"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
GO="${GO:-go}"
COMPILE_ONLY=0

if [[ "${1:-}" == "--compile-only" ]]; then
  COMPILE_ONLY=1
fi

cd "$ROOT_DIR"

mkdir -p .build
"$GO" build -o ".build/$APP_NAME" ./cmd/bram

if [[ "$COMPILE_ONLY" == "1" ]]; then
  exit 0
fi

goos="$("$GO" env GOOS)"
goarch="$("$GO" env GOARCH)"
host_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
host_arch="$(uname -m)"
case "$host_arch" in
  arm64) host_arch="arm64" ;;
  x86_64) host_arch="amd64" ;;
esac

if [[ "$goos" != "$host_os" || "$goarch" != "$host_arch" ]]; then
  echo "refusing to install foreign build: GOOS=$goos GOARCH=$goarch host=$host_os/$host_arch" >&2
  exit 1
fi

bin_dir="$PREFIX/bin"
share_dir="$PREFIX/share/$APP_NAME"
stamp="$(date -u +%Y%m%d-%H%M%S)"
install_dir="$share_dir/installs/$stamp"

mkdir -p "$bin_dir" "$install_dir"
cp ".build/$APP_NAME" "$install_dir/$APP_NAME"
chmod 0755 "$install_dir/$APP_NAME"

if [[ -f "$bin_dir/$APP_NAME" || -L "$bin_dir/$APP_NAME" ]]; then
  cp -p "$bin_dir/$APP_NAME" "$install_dir/$APP_NAME.previous" 2>/dev/null || true
fi

tmp="$bin_dir/.$APP_NAME.tmp.$$"
cp "$install_dir/$APP_NAME" "$tmp"
chmod 0755 "$tmp"
mv "$tmp" "$bin_dir/$APP_NAME"

commit="$(git rev-parse --verify HEAD 2>/dev/null || printf 'unknown')"
dirty="false"
if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
  dirty="true"
fi
sha="$(shasum -a 256 "$bin_dir/$APP_NAME" | awk '{print $1}')"

cat > "$share_dir/install-info.txt" <<INFO
tool: $APP_NAME
source: $ROOT_DIR
commit: $commit
dirty: $dirty
go: $("$GO" version)
installed_at_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)
binary_sha256: $sha
INFO

echo "installed $bin_dir/$APP_NAME"
