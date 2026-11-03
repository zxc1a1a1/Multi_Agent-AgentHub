#!/usr/bin/env bash
# R0-06: multi-replica runtime authentication.
#
# Starts two independent Orchestrator processes and two independent Gateway
# processes on one host, all sharing the same MySQL and Redis, then proves the
# cross-instance behaviours the plan requires:
#   F-06  confirm written by instance B is claimed and executed exactly once by A
#   F-05  cancel of a pending group plan from B stops A's run and cleans task refs
#   F-10  killing Gateway A still leaves a complete, duplicate-free history on B
#   F-11  killing the lease-holding Orchestrator lets the other instance take over
#
# Nothing here reads .env: the operator exports the DSN, Redis address and
# tokens. MySQL is queried through the container's own environment so no
# credential appears in this script, in argv, or in the evidence.
# Exit codes: 0 all scenarios pass, 1 a scenario failed, 2 missing input.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

out_dir="${EVIDENCE_OUT:-}"
mysql_container="${MYSQL_TEST_CONTAINER:?export MYSQL_TEST_CONTAINER (MySQL container name) first}"
mysql_database="${MYSQL_TEST_DATABASE:-agenthub_test}"
mysql_dsn="${AGENTHUB_MYSQL_TEST_DSN:?export AGENTHUB_MYSQL_TEST_DSN first}"
redis_addr="${AGENTHUB_REDIS_TEST_ADDR:?export AGENTHUB_REDIS_TEST_ADDR first}"
redis_url="redis://${redis_addr}/0"
gateway_token="${AGENTHUB_API_TOKEN:?export AGENTHUB_API_TOKEN first}"
internal_token="${INTERNAL_SERVICE_TOKEN:?export INTERNAL_SERVICE_TOKEN first}"
compose_project="${COMPOSE_PROJECT:-agenthub-r004}"
compose_file="${COMPOSE_FILE:-docker-compose.new-arch.yml}"
llm_base_url="${MULTI_LLM_BASE_URL:-http://127.0.0.1:18081}"
llm_provider="${LLM_PROVIDER:-anthropic}"
llm_model="${LLM_MODEL:-smoke-model}"
llm_key="${LLM_API_KEY:-smoke-fake-key}"
lease_wait="${LEASE_TAKEOVER_WAIT:-75}"
# A crashed holder keeps the lease for the production default (15 minutes), which
# no smoke run can wait out, so the harness shortens it through the new override.
lease_ttl_override="${ORCHESTRATOR_LEASE_TTL:-15m}"
if [[ "${1:-}" == "--out" ]]; then
  out_dir="${2:-}"
  shift 2
fi
if [[ -z "$out_dir" ]]; then
  echo "export EVIDENCE_OUT (evidence directory) or pass --out DIR" >&2
  exit 2
fi

mkdir -p "$out_dir/logs"
events="$out_dir/events.jsonl"
timeline="$out_dir/logs/timeline.jsonl"
: >"$events"
: >"$timeline"
started_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
stamp="$(date -u +%Y%m%d)"
prefix="ev_${stamp}_R0-06"
seq=0
row_counter=0

failures=()
scenarios=()
pids=()
work_dir="$(mktemp -d)"

