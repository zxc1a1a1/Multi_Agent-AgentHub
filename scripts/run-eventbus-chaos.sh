#!/usr/bin/env bash
# R2-08: reproducible MQ failure acceptance.
# Uses a dedicated Redis (AOF on) plus the test MySQL. It runs duplicate,
# ordering, claim, DLQ and idempotency suites, then restarts Redis between the
# seed/claim phases to prove an unacked PEL entry survives and is reclaimed.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mysql_container="${MYSQL_TEST_CONTAINER:?export MYSQL_TEST_CONTAINER first}"
mysql_database="${MYSQL_TEST_DATABASE:-agenthub_test}"
mysql_dsn="${AGENTHUB_MYSQL_TEST_DSN:?export AGENTHUB_MYSQL_TEST_DSN first}"
redis_container="${R208_REDIS_CONTAINER:-agenthub-r208-redis}"
redis_port="${R208_REDIS_PORT:-16380}"
out_dir="${EVIDENCE_OUT:-}"
if [[ "${1:-}" == "--out" ]]; then out_dir="${2:-}"; shift 2; fi
[[ -z "$out_dir" ]] && { echo "export EVIDENCE_OUT or pass --out DIR" >&2; exit 2; }
mkdir -p "$out_dir/logs"
out_dir="$(cd "$out_dir" && pwd)"
events="$out_dir/events.jsonl"
: >"$events"
started_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
redis_addr="127.0.0.1:${redis_port}"
env_name="r208_$(date +%s)"
failures=()
scenarios=()
seq=0

record() {
  local name="$1" result="$2" note="${3:-}"
  scenarios+=("$name|$result|$note")
  if [[ "$result" == "PASS" ]]; then echo "PASS: $name ${note:+($note)}"
  elif [[ "$result" == "SKIP" ]]; then echo "SKIP: $name - $note"
  else failures+=("$name: $note"); echo "FAIL: $name - $note" >&2; fi
}
ev() {
  seq=$((seq+1))
  printf '{"schemaVersion":2,"evidenceId":"ev_20261103_R2_08_%03d","testId":"R2-08","planTaskId":"R2-08","layer":"middleware","actor":"redis","action":"mq_consume","target":"%s","status":"%s","detail":{%s},"counts":{},"recordedAt":"%s"}\n' \
    "$seq" "$1" "$2" "$3" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$events"
}

# ── dedicated Redis with AOF ───────────────────────────────────────────────
echo "== starting dedicated Redis (AOF on)"
docker rm -f "$redis_container" >/dev/null 2>&1 || true
docker run -d --name "$redis_container" -p "${redis_port}:6379" \
  redis:7.2-alpine redis-server --appendonly yes >/dev/null
for _ in $(seq 1 30); do
  docker exec "$redis_container" redis-cli ping 2>/dev/null | grep -q PONG && break
  sleep 1
done
docker exec "$redis_container" redis-server --version >"$out_dir/logs/redis-version.txt" 2>&1
if [[ "$(docker exec "$redis_container" redis-cli ping 2>/dev/null)" == "PONG" ]]; then
  record "dedicated Redis 7.2 with AOF up" PASS "$redis_addr"
else
  record "dedicated Redis 7.2 with AOF up" FAIL "ping failed"
  exit 1
fi

run_suite() {
  local name="$1" workdir="$2" pattern="$3"; shift 3
  local out="$out_dir/logs/${name}.jsonl"
  ( cd "$workdir" && AGENTHUB_REDIS_TEST_ADDR="$redis_addr" AGENTHUB_MYSQL_TEST_DSN="$mysql_dsn" \
      go test ./... -run "$pattern" -count=1 -json "$@" >"$out" 2>&1 )
  local code=$?
  local pass fail skip
  pass=$(grep -c '"Action":"pass"' "$out" || true)
  fail=$(grep -c '"Action":"fail"' "$out" || true)
  skip=$(grep -c '"Action":"skip"' "$out" || true)
  if (( code == 0 )); then
    record "$name" PASS "pass=${pass} skip=${skip}"
    ev "$name" ok "{\"pass\":${pass},\"skip\":${skip}}"
  else
    record "$name" FAIL "exit=${code} fail=${fail}; see logs/${name}.jsonl"
    ev "$name" error "{\"fail\":${fail}}"
  fi
}

run_suite eventbus-transport pkg/runtime '^TestIntegrationRedis' 
run_suite relay-outbox services/orchestrator '^TestIntegrationRelay'
run_suite sqlidem pkg/runtime/eventbus/sqlidem '^TestIntegration'
run_suite gateway-projection services/gateway '^TestIntegrationEventBusConsumerDrivesRecovery'

