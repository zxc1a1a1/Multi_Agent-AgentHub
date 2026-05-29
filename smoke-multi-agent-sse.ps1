param()

$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $RootDir

Write-Host "=== AgentHub Multi-Agent SSE Smoke (PowerShell) ==="

Write-Host "1) Gateway multi-agent SSE route smoke"
go test ./services/gateway -run TestGateway_StaticAgentRegistry_MultiAgentChatSSE -count=1
if ($LASTEXITCODE -ne 0) {
  throw "gateway SSE smoke failed"
}

Write-Host "2) Gateway /api/agents smoke"
go test ./services/gateway/httpapi -run "TestListAgents(Default|WithOverride)" -count=1
if ($LASTEXITCODE -ne 0) {
  throw "gateway /api/agents smoke failed"
}

if (Test-Path -LiteralPath "D:\Microsoft VS Code\nodejs\npm.cmd") {
  Write-Host "3) Frontend dynamic agent fallback smoke"
  Push-Location (Join-Path $RootDir "frontend")
  try {
    $env:Path = "D:\Microsoft VS Code\nodejs;" + $env:Path
    & "D:\Microsoft VS Code\nodejs\npm.cmd" test -- --run src/stores/agentStore.test.ts src/stores/messageStore.test.ts src/agui/client.test.ts src/components/WebPreview.test.tsx
    if ($LASTEXITCODE -ne 0) {
      throw "frontend fallback smoke failed"
    }
  } finally {
    Pop-Location
  }
}

Write-Host "=== Multi-agent SSE smoke PASSED ===" -ForegroundColor Green
