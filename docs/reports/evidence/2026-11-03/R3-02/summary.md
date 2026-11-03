# R3-02 沙箱 HTML artifact

- task: R3-02（`docs/plans/v3-gap-atomic-plan.md` §4.4）
- 对应缺口：GAP-11；UI-02、S-02
- 环境：Windows；Node 20.12.2；Playwright + 系统 Edge（channel=msedge）；`go1.26.5`
- overall: **PASS**（37 Vitest + 1 Playwright + 1 Go，0 失败）

## 1. 结果

| 套件 | 结果 | 覆盖 |
|---|---|---|
| Vitest（SandboxedHtml/MessageBubbleUIComponent/skills） | 37/37 PASS | 属性、CSP、大小/类型/超时错误、消息源校验、registry |
| Playwright（真实 Edge） | 1/1 PASS | 父页读取、fetch、图片请求全部 blocked，无网络请求 |
| Go（artifact content header） | 1/1 PASS | `X-Content-Type-Options: nosniff` + attachment |
| 前端全量 | 311/311 PASS | 无回归 |
| `npm run build` | PASS | 19.59s |

## 2. 行为

- `html_sandbox` 卡片通过 `fetchArtifactContent` 读取 `/api/artifacts/{id}/content`，在
  `<iframe sandbox="allow-scripts" referrerPolicy="no-referrer" srcdoc=...>` 中渲染。
- **没有 `allow-same-origin`**：子文档为 opaque origin，`parent.document` 抛 SecurityError。
- 注入 CSP：`default-src 'none'; img-src data: blob:; style-src 'unsafe-inline';
  script-src 'unsafe-inline'; connect-src 'none'; frame-src 'none'; form-action 'none';
  base-uri 'none'`；CSP 位于 `<head>` 最前，artifact 自带 CSP 只能收紧不能放宽。
- 256KB 上限（拉取后与解码后双重检查）；非 HTML content-type 拒绝；10s 超时；错误卡片可重试。
- 父页 `message` 监听只接受 `event.source === iframe.contentWindow` 且 JSON ≤16KB 的消息；
  不使用 `event.origin` 作为唯一信任（opaque origin）。
- 所有其它 `ui.component`（json_form/chart/code_editor）仍降级为文本卡片。
- 后端 artifact content 增加 `X-Content-Type-Options: nosniff`。

## 3. Playwright 逃逸结果

子文档依次尝试：
1. `parent.document.title = 'pwn'` → blocked；
2. `fetch('http://169.254.169.254/latest/meta-data/')` → blocked（CSP connect-src none）；
3. `<img src="http://127.0.0.1:9/probe.png">` → blocked（CSP img-src 仅 data/blob）；
4. 父页 `request` 监听：上述目标 0 请求。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 组件渲染/iframe 属性 | events.jsonl | 37 | yes（Vitest） |
| 浏览器逃逸/网络 | events.jsonl + logs/playwright.json | 1 | yes（真实 Edge） |
| Artifact content header | events.jsonl + logs/go-test.jsonl | 1 | yes |
| 用户操作（点击/键盘） | Vitest | 1 | yes（文本降级卡片） |
| HITL/记忆/MQ/多实例 | - | 0 | not_applicable: 前端渲染任务 |

## 5. 边界

- 未做 iframe 内 CPU/内存限额；沙箱允许脚本运行，只限制读父页/网络。
- Playwright 用系统 Edge（channel=msedge），不是 Chromium/Firefox/WebKit 矩阵（R5-04）。
- artifact 内容来自 Gateway `/api/artifacts/{id}/content`，本任务未用真实 run 生成 HTML artifact
  （内容由 mock fetch 提供）；真实工具链见 R4-05/R0-09。

## 6. 回滚

- registry `html_sandbox.renderer` 改回 null，MessageBubble 删除 ui_component 分支；文本卡片恢复。

## 7. 证据

- `docs/reports/evidence/2026-11-03/R3-02/`（commands.txt、events.jsonl、summary.md、logs/）
