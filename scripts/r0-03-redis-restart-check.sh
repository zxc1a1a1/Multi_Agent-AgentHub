#!/usr/bin/env bash
# R0-03 supplementary gate: prove the Redis test instance keeps a run event
# stream across a real server restart (AOF), so relay recovery only has to deal
# with data loss, not with restart replay.
#
# Commands run inside the container, so no address or credential is written to
# the evidence files.
set -euo pipefail

container="${REDIS_TEST_CONTAINER:?export REDIS_TEST_CONTAINER (container name) first}"
out_dir="${EVIDENCE_OUT:?export EVIDENCE_OUT (evidence directory) first}"
db_index="${REDIS_TEST_DB:-3}"
key="agenthub:test:ws:run:restart-check:events"
ready_timeout="${REDIS_READY_TIMEOUT:-60}"

mkdir -p "$out_dir"

rediscli() {
  docker exec "$container" redis-cli -n "$db_index" "$@"
}

{
  echo "# R0-03 Redis restart durability check"
  echo "# container: ${container} db: ${db_index}"
  echo "# started: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
} >"$out_dir/redis-restart-check.txt"

docker exec "$container" redis-cli FLUSHDB >/dev/null 2>&1 || true
rediscli DEL "$key" >/dev/null

entry_id="$(rediscli XADD "$key" '*' runId restart-check seq 1 type state_update payload '{"seq":1}')"
echo "xadd id before restart: ${entry_id}" >>"$out_dir/redis-restart-check.txt"
rediscli CONFIG GET appendonly >>"$out_dir/redis-restart-check.txt" 2>&1 || true
before_len="$(rediscli XLEN "$key")"
echo "stream length before restart: ${before_len}" >>"$out_dir/redis-restart-check.txt"

echo "restarting ${container} at $(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$out_dir/redis-restart-check.txt"
docker restart "$container" >/dev/null

deadline=$(( $(date -u +%s) + ready_timeout ))
until docker exec "$container" redis-cli PING 2>/dev/null | grep -q PONG; do
  if (( $(date -u +%s) >= deadline )); then
    echo "RESULT: FAIL - redis did not answer PING within ${ready_timeout}s" >>"$out_dir/redis-restart-check.txt"
    echo "FAIL: redis not ready" >&2
    exit 1
  fi
  sleep 1
done
echo "ready again at $(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$out_dir/redis-restart-check.txt"

after_len="$(rediscli XLEN "$key")"
echo "stream length after restart: ${after_len}" >>"$out_dir/redis-restart-check.txt"
rediscli XRANGE "$key" "-" "+" >>"$out_dir/redis-restart-check.txt"
rediscli DEL "$key" >/dev/null
docker exec "$container" redis-cli FLUSHDB >/dev/null 2>&1 || true

if [[ "$before_len" != "1" || "$after_len" != "1" ]]; then
  {
    echo "RESULT: FAIL - AOF persistence across restart not confirmed"
    echo "note: if the test instance runs without appendonly, recovery must come from the SQL outbox instead,"
    echo "which is covered by TestIntegrationRelayRebuildsStreamFromSQLAfterDataLoss."
  } >>"$out_dir/redis-restart-check.txt"
  echo "FAIL: redis restart lost the stream (before=${before_len} after=${after_len})" >&2
  exit 1
fi
echo "RESULT: PASS (stream survived redis restart with AOF)" >>"$out_dir/redis-restart-check.txt"
echo "PASS: redis restart durability (AOF)"
