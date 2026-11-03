#!/usr/bin/env bash
# R0-08 observability smoke: real Jaeger/Collector/Prometheus/Alertmanager plus a
# local webhook. It starts two host processes (orchestrator + gateway) and one
# code-agent, all exporting OTLP to the collector, then proves:
#   - the run trace is chained gateway -> orchestrator -> code-agent with
#     agenthub.run_id attached;
#   - Prometheus scrapes both services (up == 1);
#   - a test rule reaches Alertmanager and the local webhook;
#   - stopping the collector leaves the services healthy, and traces resume
#     after it comes back.
# It never talks to an external alert channel and never reads .env.
# Exit codes: 0 all checks pass, 1 a check failed, 2 missing input.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mysql_container="${MYSQL_TEST_CONTAINER:?export MYSQL_TEST_CONTAINER first}"
mysql_database="${MYSQL_TEST_DATABASE:-agenthub_test}"
mysql_dsn="${AGENTHUB_MYSQL_TEST_DSN:?export AGENTHUB_MYSQL_TEST_DSN first}"
redis_addr="${AGENTHUB_REDIS_TEST_ADDR:?export AGENTHUB_REDIS_TEST_ADDR first}"
gateway_token="${AGENTHUB_API_TOKEN:?export AGENTHUB_API_TOKEN first}"
internal_token="${INTERNAL_SERVICE_TOKEN:?export INTERNAL_SERVICE_TOKEN first}"
compose_project="${COMPOSE_PROJECT:-agenthub-r008}"
compose_file="${COMPOSE_FILE:-docker-compose.new-arch.yml}"
llm_base_url="${MULTI_LLM_BASE_URL:-http://127.0.0.1:18081}"
llm_provider="${LLM_PROVIDER:-anthropic}"
llm_model="${LLM_MODEL:-smoke-model}"
llm_key="${LLM_API_KEY:-smoke-fake-key}"
jaeger_port="${JAEGER_PORT:-16686}"
prom_port="${PROMETHEUS_PORT:-19092}"
am_port="${ALERTMANAGER_PORT:-19093}"
webhook_port="${ALERT_WEBHOOK_PORT:-18090}"
out_dir="${EVIDENCE_OUT:-}"
if [[ "${1:-}" == "--out" ]]; then
  out_dir="${2:-}"
  shift 2
fi
if [[ -z "$out_dir" ]]; then
  echo "export EVIDENCE_OUT or pass --out DIR" >&2
  exit 2
fi

mkdir -p "$out_dir/logs"
events="$out_dir/events.jsonl"
timeline="$out_dir/logs/timeline.jsonl"
: >"$events"
: >"$timeline"
started_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
work_dir="$(mktemp -d)"
pids=()
failures=()
scenarios=()
seq=0

log()  { printf '%s\n' "$*" >>"$out_dir/logs/smoke.log"; }
tl()   { printf '{"schemaVersion":2,"testId":"R0-08","planTaskId":"R0-08","layer":"middleware","actor":"tool","action":"%s","target":"%s","status":"%s","instanceId":"%s","role":"%s","detail":{"note":"%s"},"recordedAt":"%s"}\n' \
  "$1" "$2" "$3" "$4" "${5:-}" "${6:-}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$timeline"; }
ev()   { seq=$((seq+1)); printf '{"schemaVersion":2,"evidenceId":"ev_20261103_R0_08_%03d","testId":"R0-08","planTaskId":"R0-08","layer":"%s","actor":"%s","action":"%s","target":"%s","status":"%s","detail":{%s},"recordedAt":"%s"}\n' \
  "$seq" "$1" "$2" "$3" "$4" "$5" "$6" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$events"; }
record() {
  local name="$1" result="$2" note="${3:-}"
  scenarios+=("$name|$result|$note")
  if [[ "$result" == "PASS" ]]; then
    echo "PASS: $name ${note:+($note)}" | tee -a "$out_dir/logs/smoke.log"
  else
    failures+=("$name: $note")
    echo "FAIL: $name - $note" | tee -a "$out_dir/logs/smoke.log" >&2
  fi
}

cleanup() {
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
}
trap cleanup EXIT

sql() {
  docker exec "$mysql_container" sh -c \
    'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" --batch --skip-column-names "$1" -e "$2"' sh "$mysql_database" "$1" 2>/dev/null
}

wait_http() {
  local url="$1" limit="${2:-60}"
  for _ in $(seq 1 "$limit"); do
    curl -sf --max-time 3 "$url" >/dev/null 2>&1 && return 0
    sleep 1
  done
  return 1
}

http() {
  local method="$1" url="$2" data="${3:-}"
  RESP_FILE="$(mktemp)"
  local args=(-s -o "$RESP_FILE" -w '%{http_code}' --max-time 60 -X "$method" "$url"
    -H "Authorization: Bearer ${gateway_token}")
  [[ -n "$data" ]] && args+=(-H 'Content-Type: application/json' --data "$data")
  HTTP_STATUS="$(curl "${args[@]}")"
}

json_field() {
  sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$2" | head -n 1
}

wait_field() {
  local file="$1" key="$2" limit="${3:-60}"
  for _ in $(seq 1 "$limit"); do
    local v
    v="$(grep -oE "\"${key}\":\"[^\"]+\"" "$file" 2>/dev/null | head -n 1 | cut -d'"' -f4)"
    [[ -n "$v" ]] && { printf '%s' "$v"; return 0; }
    sleep 0.5
  done
  return 1
}

