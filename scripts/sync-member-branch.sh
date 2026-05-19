#!/usr/bin/env bash
set -euo pipefail

MEMBER_BRANCH="${1:-}"

if [ -z "$MEMBER_BRANCH" ]; then
  echo "Usage: bash scripts/sync-member-branch.sh <c|y|g>"
  echo "Example: bash scripts/sync-member-branch.sh c"
  exit 1
fi

git fetch origin

git checkout dev
git pull origin dev

git checkout "$MEMBER_BRANCH"
git pull origin "$MEMBER_BRANCH"
git merge dev

git push origin "$MEMBER_BRANCH"

echo "Synced member branch '$MEMBER_BRANCH' with latest dev."
