#!/usr/bin/env bash
# Run the v3 quality gates locally against an explicitly provided MySQL/Redis.
# This script never starts Docker and never reads .env or credential files.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

: "${AGENTHUB_MYSQL_TEST_DSN:?export AGENTHUB_MYSQL_TEST_DSN before running}"
: "${AGENTHUB_REDIS_TEST_ADDR:?export AGENTHUB_REDIS_TEST_ADDR before running}"

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

echo "== integration (MySQL + Redis) =="
go test -v -count=1 -run 'TestIntegration' \
  ./services/orchestrator/... \
  ./services/gateway/... 2>&1 | tee "$log"

if grep -q -- "--- SKIP" "$log"; then
  echo "FAIL: integration tests were skipped despite configured services" >&2
  grep -- "--- SKIP" "$log" >&2
  exit 1
fi
if ! grep -q -- "--- PASS" "$log"; then
  echo "FAIL: no integration test passed" >&2
  exit 1
fi
echo "PASS: integration tests ran without SKIP"

echo "== Go unit/build matrix =="
modules=(
  pkg/adk
  pkg/runtime
  services/gateway
  services/orchestrator
  services/agents/code-agent
  services/agents/web-agent
  services/agents/test-agent
  services/agents/security-agent
  services/agents/diff-agent
  services/agents/markdown-agent
  services/agents/review-agent
  services/agents/document-agent
  services/agents/vision-agent
  services/agents/context-agent
  services/agents/deploy-agent
)
for module in "${modules[@]}"; do
  echo "-- $module"
  (cd "$module" && go test ./... -count=1)
  case "$module" in
    services/gateway) (cd "$module" && go build ./cmd/gateway) ;;
    services/orchestrator) (cd "$module" && go build ./cmd/orchestrator) ;;
    services/agents/code-agent) (cd "$module" && go build ./cmd/code-agent) ;;
    services/agents/web-agent) (cd "$module" && go build ./cmd/web-agent) ;;
  esac
done
echo "PASS: Go matrix"

if [[ "${RUN_FRONTEND:-1}" == "1" ]]; then
  if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
    echo "WARN: node/npm not available; frontend gate skipped" >&2
  else
    echo "== frontend =="
    (cd frontend && npm test -- --run && npm run build)
    echo "PASS: frontend"
  fi
fi

echo "== PASS: v3 gates completed =="
