param(
  [string]$GatewayURL = "http://localhost:8080",
  [string]$CodeAgentURL = "http://localhost:8081",
  [string]$WebAgentURL = "http://localhost:8082",
  [int]$TimeoutSeconds = 60,
  [switch]$SkipAgentHealth = $false
)

$ErrorActionPreference = "Stop"
$script:HasFailure = $false
$script:DockerPreflight = [ordered]@{
  DockerCommand = $false
  DockerDaemon  = $false
  ComposePlugin = $false
  ComposeConfig = $false
}

if ($TimeoutSeconds -lt 10) {
  $TimeoutSeconds = 10
}

function Write-Pass([string]$Message) {
  Write-Host "[PASS] $Message" -ForegroundColor Green
}

function Write-Warn([string]$Message) {
  Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Fail([string]$Message) {
  Write-Host "[FAIL] $Message" -ForegroundColor Red
  $script:HasFailure = $true
}

function Write-TroubleshootingHints([string]$Context) {
  Write-Host "  HINT ($Context): run docker compose -f docker-compose.new-arch.yml up --build"
  Write-Host "  HINT ($Context): verify Docker daemon with: docker version"
  Write-Host "  HINT ($Context): verify compose plugin with: docker compose version"
  Write-Host "  HINT ($Context): verify compose file with: docker compose -f docker-compose.new-arch.yml config"
  Write-Host "  HINT ($Context): check port usage for 8080/8081/8082/3000"
}

function Invoke-Check([string]$Name, [scriptblock]$Action) {
  Write-Host "-> $Name"
  try {
    & $Action
    Write-Pass $Name
  } catch {
    Write-Fail "$Name :: $($_.Exception.Message)"
    Write-TroubleshootingHints $Name
  }
}

function Assert-ContainsAny([string]$Body, [string[]]$Needles, [string]$Message) {
  foreach ($needle in $Needles) {
    if ($Body -like "*$needle*") {
      return
    }
  }
  throw $Message
}

function Assert-NotContainsAny([string]$Body, [string[]]$Needles, [string]$Message) {
  foreach ($needle in $Needles) {
    if ($Body -like "*$needle*") {
      throw "$Message (matched: $needle)"
    }
  }
}

function Test-DockerComposePreflight {
  Write-Host "== Docker/Compose preflight =="

  $dockerCmd = Get-Command docker -ErrorAction SilentlyContinue
  if (-not $dockerCmd) {
    Write-Warn "docker command not found in PATH."
    Write-TroubleshootingHints "docker preflight"
    return
  }
  $script:DockerPreflight.DockerCommand = $true

  try {
    & docker version *> $null
    if ($LASTEXITCODE -eq 0) {
      $script:DockerPreflight.DockerDaemon = $true
      Write-Pass "docker daemon reachable"
    } else {
      Write-Warn "docker version failed (daemon may be unavailable)."
    }
  } catch {
    Write-Warn "docker version failed: $($_.Exception.Message)"
  }

  try {
    & docker compose version *> $null
    if ($LASTEXITCODE -eq 0) {
      $script:DockerPreflight.ComposePlugin = $true
      Write-Pass "docker compose plugin available"
    } else {
      Write-Warn "docker compose version failed."
    }
  } catch {
    Write-Warn "docker compose version failed: $($_.Exception.Message)"
  }

  if ($script:DockerPreflight.ComposePlugin) {
    try {
      & docker compose -f docker-compose.new-arch.yml config *> $null
      if ($LASTEXITCODE -eq 0) {
        $script:DockerPreflight.ComposeConfig = $true
        Write-Pass "docker compose config check passed"
      } else {
        Write-Warn "docker compose config failed."
      }
    } catch {
      Write-Warn "docker compose config failed: $($_.Exception.Message)"
    }
  }
}

function Test-HttpReady([string]$Name, [string]$Url, [scriptblock]$ReadyPredicate) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $attempt = 0
  $lastError = "no successful response"

  while ((Get-Date) -lt $deadline) {
    $attempt++
    try {
      $resp = Invoke-WebRequest -Method Get -Uri $Url -TimeoutSec 8
      if ($null -eq $ReadyPredicate -or (& $ReadyPredicate $resp)) {
        return $resp
      }
      $lastError = "endpoint reachable but readiness predicate failed"
    } catch {
      $lastError = $_.Exception.Message
    }
    Start-Sleep -Seconds 2
  }

  throw "$Name did not become ready within $TimeoutSeconds seconds at $Url. Last error: $lastError"
}

function New-Conversation([string]$AgentName) {
  $payload = @{
    userId    = "demo-user"
    agentName = $AgentName
  } | ConvertTo-Json -Compress

  $response = Invoke-RestMethod -Method Post -Uri "$GatewayURL/api/conversations" -ContentType "application/json" -Body $payload -TimeoutSec ([Math]::Max(20, $TimeoutSeconds))
  $conversationID = [string]$response.id
  if ([string]::IsNullOrWhiteSpace($conversationID)) {
    throw "conversation id is empty"
  }
  return $conversationID
}

function Send-Chat([string]$ConversationID, [string]$AgentName, [string]$Message) {
  $payload = @{
    conversationId = $ConversationID
    message        = $Message
    agentName      = $AgentName
  } | ConvertTo-Json -Compress

  $response = Invoke-WebRequest -Method Post -Uri "$GatewayURL/api/chat" -ContentType "application/json" -Body $payload -TimeoutSec ([Math]::Max(30, $TimeoutSeconds))
  if ($response.StatusCode -ne 200) {
    throw "unexpected HTTP status: $($response.StatusCode)"
  }
  return [string]$response.Content
}

