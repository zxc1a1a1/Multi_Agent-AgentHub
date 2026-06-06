#!/usr/bin/env bash
set -euo pipefail
REPO="${1:-.}"
PACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ ! -d "$REPO" ]]; then
  echo "Repo path not found: $REPO" >&2
  exit 1
fi
cp -R "$PACK_DIR/replacements/." "$REPO/"
echo "Applied replacements to $REPO"
echo "Now review DELETE_OR_DEPRECATE.md and move stale files to docs/contracts/_legacy/."
