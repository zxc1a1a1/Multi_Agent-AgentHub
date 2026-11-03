# R3-05 声明式图表

- task: R3-05（`docs/plans/v3-gap-atomic-plan.md` §4.4）
- 对应缺口：GAP-11；UI-05
- 环境：Windows；Node 20.12.2；无新增运行时依赖
- overall: **PASS**（43/43 相关 Vitest；前端全量 338/338；构建通过）

## 1. 结果

| 能力 | 实现 |
|---|---|
| line / bar / area / pie | 纯 SVG 渲染，无 eval/表达式 |
| 数据上限 | 5000 点、10 个 series、payload ≤256KB |
| 类型白名单 | 仅 line/bar/area/pie；未知类型拒绝 |
| 标签安全 | React 文本转义；`<img onerror>` 不产生节点 |
| 懒加载 | `DeclarativeChart-CIM027tx.js` 4.54KB / gzip 2.01KB，独立 chunk |
| 入口预算 | 默认入口 gzip 160.43KiB（≤500KB） |

**Recharts 决策**：计划建议 Recharts；本实现用纯 SVG，避免约 500KB 依赖。所需
bar/line/area/pie 均可声明式渲染，且无函数/表达式执行面；已在 summary 中明确记录该偏差。
如后续需要交互/tooltip，再评估 Recharts 的懒加载接入。

## 2. 行为

- `ui.component: chart` 的 props：`{type, title?, series:[{key,name?,color?}], data:[{label?,values:number[]}]}`。
- 每个点必须为每个 series 提供一个有限数值；颜色只接受 `#rrggbb`，否则用内置色板。
- 未知字段忽略；函数/表达式不可执行（JSON 输入）。
- 非法 spec 显示 `Chart rejected` + reason，不渲染 SVG。

## 3. 测试

- validateChartProps：四种类型、非法类型、series/data 形状、NaN、series/point/payload 上限、未知字段。
- 渲染：line/bar/area/pie、legend、非法 spec、XSS 标签。
- MessageBubble：合法 spec 渲染 SVG；非法 spec 显示 reason。
- 全量 338 测试通过；`npm run build` 通过。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 图表渲染/上限 | events.jsonl | 43 | yes |
| 用户/API/SSE/HITL/记忆/MQ | - | 0 | not_applicable: 组件级 |

## 5. 边界

- 无 tooltip/缩放/交互编辑；pie 使用第一个 series 的数值。
- 未做真实大数据渲染性能测量（仅数量上限拒绝）。
- 未引入 Recharts（见 §1 决策）。

## 6. 回滚

- registry `chart` renderer 改回 null；MessageBubble 删除分支。

## 7. 证据

- `docs/reports/evidence/2026-11-03/R3-05/`（commands.txt、events.jsonl、summary.md、logs/）
