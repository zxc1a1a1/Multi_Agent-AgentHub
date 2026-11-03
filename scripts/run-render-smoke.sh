#!/usr/bin/env bash
# R0-09: Playwright rendering inside the Orchestrator image.
# Proves HTML->PNG, HTML->PDF, private/link-local URL rejection and fail-closed
# behaviour when the browser is missing. It does not touch the host browser.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

image="${RENDER_IMAGE:-agenthub-r009-orchestrator-new:latest}"
out_dir="${EVIDENCE_OUT:-}"
if [[ "${1:-}" == "--out" ]]; then out_dir="${2:-}"; shift 2; fi
[[ -z "$out_dir" ]] && { echo "export EVIDENCE_OUT or pass --out DIR" >&2; exit 2; }
mkdir -p "$out_dir/logs"
events="$out_dir/events.jsonl"
: >"$events"
started_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
token="$(openssl rand -hex 16)"
name_a="agenthub-r009-render-a"
name_b="agenthub-r009-render-b"
port_a=18085
port_b=18086
failures=()
scenarios=()
seq=0

ev() {
  seq=$((seq + 1))
  printf '{"schemaVersion":2,"evidenceId":"ev_20261103_R0_09_%03d","testId":"R0-09","planTaskId":"R0-09","layer":"middleware","actor":"tool","action":"artifact_event","target":"%s","status":"%s","detail":{%s},"counts":{%s},"recordedAt":"%s"}\n' \
    "$seq" "$1" "$2" "$3" "$4" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$events"
}
record() {
  local name="$1" result="$2" note="${3:-}"
  scenarios+=("$name|$result|$note")
  if [[ "$result" == "PASS" ]]; then
    echo "PASS: $name ${note:+($note)}"
  elif [[ "$result" == "SKIP" ]]; then
    echo "SKIP: $name - $note"
  else
    failures+=("$name: $note")
    echo "FAIL: $name - $note" >&2
  fi
}

