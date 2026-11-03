# R3-03 Streamdown flag 试点

- task: R3-03（`docs/plans/v3-gap-atomic-plan.md` §4.4）
- 对应缺口：GAP-11；UI-01
- 环境：Windows；Node 20.12.2；Vite 6.4.2；`streamdown@2.7.1`（Apache-2.0）
- overall: **PASS**（本轮 19/19；前端全量 318/318；默认/试点构建均通过）

## 1. 许可与包体

- 许可：Apache-2.0（permissive，无 AGPL/SSPL），详见 `docs/reports/r3-03-streamdown-license.md`。
- 发布包 125,435 bytes；依赖含 `remark/rehype` 系与 `rehype-harden`/`rehype-sanitize`。
- 决策：**GO，仅 opt-in 试点**；默认构建不加载。

## 2. 行为

- 开关：`vite build --mode streamdown`（`define.__STREAMDOWN_ENABLED__=true`）；其它模式为 false。
- flag off：`StreamingText` 完全走旧 `react-markdown` 路径（12 个既有测试通过），不加载 Streamdown。
- flag on：`StreamdownRenderer` 动态 `import('streamdown')` + CSS，`Suspense` 期间显示旧渲染；
  动态 chunk 失败或组件抛错时由 error boundary 回退旧渲染。
- 覆盖：未闭合代码围栏、CJK、行内数学、长文本、HTML/`javascript:` 注入（无 script/onerror/JS 链接执行）。

## 3. 构建预算

| 构建 | 入口 gzip | Streamdown |
|---|---|---|
| 默认 `npm run build` | 160.14 KiB | 不加载 |
| `vite build --mode streamdown` | 160.30 KiB | 独立 chunk 200.33 KiB / gzip 64.11 KiB，动态 import |

- 入口均 ≤500KB；重组件未进入默认首屏。
- 证据：`logs/build-default.txt`、`logs/build-streamdown.txt`、`logs/vitest-full.txt`。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 文本渲染/流式内容 | events.jsonl | 19 | yes（Vitest，含 4 个真实 Streamdown 用例） |
| 构建/懒加载 | logs/build-*.txt | 2 | yes |
| 用户/API/SSE/HITL/记忆/MQ | - | 0 | not_applicable: 渲染器级试点 |

## 5. 边界

- 测试环境为 jsdom；未在真实浏览器中做长流式增量对比（R5-04 浏览器矩阵）。
- 未替换默认渲染器；切换默认仍需性能/兼容评审。
- Streamdown 的 KaTeX/Mermaid 集成未启用（本试点只验证 Markdown 流式）。

## 6. 回滚

- 不传 `--mode streamdown`（默认）即完全回退；也可删除依赖。

## 7. 证据

- `docs/reports/evidence/2026-11-03/R3-03/`（commands.txt、events.jsonl、summary.md、logs/）
- 许可评审：`docs/reports/r3-03-streamdown-license.md`
