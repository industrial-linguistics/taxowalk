#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=$(cat "$ROOT_DIR/VERSION")

"$ROOT_DIR/scripts/build_deb.sh"

if ! command -v apt-ftparchive >/dev/null 2>&1; then
    echo "apt-ftparchive is required but not installed" >&2
    exit 1
fi

APT_ROOT="$ROOT_DIR/build/apt"
rm -rf "$APT_ROOT"
mkdir -p "$APT_ROOT/pool/main/t/taxowalk" "$APT_ROOT/dists/stable/main/binary-amd64"
cp "$ROOT_DIR/dist/deb/taxowalk_${VERSION}_amd64.deb" "$APT_ROOT/pool/main/t/taxowalk/"

pushd "$APT_ROOT" >/dev/null
apt-ftparchive packages pool/ > dists/stable/main/binary-amd64/Packages
gzip -fk dists/stable/main/binary-amd64/Packages
apt-ftparchive \
    -o APT::FTPArchive::Release::Suite="stable" \
    -o APT::FTPArchive::Release::Codename="stable" \
    -o APT::FTPArchive::Release::Components="main" \
    -o APT::FTPArchive::Release::Architectures="amd64" \
    -o APT::FTPArchive::Release::Description="taxowalk apt repository" \
    release dists/stable > dists/stable/Release
popd >/dev/null

if [[ -z "${DEPLOYMENT_SSH_KEY:-}" ]]; then
    echo "DEPLOYMENT_SSH_KEY is not set" >&2
    exit 1
fi

KEY_FILE=$(mktemp)
trap 'rm -f "$KEY_FILE"' EXIT
printf '%s' "$DEPLOYMENT_SSH_KEY" > "$KEY_FILE"
chmod 600 "$KEY_FILE"

REMOTE="taxowalk@merah.cassia.ifost.org.au:/var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk"

rsync -avz -e "ssh -i $KEY_FILE -o StrictHostKeyChecking=no" "$APT_ROOT/" "$REMOTE/apt/"

# Sign the Release file on the remote server
ssh -i "$KEY_FILE" -o StrictHostKeyChecking=no taxowalk@merah.cassia.ifost.org.au << 'REMOTE_SCRIPT'
cd /var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk/apt/dists/stable

# Sign Release file to create Release.gpg (detached signature)
gpg --batch --yes --armor --detach-sign -o Release.gpg Release

# Create InRelease (clearsigned Release)
gpg --batch --yes --clearsign -o InRelease Release

echo "Repository signed successfully"
REMOTE_SCRIPT