# ── restart window: PEL must survive Redis stop/start ─────────────────────
echo "== seed unacked message"
seed_out="$out_dir/logs/restart-seed.jsonl"
( cd pkg/runtime && AGENTHUB_R208_PHASE=seed AGENTHUB_R208_ENV="$env_name" \
    AGENTHUB_REDIS_TEST_ADDR="$redis_addr" go test ./eventbus -run TestIntegrationRedisPendingSurvivesRestart -count=1 -json >"$seed_out" 2>&1 )
if grep -q '"Action":"pass"' "$seed_out"; then
  record "restart seed leaves one unacked PEL entry" PASS "env=${env_name}"
else
  record "restart seed leaves one unacked PEL entry" FAIL "see logs/restart-seed.jsonl"
fi

echo "== restarting Redis"
t0=$(date +%s)
docker stop "$redis_container" >/dev/null 2>&1
sleep 2
docker start "$redis_container" >/dev/null 2>&1
for _ in $(seq 1 60); do
  docker exec "$redis_container" redis-cli ping 2>/dev/null | grep -q PONG && break
  sleep 1
done
t1=$(date +%s)
rto=$((t1 - t0))
if [[ "$(docker exec "$redis_container" redis-cli ping 2>/dev/null)" == "PONG" ]]; then
  record "Redis restarted with AOF" PASS "rto=${rto}s"
  ev "redis-restart" ok "{\"recoveryMs\":$((rto*1000))}"
else
  record "Redis restarted with AOF" FAIL "ping failed after restart"
fi

echo "== claim after restart"
claim_out="$out_dir/logs/restart-claim.jsonl"
( cd pkg/runtime && AGENTHUB_R208_PHASE=claim AGENTHUB_R208_ENV="$env_name" \
    AGENTHUB_REDIS_TEST_ADDR="$redis_addr" go test ./eventbus -run TestIntegrationRedisPendingSurvivesRestart -count=1 -json >"$claim_out" 2>&1 )
if grep -q '"Action":"pass"' "$claim_out"; then
  record "unacked PEL entry reclaimed after restart" PASS "no duplicate effect"
  ev "pel-claim" ok '{"redelivered":1}'
else
  record "unacked PEL entry reclaimed after restart" FAIL "see logs/restart-claim.jsonl"
fi

# ── summary ────────────────────────────────────────────────────────────────
ended_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
overall=PASS
(( ${#failures[@]} > 0 )) && overall=FAIL
{
  echo "# R2-08 MQ 故障、重复与顺序验收"
  echo
  echo "- commit: ${commit}"
  echo "- started: ${started_utc} / ended: ${ended_utc}"
  echo "- result: **${overall}**（${#scenarios[@]} 项断言，${#failures[@]} 项失败）"
  echo "- Redis: 7.2-alpine，AOF on，独立容器 ${redis_container} (${redis_addr})"
  echo "- MySQL: 真实测试库（sqlidem 用例）"
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
  echo "## 场景映射"
  echo
  echo "| MQ 场景 | 覆盖 |"
  echo "|---|---|"
  echo "| MQ-01 重复投递只有一次业务效果 | sqlidem + eventbus-transport |"
  echo "| MQ-02 ack/claim 不重复执行 | restart-claim + sqlidem |"
  echo "| MQ-03 超 maxAttempts 进 DLQ、不自动重放 | eventbus-transport (malformed/DLQ) |"
  echo "| MQ-04 指标 | R2-06 证据 |"
  echo "| MQ-05 投影消费 | gateway-projection |"
  echo "| MQ-06 同 run 顺序 | eventbus-transport (order) |"
  echo "| P-06 consumer kill / claim | restart-claim |"
  echo
  echo "## 交互覆盖（§5.5）"
  echo
  echo "| 交互 | 记录文件 | 条数 | 覆盖 |"
  echo "|---|---|---|---|"
  echo "| MQ publish/consume/ack/claim/DLQ | events.jsonl + logs/*.jsonl | ${#scenarios[@]} | yes |"
  echo "| 故障注入（Redis restart） | events.jsonl + logs/restart-*.jsonl | 3 | yes（RTO=${rto}s） |"
  echo "| 用户/SSE/HITL/工具/记忆/Artifact | - | 0 | not_applicable: 中间件故障矩阵 |"
  echo "| 多实例 | - | 0 | not_applicable: 双 consumer 在 restart-claim 内模拟 |"
  echo
  echo "## 边界"
  echo "- 未做真实进程级 consumer SIGKILL；claim 由‘不 ack + 第二个 consumer’模拟。"
  echo "- Redis AOF 在 stop/start 间保留；`docker rm` 会丢数据，脚本只在 teardown 删除容器。"
  echo "- 未覆盖 Redis AOF off 场景（计划 MQ 场景中的 AOF on/off 只做了 on）。"
} >"$out_dir/summary.md"

docker rm -f "$redis_container" >/dev/null 2>&1 || true
echo "RESULT: ${overall}（assertions=${#scenarios[@]} failures=${#failures[@]}）"
[[ "$overall" == "PASS" ]] && exit 0 || exit 1