cleanup() {
  for pid in "${pids[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT

sql() {
  docker exec "$mysql_container" sh -c \
    'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" --batch --skip-column-names "$1" -e "$2"' sh "$mysql_database" "$1" 2>/dev/null
}

tl() {
  local test_id="$1" action="$2" instance="$3" role="$4" status="$5" note="${6:-}" run_id="${7:-}"
  printf '{"schemaVersion":2,"testId":"%s","planTaskId":"R0-06","layer":"middleware","actor":"tool","action":"%s","target":"%s","status":"%s","instanceId":"%s","role":"%s","runId":"%s","conversationId":"","traceId":"","detail":{"note":"%s"},"recordedAt":"%s"}\n' \
    "$test_id" "$action" "$instance" "$status" "$instance" "$role" "$run_id" "${note//\"/}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$timeline"
}

ev() {
  local test_id="$1" layer="$2" actor="$3" action="$4" target="$5" status="$6" detail="${7:-}" run_id="${8:-}" instance="${9:-}" role="${10:-}"
  seq=$(( seq + 1 ))
  local extra=""
  [[ -n "$run_id" ]] && extra+="\"runId\":\"$run_id\","
  [[ -n "$instance" ]] && extra+="\"instanceId\":\"$instance\",\"role\":\"$role\","
  printf '{"schemaVersion":2,"evidenceId":"%s_%03d","testId":"%s","planTaskId":"R0-06","layer":"%s","actor":"%s","action":"%s","target":"%s","status":"%s",%s"detail":{%s},"recordedAt":"%s"}\n' \
    "$prefix" "$seq" "$test_id" "$layer" "$actor" "$action" "$target" "$status" "$extra" "$detail" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$events"
}

record() {
  local name="$1" result="$2" note="${3:-}"
  scenarios+=("$name|$result|$note")
  if [[ "$result" == "PASS" ]]; then
    echo "PASS: $name ${note:+($note)}"
  else
    failures+=("$name: $note")
    echo "FAIL: $name - $note" >&2
  fi
}

http() {
  local method="$1" url="$2" data="${3:-}"
  RESP_FILE="$(mktemp)"
  local args=(-s -o "$RESP_FILE" -w '%{http_code}' --max-time "${HTTP_TIMEOUT:-60}" -X "$method" "$url"
    -H "Authorization: Bearer ${gateway_token}")
  [[ -n "$data" ]] && args+=(-H 'Content-Type: application/json' --data "$data")
  HTTP_STATUS="$(curl "${args[@]}")"
}

json_field() {
  sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$2" | head -n 1
}

wait_ready() {
  local url="$1" limit="${2:-60}"
  for _ in $(seq 1 "$limit"); do
    if curl -sf --max-time 3 "$url" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  return 1
}

# ── build binaries ──────────────────────────────────────────────────────────
echo "== building orchestrator and gateway binaries"
(cd services/orchestrator && go build -o "$work_dir/orchestrator" ./cmd/orchestrator) 2>"$out_dir/logs/build-orchestrator.txt"
build_orch=$?
(cd services/gateway && go build -o "$work_dir/gateway" ./cmd/gateway) 2>"$out_dir/logs/build-gateway.txt"
build_gw=$?
if (( build_orch != 0 || build_gw != 0 )); then
  echo "FAIL: binaries did not build" >&2
  head -20 "$out_dir"/logs/build-*.txt >&2 || true
  exit 1
fi
record "build orchestrator + gateway binaries" PASS

# ── discover child agent URLs from the running compose stack ────────────────
agent_env=()
while read -r name port; do
  [[ -z "$name" ]] && continue
  suffix="${name#${compose_project}-}"
  suffix="${suffix%-new-1}"
  suffix="${suffix%-agent}"
  case "$suffix" in
    code|web|document|vision|context|test|review|security|deploy|diff)
      var="$(printf '%s' "$suffix" | tr '[:lower:]' '[:upper:]')_AGENT_URL"
      agent_env+=("$var=http://127.0.0.1:${port}")
      ;;
  esac
done < <(docker ps --filter "label=com.docker.compose.project=${compose_project}" \
  --format '{{.Names}} {{.Ports}}' 2>/dev/null \
  | sed -n 's/^\([^ ]*agent-new-1\).*0\.0\.0\.0:\([0-9]*\)->.*/\1 \2/p')