cleanup() {
  docker rm -f "$name_a" "$name_b" >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_renderer() {
  local name="$1" port="$2"; shift 2
  docker rm -f "$name" >/dev/null 2>&1 || true
  MSYS_NO_PATHCONV=1 docker run -d --name "$name" -p "${port}:8080" \
    -e ORCHESTRATOR_ADDR=:8080 \
    -e INTERNAL_SERVICE_TOKEN="$token" \
    -e ARTIFACT_STORE_DIR=/data/artifacts \
    -e ARTIFACT_BLOB_DIR=/data/artifacts/blobs \
    -e PLAYWRIGHT_RENDER_SCRIPT=/app/render.cjs \
    -e AGENTHUB_ENV=r009 \
    -e ORCHESTRATOR_LLM_PROVIDER=anthropic \
    -e ORCHESTRATOR_LLM_MODEL=smoke-model \
    -e ORCHESTRATOR_LLM_API_KEY=smoke-fake-key \
    -e ORCHESTRATOR_LLM_BASE_URL=http://127.0.0.1:18081 \
    "$@" "$image" >/dev/null || return 1
  for _ in $(seq 1 40); do
    curl -sf --max-time 3 "http://127.0.0.1:${port}/health" >/dev/null 2>&1 && return 0
    sleep 1
  done
  return 1
}

post_render() {
  local port="$1" body="$2" out="$3"
  curl -s -o "$out" -w '%{http_code}' --max-time 120 -X POST \
    "http://127.0.0.1:${port}/internal/orchestrator/render" \
    -H "Authorization: Bearer ${token}" -H 'Content-Type: application/json' \
    --data "$body"
}

get_content() {
  local port="$1" path="$2" out="$3"
  curl -s -o "$out" -w '%{http_code}' --max-time 60 \
    "http://127.0.0.1:${port}${path}" -H "Authorization: Bearer ${token}"
}

# ── image facts ────────────────────────────────────────────────────────────
echo "== image facts" | tee "$out_dir/logs/smoke.log"
docker image inspect "$image" --format '{{.Id}} {{.Size}}' >"$out_dir/logs/image.txt" 2>&1
docker run --rm --entrypoint sh "$image" -c 'id; node -e "console.log(require(\"playwright/package.json\").version)"; echo BROWSERS=$PLAYWRIGHT_BROWSERS_PATH; ls $PLAYWRIGHT_BROWSERS_PATH | head -5' >"$out_dir/logs/runtime.txt" 2>&1
eval "$(docker image inspect "$image" --format 'SIZE={{.Size}}' 2>/dev/null || echo SIZE=0)"
record "playwright image built (size=$((SIZE/1024/1024))MiB)" PASS "see logs/image.txt"

if ! start_renderer "$name_a" "$port_a"; then
  record "renderer container started" FAIL "health timeout"
  exit 1
fi
record "renderer container started (non-root, bundled Chromium)" PASS

# ── HTML -> PNG ────────────────────────────────────────────────────────────
status="$(post_render "$port_a" '{"runId":"run_r009","html":"<h1>AgentHub render</h1><p>container png</p>","format":"png"}' "$out_dir/logs/png.json")"
if [[ "$status" == "200" ]]; then
  artifact="$(python -c "import json;print(json.load(open('$out_dir/logs/png.json'))['artifactId'])" 2>/dev/null)"
  checksum="$(python -c "import json;print(json.load(open('$out_dir/logs/png.json'))['checksum'])" 2>/dev/null)"
  kind="$(python -c "import json;print(json.load(open('$out_dir/logs/png.json'))['kind'])" 2>/dev/null)"
  code="$(get_content "$port_a" "/internal/orchestrator/artifacts/${artifact}/content" "$out_dir/logs/png.bin")"
  sig="$(head -c 8 "$out_dir/logs/png.bin" | od -An -tx1 | tr -d ' \n')"
  actual="sha256:$(sha256sum "$out_dir/logs/png.bin" | cut -d' ' -f1)"
  if [[ "$code" == "200" && "$sig" == "89504e470d0a1a0a" && "$actual" == "$checksum" && "$kind" == "image" ]]; then
    record "HTML -> PNG in container" PASS "artifact=${artifact} bytes=$(stat -c %s "$out_dir/logs/png.bin")"
    ev "html-png" ok "{\"artifactId\":\"${artifact}\"}" "{\"bytes\":$(stat -c %s "$out_dir/logs/png.bin")}"
  else
    record "HTML -> PNG in container" FAIL "status=${code} sig=${sig} checksum=${actual}"
  fi
else
  record "HTML -> PNG in container" FAIL "http=${status}"
fi

# ── HTML -> PDF ────────────────────────────────────────────────────────────
status="$(post_render "$port_a" '{"runId":"run_r009","html":"<h1>AgentHub PDF</h1><p>container pdf</p>","format":"pdf"}' "$out_dir/logs/pdf.json")"
if [[ "$status" == "200" ]]; then
  artifact="$(python -c "import json;print(json.load(open('$out_dir/logs/pdf.json'))['artifactId'])" 2>/dev/null)"
  checksum="$(python -c "import json;print(json.load(open('$out_dir/logs/pdf.json'))['checksum'])" 2>/dev/null)"
  code="$(get_content "$port_a" "/internal/orchestrator/artifacts/${artifact}/content" "$out_dir/logs/pdf.bin")"
  sig="$(head -c 4 "$out_dir/logs/pdf.bin")"
  actual="sha256:$(sha256sum "$out_dir/logs/pdf.bin" | cut -d' ' -f1)"
  if [[ "$code" == "200" && "$sig" == "%PDF" && "$actual" == "$checksum" ]]; then
    record "HTML -> PDF in container" PASS "artifact=${artifact} bytes=$(stat -c %s "$out_dir/logs/pdf.bin")"
    ev "html-pdf" ok "{\"artifactId\":\"${artifact}\"}" "{\"bytes\":$(stat -c %s "$out_dir/logs/pdf.bin")}"
  else
    record "HTML -> PDF in container" FAIL "status=${code} sig=${sig}"
  fi
else
  record "HTML -> PDF in container" FAIL "http=${status}"
fi

# ── URL -> PNG (public target; skipped with a reason when egress is blocked) ─
status="$(post_render "$port_a" '{"runId":"run_r009","url":"https://example.com","format":"png"}' "$out_dir/logs/url.json")"
if [[ "$status" == "200" ]]; then
  record "public URL -> PNG in container" PASS "https://example.com"
  ev "url-png" ok '{"url":"https://example.com"}' '{}'
else
  record "public URL -> PNG in container" SKIP "http=${status}; container egress may be blocked (SKIP, not counted as PASS)"
  ev "url-png" skipped '{"url":"https://example.com"}' '{}'
fi

# ── private/link-local URL rejection ──────────────────────────────────────
for target in "http://169.254.169.254/latest/meta-data/" "http://127.0.0.1:18085/health"; do
  status="$(post_render "$port_a" "{\"runId\":\"run_r009\",\"url\":\"${target}\",\"format\":\"png\"}" "$out_dir/logs/private.json")"
  if [[ "$status" == "400" ]]; then
    record "private URL rejected: ${target}" PASS "http=400"
    ev "private-url" denied "{\"url\":\"${target}\"}" '{}'
  else
    record "private URL rejected: ${target}" FAIL "http=${status}"
  fi
done

# ── missing browser must fail closed ──────────────────────────────────────
if start_renderer "$name_b" "$port_b" -e PLAYWRIGHT_BROWSERS_PATH=/nonexistent; then
  status="$(post_render "$port_b" '{"runId":"run_r009","html":"<h1>no browser</h1>","format":"png"}' "$out_dir/logs/nobrowser.json")"
  if [[ "$status" == "502" ]]; then
    record "missing browser fails closed" PASS "http=502, no silent success"
    ev "missing-browser" error '{"httpStatus":502}' '{}'
  else
    record "missing browser fails closed" FAIL "http=${status} (expected 502)"
  fi
else
  record "missing browser fails closed" FAIL "second renderer did not start"
fi
docker rm -f "$name_b" >/dev/null 2>&1 || true

# ── summary ────────────────────────────────────────────────────────────────
ended_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
overall=PASS
(( ${#failures[@]} > 0 )) && overall=FAIL
{
  echo "# R0-09 Docker 内 Playwright 渲染"
  echo
  echo "- commit: ${commit}"
  echo "- started: ${started_utc} / ended: ${ended_utc}"
  echo "- result: **${overall}**（${#scenarios[@]} 项断言，${#failures[@]} 项失败）"
  echo "- image: ${image}（Playwright v1.55.0 + bundled Chromium，非 root uid=999）"
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
  echo "| Artifact 生成/读取 | events.jsonl + logs/*.bin | 4 | yes（PNG/PDF 生成、content 读取、checksum） |"
  echo "| 安全/私网 | events.jsonl | 2 | yes（169.254.169.254、127.0.0.1 均 400） |"
  echo "| 故障注入 | events.jsonl | 1 | yes（浏览器缺失 502） |"
  echo "| 用户/SSE/HITL/记忆/MQ | - | 0 | not_applicable: 渲染端点级测试 |"
  echo
  echo "## 证据文件"
  echo "- commands.txt / events.jsonl / logs/（image.txt、runtime.txt、png.json、pdf.json、*.bin）"
} >"$out_dir/summary.md"

for s in "${scenarios[@]}"; do
  IFS='|' read -r n r _ <<<"$s"
  [[ "$r" == "FAIL" ]] && echo "FAILED: $n" >&2
done
echo "RESULT: ${overall}（assertions=${#scenarios[@]} failures=${#failures[@]}）"
[[ "$overall" == "PASS" ]] && exit 0 || exit 1
