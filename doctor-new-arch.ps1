param(
  [string]$GatewayURL = "http://localhost:8080",
  [string]$CodeAgentURL = "http://localhost:8081",
  [string]$WebAgentURL = "http://localhost:8082",
  [string]$FrontendURL = "http://localhost:3000",
  [int]$TimeoutSeconds = 5
)

$ErrorActionPreference = "Stop"
if ($TimeoutSeconds -lt 1) {
  $TimeoutSeconds = 1
}

$script:PassCount = 0
$script:WarnCount = 0
$script:FailCount = 0

function Add-Pass([string]$Message) {
  $script:PassCount++
  Write-Host "[PASS] $Message" -ForegroundColor Green
}

function Add-Warn([string]$Message) {
  $script:WarnCount++
  Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Add-Fail([string]$Message) {
  $script:FailCount++
  Write-Host "[FAIL] $Message" -ForegroundColor Red
}

function Test-RequiredFile([string]$Path) {
  if (Test-Path -LiteralPath $Path) {
    Add-Pass "found required file: $Path"
    return
  }
  Add-Fail "missing required file: $Path"
}

function Test-Command([string]$Name) {
  $cmd = Get-Command $Name -ErrorAction SilentlyContinue
  if ($cmd) {
    Add-Pass "command available: $Name"
    return $true
  }
  Add-Warn "command not found: $Name"
  return $false
}

function Run-External([string]$Name, [string]$CommandLine, [switch]$FailOnError) {
  try {
    Invoke-Expression $CommandLine *> $null
    if ($LASTEXITCODE -eq 0) {
      Add-Pass "$Name succeeded"
      return $true
    }

    $message = "$Name failed with exit code $LASTEXITCODE"
    if ($FailOnError) {
      Add-Fail $message
    } else {
      Add-Warn $message
    }
    return $false
  } catch {
    $text = $_.Exception.Message
    if ($text -match "dockerDesktopLinuxEngine") {
      Add-Warn "$Name failed: dockerDesktopLinuxEngine pipe missing. Docker Desktop may not be running."
      return $false
    }
    if ($FailOnError) {
      Add-Fail "$Name failed: $text"
    } else {
      Add-Warn "$Name failed: $text"
    }
    return $false
  }
}

function Test-PortUsage([int]$Port) {
  $listeners = @()
  try {
    $listeners = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue
  } catch {
    $listeners = @()
  }

  if ($listeners -and $listeners.Count -gt 0) {
    $owners = @()
    foreach ($item in $listeners) {
      $pid = $item.OwningProcess
      if (-not $pid) {
        continue
      }
      try {
        $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
        if ($proc) {
          $owners += "$($proc.ProcessName)(PID=$pid)"
        } else {
          $owners += "PID=$pid"
        }
      } catch {
        $owners += "PID=$pid"
      }
    }
    if ($owners.Count -eq 0) {
      Add-Warn "port $Port is in use"
    } else {
      Add-Warn "port $Port is in use by: $($owners -join ', ')"
    }
    return
  }

  Add-Pass "port $Port is free"
}

function Test-HttpOptional([string]$Name, [string]$Url, [string]$Expected) {
  try {
    $resp = Invoke-WebRequest -Method Get -Uri $Url -TimeoutSec $TimeoutSeconds
    if ($resp.StatusCode -lt 200 -or $resp.StatusCode -ge 400) {
      Add-Warn "$Name reachable but returned HTTP $($resp.StatusCode)"
      return
    }

    if ($Expected -and ([string]$resp.Content -notlike "*$Expected*")) {
      Add-Warn "$Name reachable but expected marker '$Expected' not found"
      return
    }
    Add-Pass "$Name reachable"
  } catch {
    Add-Warn "$Name not reachable (service may be stopped): $($_.Exception.Message)"
  }
}

function Test-SecurityBoundaries {
  $composePath = "docker-compose.new-arch.yml"
  $nginxPath = "frontend/nginx.conf"
  $dockerfiles = @(
    "services/gateway/Dockerfile",
    "services/agents/code-agent/Dockerfile",
    "services/agents/web-agent/Dockerfile",
    "frontend/Dockerfile"
  )

  if (Test-Path -LiteralPath $composePath) {
    $compose = Get-Content -LiteralPath $composePath -Raw
    if ($compose -match "(?im)^\s*env_file\s*:") {
      Add-Fail "compose security risk: env_file is not allowed"
    } else {
      Add-Pass "compose security check: no env_file"
    }
  }

  foreach ($file in $dockerfiles) {
    if (-not (Test-Path -LiteralPath $file)) {
      continue
    }
    $raw = Get-Content -LiteralPath $file -Raw
    if ($raw -match "(?im)^\s*copy\s+.*\.env") {
      Add-Fail "dockerfile security risk: $file copies .env"
    } else {
      Add-Pass "dockerfile security check: $file does not copy .env"
    }
  }

  if (Test-Path -LiteralPath $nginxPath) {
    $nginx = Get-Content -LiteralPath $nginxPath -Raw
    if ($nginx -match "proxy_pass\s+http://code-agent-new:8080" -or $nginx -match "proxy_pass\s+http://web-agent-new:8080") {
      Add-Fail "nginx security boundary risk: frontend must not proxy directly to child agent"
    } else {
      Add-Pass "nginx security boundary check passed"
    }
  }
}

Write-Host "=== Doctor: New Architecture Demo Environment ==="
Write-Host "GatewayURL     : $GatewayURL"
Write-Host "CodeAgentURL   : $CodeAgentURL"
Write-Host "WebAgentURL    : $WebAgentURL"
Write-Host "FrontendURL    : $FrontendURL"
Write-Host "TimeoutSeconds : $TimeoutSeconds"

Write-Host "== Required Files =="
$requiredFiles = @(
  "docker-compose.new-arch.yml",
  "smoke-new-arch.ps1",
  "services/gateway/Dockerfile",
  "services/agents/code-agent/Dockerfile",
  "services/agents/web-agent/Dockerfile",
  "frontend/Dockerfile",
  "frontend/nginx.conf"
)
foreach ($file in $requiredFiles) {
  Test-RequiredFile $file
}

Write-Host "== Security Boundary Checks =="
Test-SecurityBoundaries

Write-Host "== Docker / Compose =="
$dockerExists = Test-Command "docker"
if ($dockerExists) {
  $dockerVersionOk = Run-External "docker version" "docker version" -FailOnError:$false
  $composeVersionOk = Run-External "docker compose version" "docker compose version" -FailOnError

  if ($composeVersionOk) {
    [void](Run-External "docker compose config" "docker compose -f docker-compose.new-arch.yml config" -FailOnError)
  } else {
    Add-Fail "docker compose config skipped because compose plugin is unavailable"
  }

  if (-not $dockerVersionOk) {
    Add-Warn "Docker Desktop may not be running. This is an environment issue, not a code failure."
  }
} else {
  Add-Fail "docker compose config check cannot run because docker command is missing"
}

Write-Host "== Port Checks =="
foreach ($port in @(8080, 8081, 8082, 3000)) {
  Test-PortUsage $port
}

Write-Host "== Optional Reachability Checks =="
Test-HttpOptional "code-agent /health" "$CodeAgentURL/health" '"status":"ok"'
Test-HttpOptional "web-agent /health" "$WebAgentURL/health" '"status":"ok"'
Test-HttpOptional "gateway /health" "$GatewayURL/health" '"status":"ok"'
Test-HttpOptional "frontend /" "$FrontendURL/" ""

Write-Host "== Summary =="
Write-Host "PASS: $($script:PassCount)"
Write-Host "WARN: $($script:WarnCount)"
Write-Host "FAIL: $($script:FailCount)"

if ($script:FailCount -gt 0) {
  Write-Host "OVERALL: FAIL" -ForegroundColor Red
  exit 1
}
if ($script:WarnCount -gt 0) {
  Write-Host "OVERALL: WARN" -ForegroundColor Yellow
  exit 0
}

Write-Host "OVERALL: PASS" -ForegroundColor Green
exit 0
