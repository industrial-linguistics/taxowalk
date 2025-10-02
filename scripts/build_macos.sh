#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=${1:-$(cat "$ROOT_DIR/VERSION")}
DIST_DIR="$ROOT_DIR/dist/macos"
WORK_DIR="$ROOT_DIR/build/macos"

rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR" "$DIST_DIR"

for arch in amd64 arm64; do
    BIN_NAME="taxowalk"
    OUT_BIN="$WORK_DIR/${BIN_NAME}-${arch}"
    GOOS=darwin GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT_BIN" "$ROOT_DIR/cmd/taxowalk"
    PKG_DIR="$WORK_DIR/package-$arch"
    mkdir -p "$PKG_DIR/usr/local/bin" "$PKG_DIR/usr/local/share/man/man1"
    install -m 0755 "$OUT_BIN" "$PKG_DIR/usr/local/bin/$BIN_NAME"
    install -m 0644 "$ROOT_DIR/docs/taxowalk.1" "$PKG_DIR/usr/local/share/man/man1/taxowalk.1"
    tarball="$DIST_DIR/taxowalk_${VERSION}_darwin_${arch}.tar.gz"
    (cd "$PKG_DIR" && tar -czf "$tarball" .)
    echo "Built $tarball"
    rm -rf "$PKG_DIR"
    rm -f "$OUT_BIN"
done