Write-Host "=== Smoke: New Architecture Compose Demo ==="
Write-Host "GatewayURL      : $GatewayURL"
Write-Host "CodeAgentURL    : $CodeAgentURL"
Write-Host "WebAgentURL     : $WebAgentURL"
Write-Host "TimeoutSeconds  : $TimeoutSeconds"
Write-Host "SkipAgentHealth : $SkipAgentHealth"

Test-DockerComposePreflight

if (-not $script:DockerPreflight.DockerDaemon) {
  Write-Warn "Docker daemon appears unavailable. If services are not started manually, smoke exit 1 is expected."
}

if (-not $SkipAgentHealth) {
  Invoke-Check "wait code-agent /health ready" {
    Test-HttpReady "code-agent /health" "$CodeAgentURL/health" {
      param($resp)
      $resp.StatusCode -eq 200 -and [string]$resp.Content -like '*"status":"ok"*'
    } | Out-Null
  }

  Invoke-Check "wait web-agent /health ready" {
    Test-HttpReady "web-agent /health" "$WebAgentURL/health" {
      param($resp)
      $resp.StatusCode -eq 200 -and [string]$resp.Content -like '*"status":"ok"*'
    } | Out-Null
  }
}

Invoke-Check "wait gateway /health ready" {
  Test-HttpReady "gateway /health" "$GatewayURL/health" {
    param($resp)
    $resp.StatusCode -eq 200 -and [string]$resp.Content -like '*"status":"ok"*'
  } | Out-Null
}

Invoke-Check "wait gateway /api/agents ready" {
  Test-HttpReady "gateway /api/agents" "$GatewayURL/api/agents" {
    param($resp)
    $body = [string]$resp.Content
    $resp.StatusCode -eq 200 -and $body -like "*code-agent*" -and $body -like "*web-agent*"
  } | Out-Null
}

if (-not $SkipAgentHealth) {
  Invoke-Check "code-agent /health" {
    $resp = Invoke-RestMethod -Method Get -Uri "$CodeAgentURL/health" -TimeoutSec 20
    if ($resp.status -ne "ok") {
      throw "unexpected status: $($resp.status)"
    }
  }

  Invoke-Check "web-agent /health" {
    $resp = Invoke-RestMethod -Method Get -Uri "$WebAgentURL/health" -TimeoutSec 20
    if ($resp.status -ne "ok") {
      throw "unexpected status: $($resp.status)"
    }
  }
}

Invoke-Check "gateway /health" {
  $resp = Invoke-RestMethod -Method Get -Uri "$GatewayURL/health" -TimeoutSec 20
  if ($resp.status -ne "ok") {
    throw "unexpected status: $($resp.status)"
  }
}

Invoke-Check "gateway /api/agents" {
  $response = Invoke-WebRequest -Method Get -Uri "$GatewayURL/api/agents" -TimeoutSec 20
  $content = [string]$response.Content
  Assert-ContainsAny $content @("code-agent") "code-agent not found in /api/agents"
  Assert-ContainsAny $content @("web-agent") "web-agent not found in /api/agents"
}

Invoke-Check "gateway /api/conversations" {
  $response = Invoke-WebRequest -Method Get -Uri "$GatewayURL/api/conversations?userId=demo-user" -TimeoutSec 20
  if ($response.StatusCode -ne 200) {
    throw "unexpected HTTP status: $($response.StatusCode)"
  }
}

Invoke-Check "gateway /api/chat agentName=code-agent" {
  $conversationID = New-Conversation "code-agent"
  $content = Send-Chat $conversationID "code-agent" "please generate a tiny go http server code sample"
  Assert-ContainsAny $content @("event: message", "event: message.delta") "missing message event in SSE"
  Assert-ContainsAny $content @('"author":"code-agent"', "code-agent v0.1 mock response", "```go") "missing code-agent response marker"
}

Invoke-Check "gateway /api/chat agentName=web-agent" {
  $conversationID = New-Conversation "web-agent"
  $content = Send-Chat $conversationID "web-agent" "please build a login html page with form and button"
  Assert-ContainsAny $content @("event: message", "event: message.delta") "missing message event in SSE"
  Assert-ContainsAny $content @('"author":"web-agent"', "web-agent v0.1 mock response", "web-agent-preview", "<section") "missing web-agent response marker"
}

Invoke-Check "gateway /api/chat agentName=unknown-agent returns safe error" {
  $conversationID = New-Conversation "code-agent"
  $content = Send-Chat $conversationID "unknown-agent" "test unknown agent routing"
  Assert-ContainsAny $content @("event: error") "expected SSE error event"
  Assert-NotContainsAny $content @("http://code-agent-new:8080", "http://web-agent-new:8080", "code-agent-new:8080", "web-agent-new:8080") "internal service URL leaked in unknown-agent error"
  if ($content -match "(?i)(token|stack|panic|sk-[A-Za-z0-9_-]{20,}|begin rsa|begin openssh)") {
    throw "unsafe unknown-agent error content detected"
  }
}

if ($script:HasFailure) {
  Write-Host "=== Smoke FAILED ===" -ForegroundColor Red
  exit 1
}

Write-Host "=== Smoke PASSED ===" -ForegroundColor Green