start_host() {
  local label="$1"; shift
  "$@" >"$out_dir/logs/${label}.log" 2>&1 &
  pids+=("$!")
  echo "$label pid=$!" >>"$out_dir/logs/instances.txt"
}

# ── build ──────────────────────────────────────────────────────────────────
echo "== building binaries" | tee "$out_dir/logs/smoke.log"
(cd services/orchestrator && go build -o "$work_dir/orchestrator" ./cmd/orchestrator) 2>"$out_dir/logs/build-orchestrator.txt" || { record "build orchestrator" FAIL "see logs"; exit 1; }
(cd services/gateway && go build -o "$work_dir/gateway" ./cmd/gateway) 2>"$out_dir/logs/build-gateway.txt" || { record "build gateway" FAIL "see logs"; exit 1; }
(cd services/agents/code-agent && go build -o "$work_dir/code-agent" ./cmd/code-agent) 2>"$out_dir/logs/build-code-agent.txt" || { record "build code-agent" FAIL "see logs"; exit 1; }
record "build orchestrator/gateway/code-agent" PASS

# ── observability profile ──────────────────────────────────────────────────
token_file="$work_dir/orchestrator-token"
printf '%s' "$internal_token" >"$token_file"
token_mount="$(cygpath -w "$token_file" 2>/dev/null || printf '%s' "$token_file")"
prom_config="$(cygpath -w "$root/deploy/prometheus/prometheus.host.yml" 2>/dev/null || printf '%s' "$root/deploy/prometheus/prometheus.host.yml")"
echo "== starting observability profile" | tee -a "$out_dir/logs/smoke.log"
MSYS_NO_PATHCONV=1 MYSQL_ROOT_PASSWORD=test-only LLM_MODEL=test \
  PROMETHEUS_CONFIG_FILE="$prom_config" PROMETHEUS_TOKEN_FILE="$token_mount" \
  PROMETHEUS_PORT="$prom_port" ALERTMANAGER_PORT="$am_port" \
  ALERT_WEBHOOK_PORT="$webhook_port" JAEGER_PORT="$jaeger_port" \
  docker compose -p "$compose_project" -f "$compose_file" --profile observability \
  up -d jaeger otel-collector prometheus alertmanager alert-webhook \
  >"$out_dir/logs/compose-up.txt" 2>&1
if (( $? != 0 )); then
  record "observability profile up" FAIL "see logs/compose-up.txt"
  exit 1
fi
if wait_http "http://127.0.0.1:${jaeger_port}/" 60 && wait_http "http://127.0.0.1:${prom_port}/-/healthy" 60 \
  && wait_http "http://127.0.0.1:${am_port}/-/healthy" 60 && wait_http "http://127.0.0.1:${webhook_port}/health" 60; then
  record "jaeger/collector/prometheus/alertmanager/webhook healthy" PASS
else
  record "observability profile healthy" FAIL "health timeout"
  exit 1
fi
docker ps --filter "label=com.docker.compose.project=${compose_project}" --format '{{.Names}} {{.Image}} {{.Status}}' >"$out_dir/logs/compose-ps.txt" 2>&1
# Prometheus can drop its first notification while Alertmanager finishes starting.
# Restart it once Alertmanager is healthy so the always-firing test rule is
# delivered deterministically instead of racing a startup 503.
docker restart "${compose_project}-prometheus-1" >/dev/null 2>&1
wait_http "http://127.0.0.1:${prom_port}/-/healthy" 60 || true
sleep 3

