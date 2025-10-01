#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=$(cat "$ROOT_DIR/VERSION")
DIST_DIR="$ROOT_DIR/dist/deb"
BUILD_DIR="$ROOT_DIR/build/deb"
BIN_DIR="$BUILD_DIR/usr/bin"
MAN_DIR="$BUILD_DIR/usr/share/man/man1"
CONTROL_DIR="$BUILD_DIR/DEBIAN"

rm -rf "$BUILD_DIR"
mkdir -p "$BIN_DIR" "$MAN_DIR" "$CONTROL_DIR" "$DIST_DIR"

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$BIN_DIR/bulksense" "$ROOT_DIR/cmd/bulksense"

install -m 0644 "$ROOT_DIR/docs/bulksense.1" "$MAN_DIR/bulksense.1"
gzip -f "$MAN_DIR/bulksense.1"

cat > "$CONTROL_DIR/control" <<CONTROL
Package: bulksense
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Industrial Linguistics <packages@industrial-linguistics.com>
Description: Shopify taxonomy classifier powered by OpenAI
 bulksense classifies product descriptions into the Shopify taxonomy by
 iteratively querying the gpt-5-mini model.
CONTROL

chmod 0755 "$BUILD_DIR/usr"
chmod -R go-w "$BUILD_DIR/usr/share/man"

OUTPUT="$DIST_DIR/bulksense_${VERSION}_amd64.deb"

dpkg-deb --build "$BUILD_DIR" "$OUTPUT"

echo "Built $OUTPUT"
