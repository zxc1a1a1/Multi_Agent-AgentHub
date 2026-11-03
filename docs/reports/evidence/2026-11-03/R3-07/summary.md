# R3-07 性能与可访问性门禁

- task: R3-07（`docs/plans/v3-gap-atomic-plan.md` §4.4）
- 对应缺口：GAP-11；UI-07
- 环境：Windows；Node 20.12.2；Playwright + 系统 Edge（msedge）；`go` 不参与
- overall: **PASS**（预算 + 2 个浏览器用例；前端全量 338/338）

## 1. 结果

| 门禁 | 结果 | 证据 |
|---|---|---|
| 首屏入口 gzip ≤500KB | PASS：160,147 B | `logs/budget.txt` |
| 重组件不进入首屏 | PASS：index.html 未引用 JsonSchemaForm/DeclarativeChart/streamdown/mermaid/Sandpack | `scripts/check-build-budget.mjs` |
| 移动端 375×667 无横向溢出 | PASS（修复后 scrollWidth=375） | `logs/playwright.json` |
| 键盘：AgentPicker 打开/Escape | PASS（Enter 打开、搜索框聚焦、Escape 关闭） | 同上 |
| 键盘：输入 → Tab → Enter 发送 | PASS（focus-visible=true，消息出现） | 同上 |
| 前端全量测试 | PASS 338/338 | `logs/vitest-full.txt` |
| `npm run build` | PASS | `logs/build.txt` |

## 2. 真实缺陷与修复

移动端 375px 下文档 `scrollWidth=433`（横向溢出 58px）。定位为聊天布局/输入区在窄视口的
min-content 溢出，修复：
- `App`/`ChatLayout` 根增加 `overflow-x-hidden` + `min-w-0`；
- 聊天滚动区 `overflow-x-hidden min-w-0`；
- 输入区 Agent 行 `flex-wrap`，描述文本 `min-w-0 flex-1 truncate`。
修复后 375px 下 `scrollWidth=375`，输入框与发送按钮可用。

## 3. 使用

```bash
cd frontend
npm run build
npm run check:budget          # 首屏 gzip + 重组件预加载检查
PLAYWRIGHT_CHANNEL=msedge npm run test:e2e:a11y
```

CI 接线：脚本已可在 CI 调用；GitHub workflow 实跑归 R0-05（尚未推送到 GitHub）。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 构建预算 | events.jsonl | 1 | yes |
| 移动端视口 | events.jsonl + playwright.json | 1 | yes（375×667） |
| 键盘操作 | events.jsonl + playwright.json | 1 | yes（Enter/Escape/Tab/focus-visible） |
| 用户消息发送 | playwright.json | 1 | yes |
| HITL/记忆/MQ/多实例 | - | 0 | not_applicable: 本任务为前端门禁 |

## 5. 边界

- 未引入 `@axe-core/playwright`（计划标注可选）；未做完整 axe 扫描。
- HITL 卡片的键盘操作未在本 spec 覆盖（mock 未触发审批计划）；R5-04 浏览器矩阵补充。
- 仅在系统 Edge 运行；Chromium/Firefox/WebKit 与 CI 实跑归 R5-04/R0-05。

## 6. 回滚

- 移除 `check:budget`/`test:e2e:a11y` 脚本与门禁；布局修复可单独回退。

## 7. 证据

- `docs/reports/evidence/2026-11-03/R3-07/`（commands.txt、events.jsonl、summary.md、logs/）