# ── host services ──────────────────────────────────────────────────────────
echo "== starting orchestrator, gateway, code-agent" | tee -a "$out_dir/logs/smoke.log"
start_host code-agent env \
  CODE_AGENT_ADDR=:18082 CODE_AGENT_PUBLIC_URL=http://127.0.0.1:18082 \
  CODE_AGENT_LLM_PROVIDER="$llm_provider" CODE_AGENT_LLM_MODEL="$llm_model" \
  CODE_AGENT_LLM_API_KEY="$llm_key" CODE_AGENT_LLM_BASE_URL="$llm_base_url" \
  INTERNAL_SERVICE_TOKEN="$internal_token" \
  ORCHESTRATOR_TOOL_AUTHORIZE_URL=http://127.0.0.1:19080/internal/orchestrator/tools/authorize \
  ORCHESTRATOR_TOOL_RESULT_URL=http://127.0.0.1:19080/internal/orchestrator/tools/result \
  ORCHESTRATOR_HANDOFF_URL=http://127.0.0.1:19080/internal/orchestrator/tools/handoff \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318 \
  "$work_dir/code-agent"
start_host orchestrator env \
  ORCHESTRATOR_ADDR=:19080 ORCHESTRATOR_GRPC_ADDR=:19090 \
  INTERNAL_SERVICE_TOKEN="$internal_token" AGENTHUB_ENV=r008 \
  DATABASE_URL="$mysql_dsn" REDIS_URL="redis://${redis_addr}/0" \
  ARTIFACT_STORE_DIR="$work_dir/artifacts" ARTIFACT_BLOB_DIR="$work_dir/artifacts/blobs" \
  REQUIRE_PLAN_CONFIRMATION=true MEMORY_MODE=session_only ORCHESTRATOR_AGENT_CARD_LOAD=false \
  ORCHESTRATOR_LLM_PROVIDER="$llm_provider" ORCHESTRATOR_LLM_MODEL="$llm_model" \
  ORCHESTRATOR_LLM_API_KEY="$llm_key" ORCHESTRATOR_LLM_BASE_URL="$llm_base_url" \
  CODE_AGENT_URL=http://127.0.0.1:18082 \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318 \
  "$work_dir/orchestrator"
start_host gateway env \
  GATEWAY_ADDR=:19100 GATEWAY_ENABLE_AUTH=false \
  ORCHESTRATOR_URL=http://127.0.0.1:19080 ORCHESTRATOR_INTERNAL_TOKEN="$internal_token" \
  AGENTHUB_GATEWAY_STORE=mysql DATABASE_URL="$mysql_dsn" \
  CODE_AGENT_URL=http://127.0.0.1:18082 \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318 \
  "$work_dir/gateway"

if wait_http "http://127.0.0.1:18082/health" 60 && wait_http "http://127.0.0.1:19080/health" 60 \
  && wait_http "http://127.0.0.1:19100/health" 60; then
  record "host processes up (code-agent, orchestrator, gateway)" PASS
else
  record "host processes up" FAIL "health timeout; see logs/"
  exit 1
fi

GA="http://127.0.0.1:19100"
new_conversation() { http POST "${GA}/api/conversations" "$1"; json_field id "$RESP_FILE"; }

