#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

echo "=== AgentHub Multi-Agent SSE Smoke (Go tests) ==="

echo "1) Gateway multi-agent SSE route smoke"
go test ./services/gateway -run TestGateway_StaticAgentRegistry_MultiAgentChatSSE -count=1

echo "2) Gateway /api/agents smoke"
go test ./services/gateway/httpapi -run 'TestListAgents(Default|WithOverride)' -count=1

echo ""
echo "=== Multi-agent SSE smoke PASSED ==="
