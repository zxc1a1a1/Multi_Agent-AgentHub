#!/usr/bin/env bash
set -euo pipefail

BRANCH="${1:-}"
BASE_BRANCH="${2:-dev}"

if [ -z "$BRANCH" ]; then
  echo "Usage: bash scripts/create-branch.sh <new-branch> [base-branch]"
  echo "Examples:"
  echo "  bash scripts/create-branch.sh feature/c/adk-base c"
  echo "  bash scripts/create-branch.sh feature/y/frontend-layout y"
  echo "  bash scripts/create-branch.sh bugfix/g/agent-stream g"
  exit 1
fi

git fetch origin

git checkout "$BASE_BRANCH"
git pull origin "$BASE_BRANCH"
git checkout -b "$BRANCH"

echo "Created branch: $BRANCH from $BASE_BRANCH"