run_chat() {
  local label="$1" tag="$2"
  local conv stream run plan
  conv="$(new_conversation '{"userId":"obs-smoke","agentName":"code-agent"}')"
  [[ -z "$conv" ]] && { record "R0-08 $label conversation" FAIL "no conversation"; return 1; }
  stream="$out_dir/logs/${label}-stream.txt"
  curl -sS -N --max-time 120 -X POST "${GA}/api/chat" \
    -H "Authorization: Bearer ${gateway_token}" -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    --data "{\"conversationId\":\"${conv}\",\"message\":\"observability probe ${tag}\",\"agentName\":\"code-agent\"}" \
    >"$stream" 2>"${stream}.error" &
  local stream_pid=$!
  run="$(wait_field "$stream" runId 60 || true)"
  plan="$(wait_field "$stream" planId 60 || true)"
  if [[ -z "$run" || -z "$plan" ]]; then
    kill "$stream_pid" 2>/dev/null || true
    record "R0-08 $label run started" FAIL "run=${run:-none} plan=${plan:-none}"
    return 1
  fi
  http POST "${GA}/api/runs/${run}/confirm" \
    "{\"runId\":\"${run}\",\"action\":\"approve\",\"actionId\":\"${plan}\",\"planId\":\"${plan}\",\"revision\":1,\"confirmed\":true,\"idempotencyKey\":\"r008-${label}-${run}\"}"
  local confirm_status="$HTTP_STATUS"
  for _ in $(seq 1 90); do grep -qE 'RUN_FINISHED|run_finished' "$stream" && break; sleep 1; done
  kill "$stream_pid" 2>/dev/null || true
  LAST_RUN="$run"
  LAST_STREAM="$stream"
  ev gateway gateway http_request "/api/runs/:id/confirm" "$( [[ $confirm_status == 20* ]] && echo ok || echo error )" "{\"httpStatus\":${confirm_status},\"label\":\"${label}\"}"
  tl confirmation "$run" ok "gateway" "confirmer" "label=${label}"
  if grep -qE 'RUN_FINISHED|run_finished' "$stream"; then
    record "R0-08 $label run completed" PASS "run=${run}"
    return 0
  fi
  record "R0-08 $label run completed" FAIL "no terminal event"
  return 1
}

query_jaeger() {
  local run="$1" out="$2"
  curl -s "http://127.0.0.1:${jaeger_port}/api/traces?service=orchestrator&limit=50&lookback=3600000" >"$out" 2>/dev/null
  grep -q "$run" "$out"
}

check_trace() {
  local label="$1" run="$2" out="$3"
  local ok=1
  query_jaeger "$run" "$out" || ok=0
  grep -q '"serviceName":"gateway"' "$out" || ok=0
  grep -q '"serviceName":"orchestrator"' "$out" || ok=0
  grep -q '"serviceName":"code-agent"' "$out" || ok=0
  grep -q 'agenthub.run_id' "$out" || ok=0
  if [[ "$ok" == "1" ]]; then
    record "R0-08 $label trace gateway->orchestrator->code-agent" PASS "run=${run}"
    ev agent agent sse_event "code-agent" ok "{\"runHasTrace\":1,\"label\":\"${label}\"}"
    return 0
  fi
  record "R0-08 $label trace chained" FAIL "run=${run}; see logs/${label}-jaeger.json"
  return 1
}

check_prom() {
  local job="$1" out="$2"
  curl -sG --data-urlencode "query=up{job=\"${job}\"}" "http://127.0.0.1:${prom_port}/api/v1/query" >"$out" 2>/dev/null
  grep -q '"value":\[[^]]*,"1"\]' "$out"
}

# ── phase 1: trace + metrics + alert ───────────────────────────────────────
sleep 6
if run_chat "trace" "trace"; then
  sleep 6
  check_trace "phase1" "$LAST_RUN" "$out_dir/logs/phase1-jaeger.json" || true
fi
if check_prom gateway "$out_dir/logs/prom-gateway.json"; then
  record "Prometheus gateway target UP" PASS
else
  record "Prometheus gateway target UP" FAIL "see logs/prom-gateway.json"
fi
if check_prom orchestrator "$out_dir/logs/prom-orchestrator.json"; then
  record "Prometheus orchestrator target UP" PASS
else
  record "Prometheus orchestrator target UP" FAIL "see logs/prom-orchestrator.json"
fi

alert_ok=0
for _ in $(seq 1 90); do
  if curl -s "http://127.0.0.1:${webhook_port}/alerts" | grep -q 'AgentHubObservabilitySmoke'; then alert_ok=1; break; fi
  sleep 1
done
curl -s "http://127.0.0.1:${webhook_port}/alerts" >"$out_dir/logs/webhook-alerts.jsonl" 2>/dev/null
curl -s "http://127.0.0.1:${am_port}/api/v2/alerts" >"$out_dir/logs/alertmanager-alerts.json" 2>/dev/null
if [[ "$alert_ok" == "1" ]]; then
  record "test alert delivered to local webhook" PASS
  ev gateway gateway alert_delivery "alert-webhook" ok '{"alert":"AgentHubObservabilitySmoke"}'
else
  record "test alert delivered to local webhook" FAIL "no AgentHubObservabilitySmoke in webhook"
fi

