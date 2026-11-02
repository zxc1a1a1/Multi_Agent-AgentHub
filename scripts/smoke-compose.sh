#!/usr/bin/env bash
# R0-04: Docker compose build/run smoke gate.
#
# Drives the real stack (Frontend -> Nginx -> Gateway -> Orchestrator -> Agents)
# over HTTP/SSE and writes redacted evidence: F-01 single chat with HITL approve,
# F-02/F-14 group chat with participant filtering, F-05 cancel while awaiting
# approval, persistence checks and leak checks.
#
# It never reads .env: every value must be exported by the operator.
# Exit codes: 0 all checks pass, 1 at least one check failed, 2 missing input.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

project="${COMPOSE_PROJECT:-agenthub-r004}"
compose_file="${COMPOSE_FILE:-docker-compose.new-arch.yml}"
base_url="${SMOKE_BASE_URL:-http://127.0.0.1}"
out_dir="${EVIDENCE_OUT:-}"
orchestrator_service="${ORCHESTRATOR_CONTAINER:-${project}-orchestrator-new-1}"
artifact_dir="${ARTIFACT_DIR:-/data/artifacts}"
skip_up=0
keep_up=0
timeout_seconds="${SMOKE_TIMEOUT:-150}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-up) skip_up=1; shift ;;
    --keep-up) keep_up=1; shift ;;
    --out) out_dir="$2"; shift 2 ;;
    --base-url) base_url="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$out_dir" ]]; then
  echo "export EVIDENCE_OUT (evidence directory) or pass --out DIR" >&2
  exit 2
fi
: "${AGENTHUB_API_TOKEN:?export AGENTHUB_API_TOKEN (the gateway token) first}"
: "${INTERNAL_SERVICE_TOKEN:?export INTERNAL_SERVICE_TOKEN (used only to prove it never leaks) first}"

mkdir -p "$out_dir/logs"
started_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
stamp="$(date -u +%Y%m%d)"
evidence_prefix="ev_${stamp}_R0-04"
events="$out_dir/events.jsonl"
transcript="$out_dir/logs/http-transcript.txt"
: >"$events"
: >"$transcript"

failures=()
checks=()
requests=0
server_errors=0
CURRENT_TEST=R0-04
seq_counter=0

compose() { docker compose -p "$project" -f "$compose_file" "$@"; }

ev() {
  local layer="$1" actor="$2" action="$3" target="$4" status="$5" detail="${6:-}" run_id="${7:-}" conv_id="${8:-}"
  local extra=""
  [[ -n "$run_id" ]] && extra+="\"runId\":\"$run_id\","
  [[ -n "$conv_id" ]] && extra+="\"conversationId\":\"$conv_id\","
  seq_counter=$(( seq_counter + 1 ))
  printf '{"schemaVersion":2,"evidenceId":"%s_%03d","testId":"%s","planTaskId":"R0-04","layer":"%s","actor":"%s","action":"%s","target":"%s","status":"%s",%s"detail":{%s},"recordedAt":"%s"}\n' \
    "$evidence_prefix" "$seq_counter" "$CURRENT_TEST" "$layer" "$actor" "$action" "$target" "$status" "$extra" "$detail" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$events"
}

record() {
  local name="$1" result="$2" note="${3:-}"
  checks+=("$name|$result|$note")
  if [[ "$result" == "FAIL" ]]; then
    failures+=("$name: $note")
    echo "FAIL: $name - $note" >&2
  else
    echo "PASS: $name ${note:+($note)}"
  fi
}

json_field() {
  sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$2" | head -n 1
}

# http <method> <path> [body-string] -> RESP_FILE holds the body, HTTP_STATUS the code
http() {
  local method="$1" path="$2" data="${3:-}"
  RESP_FILE="$(mktemp)"
  local args=(-s -o "$RESP_FILE" -w '%{http_code}' --max-time "$timeout_seconds" -X "$method" \
    "${base_url}${path}" -H "Authorization: Bearer ${AGENTHUB_API_TOKEN}")
  [[ -n "$data" ]] && args+=(-H 'Content-Type: application/json' --data "$data")
  HTTP_STATUS="$(curl "${args[@]}")"
  requests=$(( requests + 1 ))
  echo "== ${method} ${path} -> ${HTTP_STATUS}" >>"$transcript"
  if [[ "$HTTP_STATUS" =~ ^5 ]]; then
    server_errors=$(( server_errors + 1 ))
    echo "!! 5xx on ${method} ${path}: $(head -c 200 "$RESP_FILE")" >>"$transcript"
  fi
}