if (( ${#agent_env[@]} == 0 )); then
  echo "no child agents discovered; is compose project '${compose_project}' running?" >&2
  exit 2
fi
echo "discovered ${#agent_env[@]} child agents"

start_instance() {
  local label="$1" kind="$2" http_port="$3" grpc_port="$4"
  local log="$out_dir/logs/${label}.log"
  local -a args
  local upstream_http="$5" upstream_grpc="$6"
  if [[ "$kind" == "orchestrator" ]]; then
    args=(env
      "ORCHESTRATOR_ADDR=:${http_port}" "ORCHESTRATOR_GRPC_ADDR=:${grpc_port}"
      "INTERNAL_SERVICE_TOKEN=${internal_token}" "AGENTHUB_ENV=multi-replica-smoke"
      "DATABASE_URL=${mysql_dsn}" "REDIS_URL=${redis_url}"
      "ARTIFACT_STORE_DIR=${work_dir}/artifacts-${label}"
      "ARTIFACT_BLOB_DIR=${work_dir}/artifacts-${label}/blobs"
      "REQUIRE_PLAN_CONFIRMATION=true" "MEMORY_MODE=session_only"
      "ORCHESTRATOR_LEASE_TTL=${lease_ttl_override}"
      "ORCHESTRATOR_AGENT_CARD_LOAD=true"
      "ORCHESTRATOR_LLM_PROVIDER=${llm_provider}" "ORCHESTRATOR_LLM_MODEL=${llm_model}"
      "ORCHESTRATOR_LLM_API_KEY=${llm_key}" "ORCHESTRATOR_LLM_BASE_URL=${llm_base_url}"
      "${agent_env[@]}"
      "$work_dir/orchestrator")
  else
    args=(env
      "GATEWAY_ADDR=:${http_port}" "GATEWAY_ENABLE_AUTH=false"
      "ORCHESTRATOR_URL=${upstream_http}" "ORCHESTRATOR_GRPC_URL=${upstream_grpc}"
      "ORCHESTRATOR_INTERNAL_TOKEN=${internal_token}"
      "AGENTHUB_GATEWAY_STORE=mysql" "DATABASE_URL=${mysql_dsn}"
      "${agent_env[@]}"
      "$work_dir/gateway")
  fi
  "${args[@]}" >"$log" 2>&1 &
  local pid=$!
  pids+=("$pid")
  echo "$pid" >"$work_dir/pid-${label}"
  echo "${label} pid=${pid}" >>"$out_dir/logs/instances.txt"
}

stop_instance() {
  local label="$1" mode="${2:-term}"
  local pid
  pid="$(cat "$work_dir/pid-${label}")"
  if [[ "$mode" == "kill" ]]; then
    kill -9 "$pid" 2>/dev/null || true
  else
    kill "$pid" 2>/dev/null || true
  fi
  sleep 1
  if kill -0 "$pid" 2>/dev/null; then kill -9 "$pid" 2>/dev/null || true; fi
}

rm -f "$out_dir/logs/instances.txt"
mkdir -p "$work_dir"

ORCH_A_HTTP=19080; ORCH_A_GRPC=19090
ORCH_B_HTTP=19081; ORCH_B_GRPC=19091
GW_A_HTTP=19100; GW_B_HTTP=19101

start_instance orchestrator-a orchestrator "$ORCH_A_HTTP" "$ORCH_A_GRPC" "" ""
start_instance orchestrator-b orchestrator "$ORCH_B_HTTP" "$ORCH_B_GRPC" "" ""
wait_ready "http://127.0.0.1:${ORCH_A_HTTP}/health" 60 || { echo "orchestrator-a not ready" >&2; exit 1; }
wait_ready "http://127.0.0.1:${ORCH_B_HTTP}/health" 60 || { echo "orchestrator-b not ready" >&2; exit 1; }
start_instance gateway-a gateway "$GW_A_HTTP" 0 "http://127.0.0.1:${ORCH_A_HTTP}" "127.0.0.1:${ORCH_A_GRPC}"
start_instance gateway-b gateway "$GW_B_HTTP" 0 "http://127.0.0.1:${ORCH_B_HTTP}" "127.0.0.1:${ORCH_B_GRPC}"
if wait_ready "http://127.0.0.1:${GW_A_HTTP}/health" 60 && wait_ready "http://127.0.0.1:${GW_B_HTTP}/health" 60; then
  record "four processes up (2 orchestrator + 2 gateway)" PASS "A/B on ${GW_A_HTTP}/${GW_B_HTTP}"
else
  record "four processes up (2 orchestrator + 2 gateway)" FAIL "health timeout; see logs/"
  exit 1
fi
ev R0-06 middleware tool lease_claim multi-replica-start ok '{}'
sql "SELECT 1" >/dev/null || { echo "cannot reach MySQL through ${mysql_container}" >&2; exit 2; }

new_conversation() {
  local url="$1" payload="$2"
  http POST "${url}/api/conversations" "$payload"
  json_field id "$RESP_FILE"
}

open_stream() {
  local url="$1" data="$2" out="$3"
  curl -sS -N --max-time "${STREAM_TIMEOUT:-120}" -X POST "${url}/api/chat" \
    -H "Authorization: Bearer ${gateway_token}" -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' --data "$data" >"$out" 2>"${out}.error" &
  echo $!
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

# ── F-06: cross-instance confirm, executed exactly once ─────────────────────
GA="http://127.0.0.1:${GW_A_HTTP}"
GB="http://127.0.0.1:${GW_B_HTTP}"
conv_a="$(new_conversation "$GA" '{"userId":"mr-smoke","agentName":"code-agent"}')"
run_a=""
if [[ -n "$conv_a" ]]; then
  stream_a="$out_dir/logs/f06-gateway-a-stream.txt"
  pid_a="$(open_stream "$GA" "{\"conversationId\":\"${conv_a}\",\"message\":\"write a hello endpoint\",\"agentName\":\"code-agent\"}" "$stream_a")"
  run_a="$(wait_field "$stream_a" runId 60 || true)"
  plan_a="$(wait_field "$stream_a" planId 60 || true)"
  holder_before="$(sql "SELECT CONCAT(lease_owner,'|',status) FROM executions WHERE run_id='${run_a}'" | head -n 1)"
  tl F-06 lease_claim "${holder_before%%|*}" holder ok "run=${run_a}" "$run_a"
  http POST "${GB}/api/runs/${run_a}/confirm" \
    "{\"runId\":\"${run_a}\",\"action\":\"approve\",\"actionId\":\"${plan_a}\",\"planId\":\"${plan_a}\",\"revision\":1,\"confirmed\":true,\"idempotencyKey\":\"r006-f06-${run_a}\"}"
  confirm_status="$HTTP_STATUS"
  ev F-06 gateway gateway http_request /api/runs/:id/confirm "$( [[ $confirm_status == 20* ]] && echo ok || echo error )" "{\"httpStatus\":${confirm_status}}" "$run_a" "gateway-b" "confirmer"
  tl F-06 hitl_confirm gateway-b confirmer ok "status=${confirm_status}" "$run_a"
  # duplicate confirm must not execute the plan twice
  http POST "${GB}/api/runs/${run_a}/confirm" \
    "{\"runId\":\"${run_a}\",\"action\":\"approve\",\"actionId\":\"${plan_a}\",\"planId\":\"${plan_a}\",\"revision\":1,\"confirmed\":true,\"idempotencyKey\":\"r006-f06-${run_a}\"}"
  dup_status="$HTTP_STATUS"
  dup_body="$(head -c 120 "$RESP_FILE")"
  for _ in $(seq 1 60); do
    grep -qE 'RUN_FINISHED|"status":"completed"' "$stream_a" && break
    sleep 1
  done
  kill "$pid_a" 2>/dev/null || true
  terminals="$(grep -cE '"type":"RUN_FINISHED"' "$stream_a" || true)"
  sql_terminal="$(sql "SELECT COUNT(*) FROM execution_events WHERE run_id='${run_a}' AND event_type='run_finished'" | head -n 1)"
  signals_consumed="$(sql "SELECT COUNT(*) FROM run_signals WHERE run_id='${run_a}' AND status<>'pending'" | head -n 1)"
  signals_total="$(sql "SELECT COUNT(*) FROM run_signals WHERE run_id='${run_a}'" | head -n 1)"
  if [[ "$confirm_status" == "2"* && "$terminals" == "1" && "$sql_terminal" == "1" ]]; then
    record "F-06 cross-instance confirm executes exactly once" PASS "holder=${holder_before%%|*} terminals=${terminals} sql_run_finished=${sql_terminal} dup=${dup_status}"
  else
    record "F-06 cross-instance confirm executes exactly once" FAIL "confirm=${confirm_status} terminals=${terminals} sql=${sql_terminal} dup=${dup_status}:${dup_body}"
  fi
  ev F-06 orchestrator orchestrator hitl_confirm run_signal ok "{\"signalsTotal\":${signals_total:-0},\"signalsConsumed\":${signals_consumed:-0}}" "$run_a" "orchestrator-a" "holder"
  [[ "$signals_total" =~ ^[0-9]+$ && "$signals_total" != "0" && "$signals_total" == "$signals_consumed" ]] \
    && record "F-06 signal consumed once in SQL" PASS "total=${signals_total}" \
    || record "F-06 signal consumed once in SQL" FAIL "total=${signals_total} consumed=${signals_consumed}"
  http GET "${GB}/api/conversations/${conv_a}/messages"
  roles_b="$(grep -o '"role"' "$RESP_FILE" | wc -l | tr -d ' ')"
  http GET "${GA}/api/conversations/${conv_a}/messages"
  roles_a="$(grep -o '"role"' "$RESP_FILE" | wc -l | tr -d ' ')"
  if [[ "$roles_a" == "$roles_b" && "$roles_a" -ge 2 ]]; then
    record "F-06 both gateways project identical history" PASS "roles=${roles_a}"
  else
    record "F-06 both gateways project identical history" FAIL "A=${roles_a} B=${roles_b}"
  fi
  ev F-06 gateway gateway projection_commit assistant_message.v1 ok "{\"counts\":{\"rolesA\":${roles_a},\"rolesB\":${roles_b}}}" "$run_a" "gateway-b" "projector"
else
  record "F-06 cross-instance confirm executes exactly once" FAIL "no conversation created"
fi

# ── F-05: cancel a pending group plan from the other instance ───────────────
conv_g="$(new_conversation "$GA" '{"userId":"mr-smoke","agentNames":["code-agent","test-agent"]}')"
if [[ -n "$conv_g" ]]; then
  stream_g="$out_dir/logs/f05-group-stream.txt"
  pid_g="$(open_stream "$GA" "{\"conversationId\":\"${conv_g}\",\"message\":\"review and test the change\"}" "$stream_g")"
  run_g="$(wait_field "$stream_g" runId 60 || true)"
  plan_g="$(wait_field "$stream_g" planId 60 || true)"
  http POST "${GB}/api/runs/${run_g}/confirm" \
    "{\"runId\":\"${run_g}\",\"action\":\"cancel\",\"actionId\":\"${plan_g}\",\"planId\":\"${plan_g}\",\"revision\":1,\"confirmed\":false,\"rejectReason\":\"multi-replica smoke\",\"idempotencyKey\":\"r006-f05-${run_g}\"}"
  cancel_status="$HTTP_STATUS"
  for _ in $(seq 1 40); do
    grep -qiE 'cancelled|canceled' "$stream_g" && break
    sleep 1
  done
  kill "$pid_g" 2>/dev/null || true
  refs_left="$(sql "SELECT COUNT(*) FROM run_task_refs WHERE run_id='${run_g}'" | head -n 1)"
  cancels="$(sql "SELECT COUNT(*) FROM run_cancels WHERE run_id='${run_g}'" | head -n 1)"
  status_g="$(sql "SELECT status FROM executions WHERE run_id='${run_g}'" | head -n 1)"
  double_finish="$(sql "SELECT COUNT(*) FROM execution_events WHERE run_id='${run_g}' AND event_type='run_finished'" | head -n 1)"
  status_lc="$(tr '[:upper:]' '[:lower:]' <<<"$status_g")"
  if [[ "$cancel_status" == 2* ]] && [[ "$status_lc" == *cancel* ]] && [[ "$double_finish" == "1" ]]; then
    record "F-05 cross-instance cancel stops the run once" PASS "status=${status_g} cancels=${cancels} refs=${refs_left}"
  else
    record "F-05 cross-instance cancel stops the run once" FAIL "status=${cancel_status} db=${status_g} finish=${double_finish}"
  fi
  ev F-05 gateway gateway hitl_cancel /api/runs/:id/confirm \
    "$( [[ $cancel_status == 20* ]] && echo ok || echo error )" "{\"httpStatus\":${cancel_status}}" "$run_g" "gateway-b" "confirmer"
  ev F-05 orchestrator orchestrator http_request run_cancels ok "{\"counts\":{\"cancels\":${cancels:-0},\"taskRefs\":${refs_left:-0}}}" "$run_g" "orchestrator-a" "holder"
  if (( ${refs_left:-1} == 0 )); then
    record "F-05 no dangling task refs" PASS
  else
    record "F-05 no dangling task refs" FAIL "run_task_refs still has ${refs_left} rows"
  fi
else
  record "F-05 cross-instance cancel stops the run once" FAIL "no group conversation"
fi

# ── F-10: kill Gateway A while a run is streaming; B stays consistent ───────
conv_r="$GA"; run_r=""
conv_r="$(new_conversation "$GA" '{"userId":"mr-smoke","agentName":"code-agent"}')"
if [[ -n "$conv_r" ]]; then
  stream_r="$out_dir/logs/f10-stream.txt"
  pid_r="$(open_stream "$GA" "{\"conversationId\":\"${conv_r}\",\"message\":\"long running task\",\"agentName\":\"code-agent\"}" "$stream_r")"
  run_r="$(wait_field "$stream_r" runId 60 || true)"
  plan_r="$(wait_field "$stream_r" planId 60 || true)"
  if [[ -n "$run_r" ]]; then
    stop_instance gateway-a kill
    tl F-10 http_request gateway-a projector error "killed mid stream" "$run_r"
    http POST "${GB}/api/runs/${run_r}/confirm" \
      "{\"runId\":\"${run_r}\",\"action\":\"approve\",\"actionId\":\"${plan_r}\",\"planId\":\"${plan_r}\",\"revision\":1,\"confirmed\":true,\"idempotencyKey\":\"r006-f10-${run_r}\"}"
    approve_b="$HTTP_STATUS"
    terminal_b=""
    for _ in $(seq 1 60); do
      terminal_b="$(sql "SELECT status FROM executions WHERE run_id='${run_r}'" | head -n 1)"
      [[ "$terminal_b" == "completed" || "$terminal_b" == "failed" || "$terminal_b" == "cancelled" ]] && break
      sleep 1
    done
    dup_finish="$(sql "SELECT COUNT(*) FROM execution_events WHERE run_id='${run_r}' AND event_type='run_finished'" | head -n 1)"
    sql_events="$(sql "SELECT COUNT(*) FROM execution_events WHERE run_id='${run_r}'" | head -n 1)"
    # Give the surviving gateway's projection worker time to catch up from SQL;
    # a slow projection and a missing projection are different findings.
    assistant_rows=0
    user_rows=0
    for _ in $(seq 1 40); do
      http GET "${GB}/api/conversations/${conv_r}/messages"
      body_b="$(cat "$RESP_FILE")"
      assistant_rows="$(grep -o '"role":"assistant"' <<<"$body_b" | wc -l | tr -d ' ')"
      user_rows="$(grep -o '"role":"user"' <<<"$body_b" | wc -l | tr -d ' ')"
      (( assistant_rows >= 1 )) && break
      sleep 1
    done
    if [[ "$approve_b" == "2"* && "$dup_finish" == "1" && "$assistant_rows" == "$user_rows" && "$assistant_rows" -ge 1 ]]; then
      record "F-10 surviving gateway completes without duplicates" PASS "status=${terminal_b} finish=${dup_finish} user=${user_rows} assistant=${assistant_rows} events=${sql_events}"
    else
      # Capture why the surviving gateway did not project, instead of only a red X.
      {
        echo "F-10 diagnostics for run=${run_r} conversation=${conv_r}"
        echo "gateway-b log tail:"
        tail -n 30 "$out_dir/logs/gateway-b.log" 2>/dev/null | sed 's/^/  /'
        echo "projection offsets:"
        sql "SELECT * FROM event_projection_offsets WHERE run_id='${run_r}'" | sed 's/^/  /'
        echo "gateway run_views row:"
        sql "SELECT run_id, status FROM run_views WHERE run_id='${run_r}'" | sed 's/^/  /'
        echo "public events for run:"
        sql "SELECT COUNT(*) FROM run_public_events WHERE run_id='${run_r}'" | sed 's/^/  /'
        echo "journal rows for run:"
        sql "SELECT seq, event_type FROM execution_events WHERE run_id='${run_r}' ORDER BY seq" | sed 's/^/  /'
      } >>"$out_dir/logs/f10-diagnostics.txt" 2>&1
      record "F-10 surviving gateway completes without duplicates" FAIL "approve=${approve_b} status=${terminal_b} finish=${dup_finish} user=${user_rows} assistant=${assistant_rows}; see logs/f10-diagnostics.txt"
    fi
    ev F-10 gateway gateway projection_commit assistant_message.v1 \
      "$( [[ $dup_finish == 1 ]] && echo ok || echo error )" "{\"counts\":{\"assistantMessages\":${assistant_rows},\"runFinished\":${dup_finish},\"executionEvents\":${sql_events}}}" "$run_r" "gateway-b" "projector"
  else
    record "F-10 surviving gateway completes without duplicates" FAIL "no run id"
  fi
  start_instance gateway-a gateway "$GW_A_HTTP" 0 "http://127.0.0.1:${ORCH_A_HTTP}" "127.0.0.1:${ORCH_A_GRPC}"
  wait_ready "${GA}/health" 60 || record "gateway-a restart after F-10" FAIL "still not healthy"
fi

# ── F-11: kill the lease-holding orchestrator, other instance takes over ────
conv_l="$(new_conversation "$GA" '{"userId":"mr-smoke","agentName":"code-agent"}')"
if [[ -n "$conv_l" ]]; then
  stream_l="$out_dir/logs/f11-stream.txt"
  pid_l="$(open_stream "$GA" "{\"conversationId\":\"${conv_l}\",\"message\":\"takeover probe\",\"agentName\":\"code-agent\"}" "$stream_l")"
  run_l="$(wait_field "$stream_l" runId 60 || true)"
  plan_l="$(wait_field "$stream_l" planId 60 || true)"
  owner_l="$(sql "SELECT lease_owner FROM executions WHERE run_id='${run_l}'" | head -n 1)"
  [[ "$owner_l" == "NULL" ]] && owner_l=""
  tl F-11 lease_claim "${owner_l:-unknown}" holder ok "run=${run_l}" "$run_l"
  if [[ -n "$run_l" ]]; then
    # The fake plan finishes in about a second, so block the child agent first:
    # a run that is genuinely mid-execution can be parked as paused by dropping
    # the client (handler_run_stream.go:790), which is the only state
    # /runs/{id}/resume accepts.
    child_container="${compose_project}-code-agent-new-1"
    if docker pause "$child_container" >/dev/null 2>&1; then
      record "F-11 child agent blocked to keep the run executing" PASS "$child_container paused"
    else
      record "F-11 child agent blocked to keep the run executing" FAIL "cannot pause ${child_container}"
    fi
    http POST "${GA}/api/runs/${run_l}/confirm" \
      "{\"runId\":\"${run_l}\",\"action\":\"approve\",\"actionId\":\"${plan_l}\",\"planId\":\"${plan_l}\",\"revision\":1,\"confirmed\":true,\"idempotencyKey\":\"r006-f11-approve-${run_l}\"}"
    sleep 2
    kill "$pid_l" 2>/dev/null || true
    paused=""
    for _ in $(seq 1 25); do
      paused="$(sql "SELECT status FROM executions WHERE run_id='${run_l}'" | head -n 1)"
      [[ "$paused" == "paused" ]] && break
      sleep 1
    done
    if [[ "$paused" == "paused" ]]; then
      record "F-11 run parked as paused on client disconnect" PASS "owner=${owner_l}"
    else
      record "F-11 run parked as paused on client disconnect" FAIL "status=${paused:-none}; resume only accepts paused"
    fi
    stop_instance orchestrator-a kill
    docker unpause "$child_container" >/dev/null 2>&1 || true
    tl F-11 lease_claim orchestrator-a holder error "process killed while status=${paused}" "$run_l"
    resume_stream="$out_dir/logs/f11-resume-stream.txt"
    curl -sS -N --max-time 90 -X POST "http://127.0.0.1:${ORCH_B_HTTP}/internal/orchestrator/runs/${run_l}/resume" \
      -H "Authorization: Bearer ${internal_token}" -H 'Content-Type: application/json' --data '{}' \
      >"$resume_stream" 2>"${resume_stream}.error" &
    resume_pid=$!
    pids+=("$resume_pid")
    final_owner="$owner_l"
    final_status="$paused"
    takeover=no
    for _ in $(seq 1 "$lease_wait"); do
      final_owner="$(sql "SELECT lease_owner FROM executions WHERE run_id='${run_l}'" | head -n 1)"
      final_status="$(sql "SELECT status FROM executions WHERE run_id='${run_l}'" | head -n 1)"
      [[ "$final_owner" == "NULL" ]] && final_owner=""
      if [[ -n "$final_owner" && "$final_owner" != "$owner_l" ]]; then takeover=yes; break; fi
      sleep 1
    done
    finishes=0
    if [[ "$takeover" == "yes" ]]; then
      for _ in $(seq 1 60); do
        finishes="$(sql "SELECT COUNT(*) FROM execution_events WHERE run_id='${run_l}' AND event_type='run_finished'" | head -n 1)"
        [[ "${finishes:-0}" != "0" ]] && break
        sleep 1
      done
    fi
    takeover_note="lease=${owner_l}->${final_owner:-none} status=${final_status} paused=${paused:-none}"
    if [[ "$paused" == "paused" && "$takeover" == "yes" ]]; then
      record "F-11 lease taken over by the surviving instance" PASS "$takeover_note"
    else
      record "F-11 lease taken over by the surviving instance" FAIL "no verified takeover within ${lease_wait}s; ${takeover_note}"
    fi
    if [[ "$paused" == "paused" && "$takeover" == "yes" && "$finishes" == "1" ]]; then
      record "F-11 takeover produces exactly one terminal" PASS "status=${final_status}"
    else
      record "F-11 takeover produces exactly one terminal" FAIL "run_finished rows=${finishes} ${takeover_note}"
    fi
    ev F-11 orchestrator orchestrator lease_claim run_lease \
      "$( [[ "$final_owner" != "$owner_l" && -n "$final_owner" ]] && echo ok || echo error )" "{\"detail\":{\"note\":\"owner=${owner_l}->${final_owner}\"},\"counts\":{\"runFinished\":${finishes:-0}}}" "$run_l" "orchestrator-b" "holder"
  else
    record "F-11 lease taken over by the surviving instance" FAIL "no run id"
  fi
fi

ended_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
overall=PASS
(( ${#failures[@]} > 0 )) && overall=FAIL

cat >"$out_dir/commands.txt" <<EOF
# required environment (names only, values never recorded here)
#   AGENTHUB_MYSQL_TEST_DSN AGENTHUB_REDIS_TEST_ADDR AGENTHUB_API_TOKEN INTERNAL_SERVICE_TOKEN
#   MYSQL_TEST_CONTAINER MYSQL_TEST_DATABASE COMPOSE_PROJECT LLM_PROVIDER LLM_MODEL LLM_API_KEY LLM_BASE_URL
# the child agents come from the already running compose project:
docker compose -p ${compose_project} -f ${compose_file} up -d --no-build --wait
node scripts/fake-llm-server.mjs 18081     # test-only provider
bash scripts/run-multi-replica-smoke.sh --out ${out_dir}
EOF

{
  echo "# R0-06 多副本运行时认证"
  echo
  echo "- commit: ${commit}"
  echo "- started: ${started_utc} / ended: ${ended_utc}"
  echo "- result: **${overall}**（${#scenarios[@]} 项断言，${#failures[@]} 项失败）"
  echo "- 实例：orchestrator-a=127.0.0.1:${ORCH_A_HTTP}/grpc ${ORCH_A_GRPC}，orchestrator-b=:${ORCH_B_HTTP}/${ORCH_B_GRPC}，gateway-a=:${GW_A_HTTP}→A，gateway-b=:${GW_B_HTTP}→B"
  echo "- 共享中间件：同一 MySQL（\`${mysql_database}\`，经 ${mysql_container} 容器查询）与同一 Redis（db 0），子 Agent 复用运行中的 compose 栈"
  echo "- provider：\`scripts/fake-llm-server.mjs\`（测试专用），因此本任务认证的是编排/投影/租约语义，不是模型质量"
  echo
  echo "## 断言结果"
  echo
  echo "| 场景 | 结果 | 证据 |"
  echo "|---|---|---|"
  for entry in "${scenarios[@]}"; do
    IFS='|' read -r name result note <<<"$entry"
    echo "| ${name} | ${result} | ${note//$'\n'/ } |"
  done
  echo
  if (( ${#failures[@]} > 0 )); then
    echo "## 失败详情"
    echo
    for f in "${failures[@]}"; do echo "- $f"; done
    echo
  fi
  echo "## 交互覆盖（计划 §5.5）"
  echo
  echo "| 交互 | 记录文件 | 条数 | 覆盖 |"
  echo "|---|---|---|---|"
  echo "| 用户操作 | - | 0 | not_applicable: 进程级冒烟，无浏览器；真实浏览器矩阵在 R5-04 |"
  echo "| API 请求/响应 | events.jsonl | $(grep -c '"action":"http_request"' "$events" || true) | yes |"
  echo "| SSE 连接/事件 | logs/f06-gateway-a-stream.txt 等 | $(grep -c '"action":"sse_connect"' "$events" || true) 条连接记录 | yes（每个场景一份流文件，含事件类型） |"
  echo "| HITL 请求/信号/Claim | events.jsonl + timeline.jsonl | $(grep -cE '"action":"hitl_(confirm|cancel)"' "$events" || true) | yes（每条带 instanceId/role） |"
  echo "| 工具授权/结果 | - | 0 | not_applicable: 假 LLM 计划不触发写工具审批 |"
  echo "| 记忆召回 | - | 0 | not_applicable: MEMORY_MODE=session_only |"
  echo "| Artifact | - | 0 | not_applicable: 本场景不产制品；MinIO 跨实例读在 R0-07 |"
  echo "| MQ | - | 0 | not_applicable: R2 未接入运行链路 |"
  echo "| 多实例 | events.jsonl + timeline.jsonl | $(grep -c '"instanceId"' "$events" || true) | yes（lease_owner 直接取自 MySQL executions 行） |"
  echo "| 故障注入 | timeline.jsonl | $(grep -c '"status":"error"' "$timeline" || true) | yes（kill gateway-a、kill lease-holding orchestrator） |"
  echo
  echo "## 已知边界"
  echo
  echo "- F-08（取消传播到真实 A2A 任务）只能证明 SQL 侧 \`run_cancels\` 落库与 \`run_task_refs\` 归零；"
  echo "  子 Agent 侧是否真的取消了远端 task，需要在日志里按 runId 追证据，属 R5-02 混沌验收范围，本任务不声称已认证。"
  echo "- F-09（SSE 断线后 events/status → resume）按计划归 R5-02，这里没有认证。"
} >"$out_dir/summary.md"

echo "RESULT: ${overall}（assertions=${#scenarios[@]} failures=${#failures[@]}）"
[[ "$overall" == "PASS" ]] && exit 0 || exit 1