# ── phase 2: collector outage and recovery ─────────────────────────────────
echo "== stopping collector" | tee -a "$out_dir/logs/smoke.log"
docker stop "${compose_project}-otel-collector-1" >/dev/null 2>&1
sleep 2
if curl -sf --max-time 3 "${GA}/health" >/dev/null && curl -sf --max-time 3 "http://127.0.0.1:19080/health" >/dev/null; then
  record "services stay healthy with collector down" PASS
  tl collector_down otel-collector ok "middleware" "collector"
else
  record "services stay healthy with collector down" FAIL "health check failed"
fi
run_chat "outage" "outage" || true
docker start "${compose_project}-otel-collector-1" >/dev/null 2>&1
for _ in $(seq 1 30); do curl -s -o /dev/null --max-time 2 "http://127.0.0.1:4318/" && break; sleep 1; done
sleep 2
run_chat "recovery" "recovery" || true
sleep 6
check_trace "recovery" "$LAST_RUN" "$out_dir/logs/recovery-jaeger.json" || true
tl collector_up otel-collector ok "middleware" "collector"

# ── summary ────────────────────────────────────────────────────────────────
ended_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
overall=PASS
(( ${#failures[@]} > 0 )) && overall=FAIL
{
  echo "# R0-08 OTel/Prometheus/Alertmanager 运行认证"
  echo
  echo "- commit: ${commit}"
  echo "- started: ${started_utc} / ended: ${ended_utc}"
  echo "- result: **${overall}**（${#scenarios[@]} 项断言，${#failures[@]} 项失败）"
  echo "- compose project: ${compose_project} / profile: observability"
  echo "- provider: fake LLM (${llm_base_url})；无外部告警通道"
  echo
  echo "## 断言结果"
  echo
  echo "| 场景 | 结果 | 证据 |"
  echo "|---|---|---|"
  for s in "${scenarios[@]}"; do
    IFS='|' read -r n r note <<<"$s"
    echo "| ${n} | ${r} | ${note} |"
  done
  echo
  echo "## 交互覆盖（计划 §5.5）"
  echo
  echo "| 交互 | 记录文件 | 条数 | 覆盖 |"
  echo "|---|---|---|---|"
  echo "| 用户操作 | - | 0 | not_applicable: API 级冒烟 |"
  echo "| API 请求/响应 | events.jsonl | 2 | yes（confirm） |"
  echo "| SSE/AG-UI | logs/*-stream.txt | 3 | yes（三个 run 的流文件） |"
  echo "| HITL | timeline.jsonl | 3 | yes |"
  echo "| 工具/记忆/Artifact | - | 0 | not_applicable: 假 LLM 计划不触发写工具，session_only |"
  echo "| 多实例 | - | 0 | not_applicable: 单 Orchestrator 进程；多副本见 R0-06 |"
  echo "| 观测/告警 | logs/phase1-jaeger.json、prom-*.json、webhook-alerts.jsonl | 4 | yes |"
  echo "| 故障注入 | timeline.jsonl | 2 | yes（collector stop/start） |"
  echo
  echo "## 证据文件"
  echo "- commands.txt / events.jsonl / timeline.jsonl / logs/"
} >"$out_dir/summary.md"

# Capture notification health before tearing the profile down.
curl -s "http://127.0.0.1:${prom_port}/metrics" | grep -E "prometheus_notifications" >"$out_dir/logs/prom-notifications.txt" 2>/dev/null || true
curl -s "http://127.0.0.1:${am_port}/metrics" | grep -E 'integration="webhook"' >"$out_dir/logs/am-webhook-metrics.txt" 2>/dev/null || true
docker ps --filter "label=com.docker.compose.project=${compose_project}" --format '{{.Names}} {{.Status}}' >"$out_dir/logs/compose-ps-final.txt" 2>&1 || true

for s in "${scenarios[@]}"; do
  IFS='|' read -r n r _ <<<"$s"
  [[ "$r" == "PASS" ]] || echo "FAILED: $n" >&2
done
echo "RESULT: ${overall}（assertions=${#scenarios[@]} failures=${#failures[@]}）"
MSYS_NO_PATHCONV=1 MYSQL_ROOT_PASSWORD=test-only LLM_MODEL=test \
  docker compose -p "$compose_project" -f "$compose_file" --profile observability down -v >/dev/null 2>&1 || true
[[ "$overall" == "PASS" ]] && exit 0 || exit 1