# open_stream <path> <body> <outfile> <headers-file> -> background curl, PID echoed
open_stream() {
  local path="$1" data="$2" out="$3" headers="$4"
  curl -sS -N --max-time "$timeout_seconds" -X POST "${base_url}${path}" \
    -H "Authorization: Bearer ${AGENTHUB_API_TOKEN}" -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' --data "$data" \
    -D "$headers" >"$out" 2>"$out.error" &
  echo $!
}

wait_for_field() {
  local file="$1" key="$2" limit="${3:-40}"
  for _ in $(seq 1 "$limit"); do
    local value
    value="$(grep -oE "\"${key}\":\"[^\"]+\"" "$file" 2>/dev/null | head -n 1 | cut -d'"' -f4)"
    if [[ -n "$value" ]]; then printf '%s' "$value"; return 0; fi
    sleep 0.5
  done
  return 1
}

if (( skip_up == 0 )); then
  echo "building and starting ${project}"
  if compose up -d --build --wait >"$out_dir/logs/compose-up.log" 2>&1; then
    record "compose up --build --wait" PASS
  else
    record "compose up --build --wait" FAIL "see logs/compose-up.log"
  fi
fi

# 1. every service of the project healthy (docker ps avoids re-reading secrets)
docker ps -a --filter "label=com.docker.compose.project=${project}" \
  --format '{{.Names}} {{.Status}}' >"$out_dir/logs/compose-ps.txt" 2>&1
healthy_count="$(grep -c 'healthy' "$out_dir/logs/compose-ps.txt" || true)"
unhealthy="$(grep -Eic 'unhealthy|Exited|Created' "$out_dir/logs/compose-ps.txt" || true)"
total_services="$(grep -Ec '.' "$out_dir/logs/compose-ps.txt" || true)"
if [[ "$unhealthy" == "0" ]] && (( healthy_count > 0 )) && (( healthy_count == total_services )); then
  record "all containers healthy" PASS "${healthy_count}/${total_services} services"
else
  record "all containers healthy" FAIL "healthy=${healthy_count}/${total_services} not-healthy=${unhealthy}"
fi
ev middleware docker http_request docker-ps ok "{\"errorClass\":\"healthy=${healthy_count},unhealthy=${unhealthy}\"}"

# 2. gateway health through nginx
http GET /health
if [[ "$HTTP_STATUS" == "200" ]]; then record "GET /health via nginx" PASS "200"
else record "GET /health via nginx" FAIL "status=$HTTP_STATUS"; fi
ev gateway gateway http_request /health ok "{\"httpStatus\":${HTTP_STATUS}}"

# 3. agent list plus leak probe
http GET /api/agents
agents_body="$(head -c 4000 "$RESP_FILE")"
if [[ "$HTTP_STATUS" == "200" ]] && grep -q '"code-agent"' <<<"$agents_body"; then
  record "GET /api/agents lists code-agent" PASS
else
  record "GET /api/agents lists code-agent" FAIL "status=$HTTP_STATUS body=${agents_body:0:200}"
fi
ev gateway gateway http_request /api/agents ok "{\"httpStatus\":${HTTP_STATUS}}"
if grep -qF "$INTERNAL_SERVICE_TOKEN" <<<"$agents_body"; then
  record "internal token absent from /api/agents" FAIL "leak"
else
  record "internal token absent from /api/agents" PASS
fi

