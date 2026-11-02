#!/usr/bin/env bash
# R0-01: integration gate with structured evidence collection.
#
# Runs selected TestIntegration suites against middleware that the caller must
# export explicitly, then writes redacted evidence (commands.txt, events.jsonl,
# summary.md). This script never starts containers, never reads .env or
# credential files, and never prints environment variable values.
#
# Exit codes: 0 PASS, 1 other failure, 2 missing environment, 3 SKIP present,
# 4 no PASS.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

test_id="GATE"
plan_task="R0-01"
packages=("./services/orchestrator/..." "./services/gateway/...")
run_pattern='^TestIntegration'
out_dir=""
go_timeout="20m"
required_vars=("AGENTHUB_MYSQL_TEST_DSN" "AGENTHUB_REDIS_TEST_ADDR")

usage() {
  cat <<'EOF'
usage: run-integration-evidence.sh [--test-id ID] [--plan-task ID] [--packages "P1 P2"]
                                   [--run REGEX] [--out DIR] [--timeout DUR]
                                   [--require-env VAR1,VAR2]

  --test-id     test identifier written into every evidence row (e.g. F-03)
  --plan-task   plan task identifier (e.g. R0-02)
  --packages    space separated go package patterns, repeated allowed
  --run         go test -run regex
  --out         evidence output directory (required)
  --timeout     go test -timeout value
  --require-env comma separated env var names that must be exported
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --test-id) test_id="$2"; shift 2 ;;
    --plan-task) plan_task="$2"; shift 2 ;;
    --packages)
      read -r -a picked <<<"$2"
      packages=("${picked[@]}")
      shift 2 ;;
    --package) packages=("${packages[@]}" "$2"); shift 2 ;;
    --run) run_pattern="$2"; shift 2 ;;
    --out) out_dir="$2"; shift 2 ;;
    --timeout) go_timeout="$2"; shift 2 ;;
    --require-env)
      IFS=',' read -r -a required_vars <<<"$2"
      shift 2 ;;
    -h | --help)
      usage
      exit 0 ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1 ;;
  esac
done

if [[ -z "$out_dir" ]]; then
  echo "error: --out is required" >&2
  usage >&2
  exit 2
fi

# Environment gate: presence only, names only, never values.
missing=()
for name in "${required_vars[@]}"; do
  if [[ -z "${!name:-}" ]]; then
    missing+=("$name")
  fi
