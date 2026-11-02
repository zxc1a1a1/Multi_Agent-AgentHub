#!/usr/bin/env bash
# R0-02 supplementary gate: prove that terminal run state and a resolved
# approval survive a real MySQL server restart.
#
# The MySQL root password is read from the container's own environment inside
# the container, so no credential ever appears in this command line, in the
# evidence files, or in the repository.
set -euo pipefail

container="${MYSQL_TEST_CONTAINER:?export MYSQL_TEST_CONTAINER (container name) first}"
ready_timeout="${MYSQL_READY_TIMEOUT:-120}"
out_dir="${EVIDENCE_OUT:?export EVIDENCE_OUT (evidence directory) first}"
tag="r0-02-restart-$(date -u +%s)"

mkdir -p "$out_dir"

mysql_in_container() {
  docker exec -i "$container" sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" --batch --skip-column-names "$@"' sh "$@"
}

wait_ready() {
  local deadline=$(( $(date -u +%s) + ready_timeout ))
  until docker exec "$container" sh -c 'mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" >/dev/null 2>&1
         mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SELECT 1" >/dev/null 2>&1' 2>/dev/null; do
    if (( $(date -u +%s) >= deadline )); then
      echo "TIMEOUT: mysql did not become ready within ${ready_timeout}s" >&2
      return 1
    fi
    sleep 2
  done
}

{
  echo "# R0-02 MySQL restart durability check"
  echo "# container: ${container} (credentials read from the container environment, never recorded)"
  echo "# started: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "# tag: ${tag}"
} >"$out_dir/restart-check.txt"

# 1. Write a terminal execution row plus a resolved approval through SQL.
mysql_in_container agenthub_test <<SQL
INSERT INTO executions (run_id, request_id, conversation_id, workspace_id, principal_id,
  execution_path, status, current_seq, fencing_version, terminal_reason)
  VALUES ('${tag}-run', '${tag}-req', '${tag}-conv', '${tag}-ws', '${tag}-principal',
          'main_agent_orchestration', 'completed', 3, 1, 'completed');
INSERT INTO approvals (approval_id, run_id, plan_id, plan_revision, workspace_id, principal_id,
  status, expires_at, idempotency_key)
  VALUES ('${tag}-appr', '${tag}-run', '${tag}-plan', 1, '${tag}-ws', '${tag}-principal',
          'approved', DATE_ADD(NOW(), INTERVAL 10 MINUTE), '${tag}-idem');
SQL

before="$(mysql_in_container agenthub_test -e \
  "SELECT CONCAT(e.status, '|', e.current_seq, '|', a.status) FROM executions e JOIN approvals a ON a.run_id = e.run_id WHERE e.run_id = '${tag}-run'")"
echo "before restart: ${before}" >>"$out_dir/restart-check.txt"

# 2. Restart the server for real.
echo "restarting ${container} at $(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$out_dir/restart-check.txt"
docker restart "$container" >/dev/null
wait_ready
echo "ready again at $(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$out_dir/restart-check.txt"

# 3. Read the same rows back.
after="$(mysql_in_container agenthub_test -e \
  "SELECT CONCAT(e.status, '|', e.current_seq, '|', a.status) FROM executions e JOIN approvals a ON a.run_id = e.run_id WHERE e.run_id = '${tag}-run'")"
echo "after restart: ${after}" >>"$out_dir/restart-check.txt"

mysql_in_container agenthub_test -e \
  "DELETE FROM approvals WHERE run_id = '${tag}-run'; DELETE FROM executions WHERE run_id = '${tag}-run';" >>"$out_dir/restart-check.txt"

expected="completed|3|approved"
if [[ "$before" != "$expected" || "$after" != "$expected" ]]; then
  {
    echo "RESULT: FAIL - expected '${expected}', before='${before}' after='${after}'"
    echo "note: InnoDB does not commit these rows until the transaction ends, so a restart must not lose them."
  } >>"$out_dir/restart-check.txt"
  echo "RESULT: FAIL (restart durability) expected='${expected}' before='${before}' after='${after}'" >&2
  exit 1
fi
echo "RESULT: PASS (terminal run state and resolved approval readable after MySQL restart)" >>"$out_dir/restart-check.txt"
echo "RESULT: PASS (restart durability)"
echo "- before: ${before}" >>"$out_dir/restart-check.txt"
echo "- after:  ${after}" >>"$out_dir/restart-check.txt"
