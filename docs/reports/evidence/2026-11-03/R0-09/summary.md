# R0-09 Docker 内 Playwright 渲染

- commit: ccc143eda3d9d2eb3bb4ab07ef5d230dd138521a
- started: 2026-11-03T04:09:32Z / ended: 2026-11-03T04:09:43Z
- result: **PASS**（8 项断言，0 项失败）
- image: agenthub-r009-orchestrator-new:latest（Playwright v1.55.0 + bundled Chromium，非 root uid=999）

## 断言结果

| 场景 | 结果 | 证据 |
|---|---|---|
| playwright image built (size=859MiB) | PASS | see logs/image.txt |
| renderer container started (non-root, bundled Chromium) | PASS |  |
| HTML -> PNG in container | PASS | artifact=art_3b19611d7746e8a2c218c63e bytes=9859 |
| HTML -> PDF in container | PASS | artifact=art_de7d699c2f256574a3516cf3 bytes=13905 |
| public URL -> PNG in container | PASS | https://example.com |
| private URL rejected: http://169.254.169.254/latest/meta-data/ | PASS | http=400 |
| private URL rejected: http://127.0.0.1:18085/health | PASS | http=400 |
| missing browser fails closed | PASS | http=502, no silent success |

## 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| Artifact 生成/读取 | events.jsonl + logs/*.bin | 4 | yes（PNG/PDF 生成、content 读取、checksum） |
| 安全/私网 | events.jsonl | 2 | yes（169.254.169.254、127.0.0.1 均 400） |
| 故障注入 | events.jsonl | 1 | yes（浏览器缺失 502） |
| 用户/SSE/HITL/记忆/MQ | - | 0 | not_applicable: 渲染端点级测试 |

## 关键事实

- 基础镜像：`mcr.microsoft.com/playwright:v1.55.0-jammy@sha256:55db7b71...`；Playwright 1.55.0，
  内置 Chromium 1187；运行用户 `app`（uid=999，非 root）。
- 最终镜像 901,394,995 bytes（约 859MiB）；本地镜像 ID `sha256:a74fbecd...`。
- HTML→PNG：9,859 bytes，PNG magic + SHA-256 与返回 checksum 一致。
- HTML→PDF：13,905 bytes，`%PDF` + SHA-256 一致。
- 公网 URL→PNG：`https://example.com` 成功（容器有出网）。
- 私网/链路本地 URL：`169.254.169.254`、`127.0.0.1` 均 HTTP 400，且 `render.ValidateURL`
  在启动浏览器前拒绝。
- 浏览器缺失：`PLAYWRIGHT_BROWSERS_PATH=/nonexistent` 时渲染返回 502，不静默成功。

## 已知边界

- 私网拒绝是 URL 预校验；代码注释已说明 DNS rebinding 需网络层 egress 控制兜底，本任务未做网络策略隔离。
- 镜像体积从 alpine 增至 ~859MiB，首拉/部署成本上升；`docker compose build orchestrator-new` 缓存后约 1 分钟。
- 渲染超时默认 60s（`PLAYWRIGHT_RENDER_TIMEOUT_SECONDS`），未在容器内做超时注入测试。
- URL→PNG 依赖容器出网；本次成功，若 CI 无出网应记 SKIP 而非 PASS。

## 回滚

- 恢复原 alpine Dockerfile，并移除 compose 中的 `PLAYWRIGHT_RENDER_SCRIPT` / `PLAYWRIGHT_NODE_PATH`；
  渲染端点恢复为“渲染器不可用”，其他功能不变。

## 证据文件
- commands.txt / events.jsonl / logs/（image.txt、runtime.txt、png.json、pdf.json、*.bin）