done
if (( ${#missing[@]} > 0 )); then
  {
    echo "missing required environment variables (names only):"
    printf '  %s\n' "${missing[@]}"
    echo "export them in the calling shell before running this gate"
  } >&2
  exit 2
fi

mkdir -p "$out_dir/logs"
started_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
started_epoch="$(date -u +%s)"
commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
go_version="$(go version 2>/dev/null || echo unknown)"
stamp="$(date -u +%Y%m%d)"
evidence_prefix="ev_${stamp}_${test_id//[^A-Za-z0-9]/_}"

raw_jsonl="$out_dir/logs/go-test-raw.jsonl"
redacted_jsonl="$out_dir/logs/go-test.jsonl"
interactions="$out_dir/interactions.jsonl"
rm -f "$raw_jsonl" "$redacted_jsonl" "$interactions"

lit_repl='
function litrepl(s, from, to,   out, p) {
  out = ""
  while (from != "" && (p = index(s, from)) > 0) {
    out = out substr(s, 1, p - 1) to
    s = substr(s, p + length(from))
  }
  return out s
}
'

# Redaction happens inside awk via ENVIRON so no secret ever reaches argv.
redact_program="${lit_repl}
BEGIN {
  dsn = ENVIRON[\"AGENTHUB_MYSQL_TEST_DSN\"]
  addr = ENVIRON[\"AGENTHUB_REDIS_TEST_ADDR\"]
}
{
  line = \$0
  if (dsn != \"\") line = litrepl(line, dsn, \"<redacted-dsn>\")
  if (addr != \"\") line = litrepl(line, addr, \"<redacted-addr>\")
  print line
}"

parse_program='
function jesc(s,   out, i, c) {
  out = ""
  for (i = 1; i <= length(s); i++) {
    c = substr(s, i, 1)
    if (c == "\\") out = out "\\\\"
    else if (c == "\"") out = out "\\\""
    else out = out c
  }
  return out
}
{
  line = $0
  pos = index(line, "\"Action\":\"")
  if (pos == 0) next
  rest = substr(line, pos + 10)
  q = index(rest, "\"")
  if (q == 0) next
  action = substr(rest, 1, q - 1)
  if (action != "run" && action != "pass" && action != "fail" && action != "skip") next
  outp = index(line, "\"Output\":\"")
  pkg = ""
  p = index(line, "\"Package\":\"")
  if (p > 0 && (outp == 0 || p < outp)) {
    r = substr(line, p + 11); qq = index(r, "\""); pkg = substr(r, 1, qq - 1)
  }
  tst = ""
  p = index(line, "\"Test\":\"")
  if (p > 0 && (outp == 0 || p < outp)) {
    r = substr(line, p + 8); qq = index(r, "\""); tst = substr(r, 1, qq - 1)
  }
  if (tst == "") next
  key = pkg "|" tst
  if (!(key in seen)) { seen[key] = 1; order[++n] = key; pkgof[key] = pkg; tsof[key] = tst }
  elapsed = ""
  p = index(line, "\"Elapsed\":")
  if (p > 0) {
    r = substr(line, p + 10)
    if (substr(r, 1, 1) == "\"") { r = substr(r, 2); qq = index(r, "\""); elapsed = substr(r, 1, qq - 1) }
    else {
      qq = index(r, ","); qb = index(r, "}")
      if (qq == 0) qq = qb
      elapsed = substr(r, 1, qq - 1)
    }
  }
  if (action == "run") { if (!(key in result)) result[key] = "running" }
  else {
    result[key] = action
    if (elapsed != "") elof[key] = elapsed
  }
}
END {
  dsn = ENVIRON["AGENTHUB_MYSQL_TEST_DSN"]
  addr = ENVIRON["AGENTHUB_REDIS_TEST_ADDR"]
  ev = ENVIRON["EVIDENCE_ID"]
  tid = ENVIRON["EVIDENCE_TEST_ID"]
  ptid = ENVIRON["EVIDENCE_PLAN_TASK"]
  rec = ENVIRON["EVIDENCE_RECORDED_AT"]
  for (i = 1; i <= n; i++) {
    key = order[i]
    st = result[key]
    status = "error"
    if (st == "pass") status = "ok"
    else if (st == "skip") status = "skipped"
    else if (st == "running") status = "error"
    el = (key in elof) ? elof[key] + 0 : 0
    printf "{\"schemaVersion\":2,\"evidenceId\":\"%s_%03d\",\"testId\":\"%s\",\"planTaskId\":\"%s\",\"layer\":\"middleware\",\"actor\":\"tool\",\"action\":\"test_case\",\"target\":\"%s#%s\",\"status\":\"%s\",\"runId\":\"\",\"conversationId\":\"\",\"traceId\":\"\",\"taskId\":\"\",\"messageId\":\"\",\"errorCode\":\"\",\"elapsedSec\":%.3f,\"detail\":{\"goStatus\":\"%s\"},\"recordedAt\":\"%s\"}\n", \
      ev, i, jesc(tid), jesc(ptid), jesc(pkgof[key]), jesc(tsof[key]), status, el, st, rec
    if (st == "pass") pass++
    else if (st == "skip") skip++
    else if (st == "fail") fail++
    else running++
  }
  printf "%d %d %d %d %d\n", n + 0, pass + 0, fail + 0, skip + 0, running + 0 > "/dev/stderr"
  if (dsn != "" || addr != "") { }
}
'

export EVIDENCE_ID="$evidence_prefix"
export EVIDENCE_TEST_ID="$test_id"
export EVIDENCE_PLAN_TASK="$plan_task"
export AGENTHUB_EVIDENCE_JSONL="$interactions"

set +e
go test -json -count=1 -timeout "$go_timeout" -run "$run_pattern" "${packages[@]}" \
  >"$raw_jsonl" 2>"$out_dir/logs/go-test-stderr.txt"
go_exit=$?
set -e

awk "$redact_program" "$raw_jsonl" >"$redacted_jsonl"
rm -f "$raw_jsonl"
awk "$redact_program" "$out_dir/logs/go-test-stderr.txt" >"$out_dir/logs/go-test-stderr.redacted.txt"
mv "$out_dir/logs/go-test-stderr.redacted.txt" "$out_dir/logs/go-test-stderr.txt"

stats_line=""
set +e
stats_line="$(EVIDENCE_RECORDED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  awk "$parse_program" "$redacted_jsonl" 2>&1 >"$out_dir/events.jsonl" | tail -n 1)"
awk_exit=$?
set -e

read -r total pass fail skip running <<<"${stats_line:-0 0 0 0 0}"
[[ "$total" =~ ^[0-9]+$ ]] || { total=0; pass=0; fail=0; skip=0; running=0; awk_exit=1; }

interactions_rows=0
if [[ -f "$interactions" ]]; then
  interactions_rows="$(grep -c . "$interactions" || true)"
fi

ended_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
duration=$(( $(date -u +%s) - started_epoch ))

cat >"$out_dir/commands.txt" <<EOF
# reproducible commands; environment values are never recorded, only names
git rev-parse HEAD                 -> ${commit}
$(printf 'go test -json -count=1 -timeout %s -run %s' "$go_timeout" "$run_pattern") $(printf '%s ' "${packages[@]}")
# required environment (names only, values must be exported by the operator):
$(printf '%s\n' "${required_vars[@]}" | sed 's/^/#   /')
# optional environment consumed by the gate itself:
#   RUN_INTEGRATION, AGENTHUB_EVIDENCE_JSONL (set by this script)
EOF

status_total="FAIL"
if (( go_exit == 0 )) && (( fail == 0 )) && (( skip == 0 )) && (( pass > 0 )); then
  status_total="PASS"
elif (( pass == 0 && fail == 0 && skip == 0 )); then
  status_total="NO-TESTS"
elif (( skip > 0 )); then
  status_total="SKIP"
fi

{
  echo "# ${test_id} ${plan_task} integration evidence"
  echo
  echo "- task: ${plan_task}"
  echo "- testId: ${test_id}"
  echo "- commit: ${commit} (${branch})"
  echo "- go: ${go_version}"
  echo "- started: ${started_utc}"
  echo "- ended: ${ended_utc}"
  echo "- duration: ${duration}s"
  echo "- packages: ${packages[*]}"
  echo "- run pattern: ${run_pattern}"
  echo "- go test exit: ${go_exit}"
  echo
  echo "## results"
  echo
  echo "| metric | count |"
  echo "|---|---|"
  echo "| total tests | ${total} |"
  echo "| PASS | ${pass} |"
  echo "| FAIL | ${fail} |"
  echo "| SKIP | ${skip} |"
  echo "| incomplete | ${running} |"
  echo "| interaction rows | ${interactions_rows} |"
  echo "| overall | ${status_total} |"
  echo
  echo "## per test"
  echo
  echo "| test | status | elapsed(s) |"
  echo "|---|---|---|"
  if (( total > 0 )); then
    awk -F'"' '
      { for (i = 1; i < NF; i++) { if ($i == "target") tgt = $(i + 2); if ($i == "status") stt = $(i + 2) }
        match($0, /"elapsedSec":[0-9.]+/); el = substr($0, RSTART + 13, RLENGTH - 13)
        printf "| %s | %s | %s |\n", tgt, stt, el
        tgt = ""; stt = "" }' "$out_dir/events.jsonl"
  else
    echo "| (no matching tests) | error | 0 |"
  fi
  echo
  echo "## skipped tests (must be explained)"
  echo
  if (( skip > 0 )); then
    grep '"status":"skipped"' "$out_dir/events.jsonl" | sed 's/^/  /' | head -n 50
    echo
    echo "SKIP rows are never counted as passing evidence."
  else
    echo "- none"
  fi
  echo
  echo "## failed tests"
  echo
  if (( fail > 0 )); then
    grep '"status":"error"' "$out_dir/events.jsonl" | sed 's/^/  /' | head -n 50
  else
    echo "- none"
  fi
  echo
  echo "## interaction coverage (plan §5.5)"
  echo
  echo "| interaction | record file | rows | coverage |"
  echo "|---|---|---|---|"
  echo "| test case result | events.jsonl | ${total} | yes |"
  echo "| middleware interactions | interactions.jsonl | ${interactions_rows} | $(
    if (( interactions_rows > 0 )); then echo yes
    else echo "not_applicable: no test in this selection emitted interaction rows"; fi) |"
  echo "| user actions | - | 0 | not_applicable: gate script runs no browser code |"
  echo "| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |"
  echo
  echo "## redaction"
  echo
  echo "- DSN and Redis address values are replaced with \`<redacted-dsn>\` / \`<redacted-addr>\` in all logs."
  echo "- Environment values are read only through \`ENVIRON\` inside awk, so they never reach argv or evidence files."
} >"$out_dir/summary.md"

# Secret scan on the produced evidence before deciding the exit code.
scan_hits=0
while read -r name; do
  value="${!name:-}"
  [[ -z "$value" ]] && continue
  if grep -R -F -- "$value" "$out_dir" >/dev/null 2>&1; then
    echo "SECURITY: evidence directory contains the value of ${name}" >&2
    scan_hits=1
  fi
done <<<"$(printf '%s\n' "${required_vars[@]}")"
if grep -R -E -n -- "sk-[A-Za-z0-9_-]{20,}|BEGIN [A-Z ]*PRIVATE KEY|OPENAI_API_KEY=[^ ]|//[^/ ]*:[^/ ]*@" \
  "$out_dir" --include='*.md' --include='*.txt' --include='*.jsonl' >/dev/null 2>&1; then
  echo "SECURITY: evidence directory matches secret patterns" >&2
  scan_hits=1
fi
if (( scan_hits != 0 )); then
  exit 1
fi

echo "gate ${test_id}: total=${total} pass=${pass} fail=${fail} skip=${skip} incomplete=${running} go_exit=${go_exit}"
if (( fail > 0 )) || (( go_exit != 0 )) || (( awk_exit != 0 )) || (( running > 0 )); then
  echo "RESULT: FAIL (${test_id})" >&2
  exit 1
fi
if (( skip > 0 )); then
  echo "RESULT: SKIP present (${test_id}) - not a passing gate" >&2
  exit 3
fi
if (( pass == 0 )); then
  echo "RESULT: NO PASS (${test_id}) - selection matched no passing test" >&2
  exit 4
fi
echo "RESULT: PASS (${test_id})"
exit 0
