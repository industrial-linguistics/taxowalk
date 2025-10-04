#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=${1:-$(cat "$ROOT_DIR/VERSION")}
DIST_DIR="$ROOT_DIR/dist/macos"
WORK_DIR="$ROOT_DIR/build/macos"
BINARIES=(taxowalk taxoname taxopath)

rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR" "$DIST_DIR"

for arch in amd64 arm64; do
    PKG_DIR="$WORK_DIR/package-$arch"
    mkdir -p "$PKG_DIR/usr/local/bin" "$PKG_DIR/usr/local/share/man/man1"

    for bin in "${BINARIES[@]}"; do
        GOOS=darwin GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$PKG_DIR/usr/local/bin/$bin" "$ROOT_DIR/cmd/$bin"
        chmod 0755 "$PKG_DIR/usr/local/bin/$bin"
        install -m 0644 "$ROOT_DIR/docs/${bin}.1" "$PKG_DIR/usr/local/share/man/man1/${bin}.1"
    done

    tarball="$DIST_DIR/taxowalk_${VERSION}_darwin_${arch}.tar.gz"
    (cd "$PKG_DIR" && tar -czf "$tarball" .)
    echo "Built $tarball"
    rm -rf "$PKG_DIR"
done