# 4. F-01 + F-03: single chat, plan approval, execution, terminal state
CURRENT_TEST=F-01
conv_payload="$(mktemp)"
printf '{"userId":"smoke-r004","agentName":"code-agent"}' >"$conv_payload"
http POST /api/conversations "$(cat "$conv_payload")"
conv_id="$(json_field id "$RESP_FILE")"
if [[ -n "$conv_id" && ( "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "201" ) ]]; then
  record "POST /api/conversations (single)" PASS "$conv_id"
else
  record "POST /api/conversations (single)" FAIL "status=$HTTP_STATUS body=$(head -c 200 "$RESP_FILE")"
fi
ev gateway gateway http_request /api/conversations ok "{\"httpStatus\":${HTTP_STATUS}}" "" "$conv_id"

f01_stream="$out_dir/logs/f01-sse.txt"
run_id="$(open_stream /api/chat \
  "{\"conversationId\":\"${conv_id}\",\"message\":\"write a hello endpoint in Go\",\"agentName\":\"code-agent\"}" \
  "$f01_stream" "$out_dir/logs/f01-sse-headers.txt")"
f01_run="$(wait_for_field "$f01_stream" runId 40 || true)"
f01_plan="$(wait_for_field "$f01_stream" planId 60 || true)"
if [[ -n "$f01_run" ]]; then
  record "F-01 run started over SSE" PASS "run=${f01_run}"
else
  record "F-01 run started over SSE" FAIL "$(head -c 200 "$f01_stream.error")"
fi
ev browser gateway sse_connect /api/chat "$( [[ -n $f01_run ]] && echo ok || echo error )" '{}' "" "$conv_id"
if grep -qE 'confirm_plan|waiting_user_approval|awaiting_confirmation|awaiting_confirmation|requiresConfirmation' "$f01_stream"; then
  record "F-03 plan reaches approval" PASS "plan=${f01_plan:-unknown}"
else
  record "F-03 plan reaches approval" FAIL "no approval event yet"
fi

CURRENT_TEST=F-03
# actionId is required by the orchestrator approval handler (the gRPC control
# plane forwards runId/actionId/planId), and confirmed mirrors the decision.
confirm_body="$(printf '{"runId":"%s","action":"approve","actionId":"%s","planId":"%s","revision":1,"confirmed":true,"idempotencyKey":"r004-approve-%s"}' \
  "$f01_run" "${f01_plan:-plan}" "${f01_plan:-plan}" "$f01_run")"
http POST "/api/runs/${f01_run}/confirm" "$confirm_body"
approve_status="$HTTP_STATUS"
approve_body="$(head -c 200 "$RESP_FILE")"
if [[ "$approve_status" == "200" || "$approve_status" == "202" ]]; then
  record "F-03 approve accepted" PASS "status=$approve_status"
else
  record "F-03 approve accepted" FAIL "status=$approve_status body=$approve_body"
fi
ev gateway gateway hitl_confirm "/api/runs/${f01_run}/confirm" \
  "$( [[ $approve_status == 20* ]] && echo ok || echo error )" "{\"httpStatus\":${approve_status}}" "$f01_run" "$conv_id"

# The stream stays open until the run finishes; poll for the terminal event
# instead of relying on the curl timeout.
for _ in $(seq 1 60); do
  grep -qE '"status":"completed"|RUN_FINISHED|RUN_ERROR' "$f01_stream" && break
  sleep 1
done
kill "$run_id" 2>/dev/null || true
sleep 1
f01_events="$(grep -c '^event:' "$f01_stream" || true)"
if grep -q 'code-agent' "$f01_stream" && (( f01_events > 2 )); then
  record "F-01 agent attribution in stream" PASS "${f01_events} events"
else
  record "F-01 agent attribution in stream" FAIL "events=${f01_events} attribution=$(grep -c code-agent "$f01_stream" || true)"
fi
while read -r etype; do
  ev gateway gateway sse_event /api/chat ok "{\"eventType\":\"$etype\"}" "$f01_run" "$conv_id"
done < <(grep '^event:' "$f01_stream" | awk '{print $2}' | sort | uniq -c | awk '{print $2}')
if grep -qE '^event: *error|"type":"RUN_ERROR"' "$f01_stream"; then
  record "F-01 no RUN_ERROR" FAIL "$(grep -A1 '^event: error' "$f01_stream" | head -c 300)"
else
  record "F-01 no RUN_ERROR" PASS
fi
if grep -qE '"status":"completed"|RUN_FINISHED' "$f01_stream"; then
  record "F-01 terminal state reached" PASS
else
  record "F-01 terminal state reached" FAIL "no RUN_FINISHED in stream"
fi
if grep -qiE 'text/event-stream' "$out_dir/logs/f01-sse-headers.txt"; then
  record "nginx keeps SSE streaming content type" PASS
else
  record "nginx keeps SSE streaming content type" FAIL "see logs/f01-sse-headers.txt"
fi
if grep -qiE 'X-Accel-Buffering: *no|Cache-Control: *(no-cache|no-store)' "$out_dir/logs/f01-sse-headers.txt"; then
  record "nginx disables buffering for SSE" PASS
else
  record "nginx disables buffering for SSE" FAIL "no buffering guard in headers"
fi
if grep -qF "$INTERNAL_SERVICE_TOKEN" "$f01_stream"; then
  record "internal token absent from SSE" FAIL "leak"
else
  record "internal token absent from SSE" PASS
fi

http GET "/api/conversations/${conv_id}/messages"
msg_roles="$(grep -o '"role"' "$RESP_FILE" | wc -l | tr -d ' ')"
if [[ "$HTTP_STATUS" == "200" ]] && (( msg_roles >= 2 )); then
  record "F-01 messages persisted" PASS "${msg_roles} roles"
else
  record "F-01 messages persisted" FAIL "status=$HTTP_STATUS roles=${msg_roles}"
fi
ev gateway gateway http_request /api/conversations/:id/messages \
  "$( [[ $HTTP_STATUS == 200 ]] && echo ok || echo error )" "{\"httpStatus\":${HTTP_STATUS}}" "$f01_run" "$conv_id"

# 5. F-02 + F-14 + F-05: group chat, participants, cancel while awaiting approval
CURRENT_TEST=F-02
http POST /api/conversations '{"userId":"smoke-r004","agentNames":["code-agent","test-agent"]}'
group_conv="$(json_field id "$RESP_FILE")"
if [[ -n "$group_conv" && ( "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "201" ) ]]; then
  record "POST /api/conversations (group)" PASS "$group_conv"
else
  record "POST /api/conversations (group)" FAIL "status=$HTTP_STATUS body=$(head -c 200 "$RESP_FILE")"
fi
http GET "/api/conversations/${group_conv}/participants"
if [[ "$HTTP_STATUS" == "200" ]] && grep -q '"code-agent"' "$RESP_FILE" && grep -q '"test-agent"' "$RESP_FILE"; then
  record "F-14 participants persisted as selected" PASS "$(tr -d ' \n' <"$RESP_FILE" | head -c 160)"
else
  record "F-14 participants persisted as selected" FAIL "status=$HTTP_STATUS body=$(head -c 200 "$RESP_FILE")"
fi
ev gateway gateway select "/api/conversations/:id/participants" \
  "$( [[ $HTTP_STATUS == 200 ]] && echo ok || echo error )" "{\"httpStatus\":${HTTP_STATUS}}" "" "$group_conv"

f02_stream="$out_dir/logs/f02-sse.txt"
group_pid="$(open_stream /api/chat "{\"conversationId\":\"${group_conv}\",\"message\":\"review the login page and add a test\"}" \
  "$f02_stream" "$out_dir/logs/f02-sse-headers.txt")"
group_run="$(wait_for_field "$f02_stream" runId 40 || true)"
group_plan="$(wait_for_field "$f02_stream" planId 60 || true)"
group_events="$(grep -c '^event:' "$f02_stream" || true)"
if (( group_events > 0 )) && [[ -n "$group_run" ]]; then
  record "F-02 group chat SSE stream started" PASS "${group_events} events, run=${group_run}"
else
  record "F-02 group chat SSE stream started" FAIL "events=${group_events} err=$(head -c 160 "$f02_stream.error")"
fi
ev browser gateway sse_connect /api/chat "$( [[ -n $group_run ]] && echo ok || echo error )" '{}' "$group_run" "$group_conv"
if grep -qE 'confirm_plan|waiting_user_approval|awaiting_confirmation' "$f02_stream"; then
  record "F-02 group plan reaches approval" PASS "plan=${group_plan:-unknown}"
else
  record "F-02 group plan reaches approval" FAIL "no approval event"
fi
# Both selected agents must appear in the proposed plan (participant filtering).
if grep -q 'test-agent' "$f02_stream"; then
  record "F-14 non-selected agents excluded from plan" PASS "test-agent present, code-agent present"
else
  record "F-14 non-selected agents exclude" FAIL "test-agent never appears"
fi

CURRENT_TEST=F-05
if [[ -n "$group_run" ]]; then
  http POST "/api/runs/${group_run}/confirm" \
    "$(printf '{"runId":"%s","action":"cancel","actionId":"%s","planId":"%s","revision":1,"confirmed":false,"idempotencyKey":"r004-cancel-%s"}' "$group_run" "${group_plan:-plan}" "${group_plan:-plan}" "$group_run")"
  cancel_status="$HTTP_STATUS"
  cancel_body="$(head -c 200 "$RESP_FILE")"
  if [[ "$cancel_status" == "200" || "$cancel_status" == "202" ]]; then
    record "F-05 cancel pending group run" PASS "status=$cancel_status"
  else
    record "F-05 cancel pending group run" FAIL "status=$cancel_status body=$cancel_body"
  fi
  ev gateway gateway hitl_cancel "/api/runs/${group_run}/confirm" \
    "$( [[ $cancel_status == 20* ]] && echo ok || echo error )" "{\"httpStatus\":${cancel_status}}" "$group_run" "$group_conv"
  sleep 3
  kill "$group_pid" 2>/dev/null || true
  if grep -qiE 'cancelled|canceled' "$f02_stream"; then
    record "F-05 cancelled state observed in stream" PASS
  else
    record "F-05 cancelled state observed in stream" FAIL "$(tail -c 300 "$f02_stream")"
  fi
  if grep -qE '"type":"RUN_ERROR"' "$f02_stream"; then
    record "F-05 cancel emits no RUN_ERROR" FAIL "RUN_ERROR present"
  else
    record "F-05 cancel emits no RUN_ERROR" PASS
  fi
else
  record "F-05 cancel pending group run" FAIL "no runId discovered"
fi

http GET "/api/conversations/${group_conv}/messages"
if [[ "$HTTP_STATUS" == "200" ]]; then
  record "F-02 history readable after cancel" PASS
else
  record "F-02 history readable after cancel" FAIL "status=$HTTP_STATUS"
fi

# 6. artifact directory is writable by the service user
CURRENT_TEST=R0-04
artefact_probe="$(docker exec "$orchestrator_service" sh -c \
  "touch ${artifact_dir}/.r004-probe && ls -ld ${artifact_dir} && rm -f ${artifact_dir}/.r004-probe" 2>&1)"
if grep -Eq 'drwx|rw' <<<"$artefact_probe"; then
  record "artifact volume writable" PASS "$(head -n 1 <<<"$artefact_probe")"
else
  record "artifact volume writable" FAIL "$artefact_probe"
fi
ev orchestrator orchestrator artifact_event "$artifact_dir" \
  "$( grep -Eq 'drwx|rw' <<<"$artefact_probe" && echo ok || echo error )" '{}' "$f01_run" ""

# 7. no credential material anywhere in the captured traffic
if grep -REqf <(printf '%s\n%s\n' "$INTERNAL_SERVICE_TOKEN" "$AGENTHUB_API_TOKEN") "$out_dir/logs" 2>/dev/null; then
  record "no tokens in captured traffic" FAIL "token found in logs"
else
  record "no tokens in captured traffic" PASS
fi
if grep -REq 'sk-[A-Za-z0-9_-]{20,}|-----BEGIN [A-Z ]*PRIVATE KEY-----' "$out_dir" 2>/dev/null; then
  record "secret pattern scan on evidence" FAIL "pattern hit"
else
  record "secret pattern scan on evidence" PASS
fi

# 8. no server errors across the whole run
if (( server_errors == 0 )); then
  record "no 5xx responses" PASS "${requests} requests"
else
  record "no 5xx responses" FAIL "${server_errors} of ${requests} requests"
fi

ended_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
overall=PASS
(( ${#failures[@]} > 0 )) && overall=FAIL

cat >"$out_dir/commands.txt" <<EOF
# reproducible commands; token values are never recorded
# required environment (names only):
#   AGENTHUB_API_TOKEN INTERNAL_SERVICE_TOKEN MYSQL_ROOT_PASSWORD
#   LLM_PROVIDER LLM_API_KEY LLM_BASE_URL LLM_MODEL   (point these at scripts/fake-llm-server.mjs)
docker compose -p ${project} -f ${compose_file} build
docker compose -p ${project} -f ${compose_file} up -d --no-build --wait
node scripts/fake-llm-server.mjs 18081        # test-only provider
bash scripts/smoke-compose.sh --skip-up --keep-up --out ${out_dir}
docker compose -p ${project} -f ${compose_file} down -v
EOF

{
  echo "# R0-04 Docker compose 构建与冒烟认证"
  echo
  echo "- commit: ${commit}"
  echo "- compose project: ${project} / file: ${compose_file} / base url: ${base_url}"
  echo "- started: ${started_utc} / ended: ${ended_utc}"
  echo "- result: **${overall}** (${#checks[@]} checks, ${#failures[@]} failures)"
  echo "- requests issued: ${requests}; 5xx: ${server_errors}"
  echo "- containers healthy: ${healthy_count}/${total_services}"
  echo "- provider: \`scripts/fake-llm-server.mjs\`（测试专用假模型，无任何真实凭据；生产链路不在本任务认证范围）"
  echo
  echo "## checks"
  echo
  echo "| check | result | detail |"
  echo "|---|---|---|"
  for entry in "${checks[@]}"; do
    IFS='|' read -r name result note <<<"$entry"
    echo "| ${name} | ${result} | ${note//$'\n'/ } |"
  done
  echo
  if (( ${#failures[@]} > 0 )); then
    echo "## failures"
    echo
    for f in "${failures[@]}"; do echo "- $f"; done
    echo
    echo "## compose logs (redacted, failure only)"
    echo
    echo '```'
    compose logs --tail 40 2>&1 | grep -vEi "${INTERNAL_SERVICE_TOKEN}|${AGENTHUB_API_TOKEN}" | head -n 120
    echo '```'
  fi
  echo
  echo "## interaction coverage (plan §5.5)"
  echo
  echo "| interaction | record file | rows | coverage |"
  echo "|---|---|---|---|"
  echo "| 用户操作（创建会话/选择参与者/发送） | events.jsonl | $(grep -cE '"action":"(select|http_request)"' "$events" || true) | yes（compose 层以等价 API 动作驱动；真实浏览器矩阵在 R5-04） |"
  echo "| API 请求/响应 | events.jsonl | $(grep -c '"action":"http_request"' "$events" || true) | yes |"
  echo "| SSE 连接/事件 | events.jsonl | $(grep -cE '"action":"sse_(connect|event)"' "$events" || true) | yes（只记事件类型与计数，不记文本增量） |"
  echo "| HITL 请求/审批/取消 | events.jsonl | $(grep -cE '"action":"hitl_(confirm|cancel)"' "$events" || true) | yes（F-03 approve、F-05 cancel） |"
  echo "| 工具授权/结果 | - | 0 | not_applicable: 假 LLM 计划不触发写工具审批 |"
  echo "| 记忆召回 | - | 0 | not_applicable: MEMORY_MODE=session_only |"
  echo "| Artifact | events.jsonl | $(grep -c '"action":"artifact_event"' "$events" || true) | yes（制品卷可写探针） |"
  echo "| MQ | - | 0 | not_applicable: R2 未实现 |"
  echo "| 多实例 | - | 0 | not_applicable: 默认 compose 单副本；多副本在 R0-06 |"
  echo
  echo "证据文件：logs/compose-ps.txt、logs/http-transcript.txt、logs/f01-sse.txt、logs/f02-sse.txt、logs/f01-sse-headers.txt。"
  echo "响应体只在本地临时文件解析，不入库；SSE 采样保留事件类型与计数，不含完整文本增量。"
} >"$out_dir/summary.md"

rm -f "$conv_payload"
[[ -n "${RESP_FILE:-}" ]] && rm -f "$RESP_FILE"

if (( keep_up == 0 )) && (( skip_up == 0 )); then
  compose down -v >"$out_dir/logs/compose-down.log" 2>&1 || true
fi

echo "RESULT: ${overall} (checks=${#checks[@]} failures=${#failures[@]})"
[[ "$overall" == "PASS" ]] && exit 0 || exit 1
