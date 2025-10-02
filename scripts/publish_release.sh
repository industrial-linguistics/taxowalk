#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=$(cat "$ROOT_DIR/VERSION")

"$ROOT_DIR/scripts/publish_debian.sh"
"$ROOT_DIR/scripts/build_macos.sh" "$VERSION"
pwsh -File "$ROOT_DIR/scripts/build_windows.ps1" -Version "$VERSION"

if [[ -z "${DEPLOYMENT_SSH_KEY:-}" ]]; then
    echo "DEPLOYMENT_SSH_KEY is not set" >&2
    exit 1
fi

KEY_FILE=$(mktemp)
trap 'rm -f "$KEY_FILE"' EXIT
printf '%s' "$DEPLOYMENT_SSH_KEY" > "$KEY_FILE"
chmod 600 "$KEY_FILE"

REMOTE="taxowalk@merah.cassia.ifost.org.au:/var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk"

rsync -avz -e "ssh -i $KEY_FILE -o StrictHostKeyChecking=no" "$ROOT_DIR/dist/macos/" "$REMOTE/macos/"
rsync -avz -e "ssh -i $KEY_FILE -o StrictHostKeyChecking=no" "$ROOT_DIR/dist/windows/" "$REMOTE/windows/"
