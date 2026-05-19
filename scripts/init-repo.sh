#!/usr/bin/env bash
set -euo pipefail

REMOTE_URL="${1:-}"
MEMBER_BRANCHES=(c y g)

if [ -z "$REMOTE_URL" ]; then
  echo "Usage: bash scripts/init-repo.sh <remote-url>"
  echo "Example: bash scripts/init-repo.sh git@github.com:your-org/multi-agent-framework.git"
  exit 1
fi

if [ ! -d .git ]; then
  git init
fi

git add .
if ! git diff --cached --quiet; then
  git commit -m "init react go multi-agent framework"
fi

git branch -M master

if ! git remote | grep -q '^origin$'; then
  git remote add origin "$REMOTE_URL"
else
  git remote set-url origin "$REMOTE_URL"
fi

git push -u origin master

# dev is created from master and used as the team integration branch.
git checkout -B dev master
git push -u origin dev

# Long-lived member branches are created from dev.
for branch in "${MEMBER_BRANCHES[@]}"; do
  git checkout -B "$branch" dev
  git push -u origin "$branch"
done

git checkout dev

echo "Done. Remote has master, dev, and member branches: ${MEMBER_BRANCHES[*]}."
echo "Recommended protection: master and dev require PR/MR; c/y/g can be owned by each member."
