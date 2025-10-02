#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=$(cat "$ROOT_DIR/VERSION")

"$ROOT_DIR/scripts/build_deb.sh"

APT_ROOT="$ROOT_DIR/build/apt"
rm -rf "$APT_ROOT"
mkdir -p "$APT_ROOT/pool/main/b/bulksense" "$APT_ROOT/dists/stable/main/binary-amd64"
cp "$ROOT_DIR/dist/deb/bulksense_${VERSION}_amd64.deb" "$APT_ROOT/pool/main/b/bulksense/"

dpkg-scanpackages "$APT_ROOT/pool" > "$APT_ROOT/dists/stable/main/binary-amd64/Packages"
gzip -kf "$APT_ROOT/dists/stable/main/binary-amd64/Packages"

cat > "$APT_ROOT/dists/stable/Release" <<RELEASE
Suite: stable
Codename: stable
Components: main
Architectures: amd64
Description: bulksense apt repository
RELEASE

if [[ -z "${DEPLOYMENT_SSH_KEY:-}" ]]; then
    echo "DEPLOYMENT_SSH_KEY is not set" >&2
    exit 1
fi

KEY_FILE=$(mktemp)
trap 'rm -f "$KEY_FILE"' EXIT
printf '%s' "$DEPLOYMENT_SSH_KEY" > "$KEY_FILE"
chmod 600 "$KEY_FILE"

REMOTE="shopify@merah.cassia.ifost.org.au:/var/www/vhosts/packages.industrial-linguistics.com/htdocs/shopify"

rsync -avz -e "ssh -i $KEY_FILE -o StrictHostKeyChecking=no" "$APT_ROOT/" "$REMOTE/apt/"
