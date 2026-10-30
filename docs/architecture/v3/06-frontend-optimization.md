# 06 前端优化与技术选型

- 状态：待评审
- 当前栈：React 18.3、Zustand 4.5、Vite 6、自定义 SSE、react-markdown 9、highlight.js 11
- 目标：降低流式与历史状态耦合，提升长会话性能，保持 AG-UI 协议不变

## 1. 当前问题（有证据）

| 问题 | 证据 | 影响 |
|---|---|---|
| 服务端状态与流式状态都在 Zustand | `messageStore.ts` 体量大、闭包多 | 难测试、恢复逻辑重复 |
| 消息列表无虚拟化 | `ChatWindow` 直接渲染全部 | 长会话滚动/内存压力 |
| 大 chunk | `npm run build` 约 1.44MB JS（gzip 446KB） | 首屏慢 |
| SSE 解析与状态 reducer 耦合 | `agui/client.ts` + store | 重连/游标逻辑分散 |
| 代码高亮 | highlight.js 全量引入 | 包体与主题成本 |
| 刷新恢复 | localStorage 游标 + recover | Flash/工具卡片未完全验证 |

## 2. 选型对比

| 能力 | 方案 A | 方案 B | 建议 |
|---|---|---|---|
| 服务端状态 | 继续 Zustand 手写缓存 | TanStack Query | **B（新代码）**，渐进迁移 |
| 长列表 | 直接渲染 | TanStack Virtual | **B** |
| 代码展示 | highlight.js | Shiki（静态）/ Monaco（编辑） | 展示用 Shiki；编辑再 Monaco |
| 实时通道 | SSE | WebSocket | **SSE 保留**；双向交互再评估 WS |
| 协议 | 自定义 reducer | AG-UI 官方客户端 | 继续 AG-UI 事件语义，客户端可自研 |
| 本地状态 | Zustand | Redux/Jotai | 保留 Zustand（轻量） |

## 3. 推荐架构

```text
server state: TanStack Query
  conversations / messages / agents / memory / run status
ephemeral stream state: Zustand
  active run, deltas, approval card, tool call cards
transport: fetch + ReadableStream (SSE)
  cursor: (runId, seq, subIndex)
render: virtualized message list + lazy heavy components
```

### 3.1 状态分层

- Query 负责：列表、分页、缓存、失效、重试、背景刷新。
- Zustand 负责：当前流、临时消息、确认卡、工具卡、UI 开关。
- 流结束后用 Query invalidate/refetch 与 SQL 历史对齐。

### 3.2 流式渲染

- 保留 POST SSE（需要 Authorization/body）。
- 事件归并：`seq` 单调、`subIndex` 同 seq 排序；重复事件幂等。
- 断线：先读 `/events?after=&subIndex=`，再继续；终态停止。
- 审批：快照丢失时从事件回放重建；不要只依赖本地状态。

### 3.3 性能

- 路由/重组件 lazy：MemoryPanel、CodePreview、Monaco。
- 消息列表虚拟化；流式消息固定在可见区。
- 代码高亮按语言懒加载；长代码折叠。
- 包体预算：首屏 JS ≤ 500KB gzip（目标，需实测）。
- 避免在每次 delta 触发全树渲染；按 message 分片订阅。

## 4. 迁移步骤

1. 引入 TanStack Query，仅接管 conversations/messages 读取，不改 SSE。
2. 消息列表接 TanStack Virtual，保持现有 DOM 结构与测试。
3. 抽离 `runEventReducer`，让流式/回放共用同一 reducer。
4. 懒加载 CodePreview/MemoryPanel，设 bundle 预算。
5. 视觉与交互回归：审批、取消、刷新恢复、长会话。

## 5. 验收指标

| 指标 | 目标 | 测量 |
|---|---|---|
| 首 token 时间 | 不劣化 | 浏览器 Performance |
| 长会话滚动 | 60fps 目标 | DevTools |
| 首屏 JS | ≤500KB gzip | `vite build` |
| 重连重复/缺口 | 0 | 游标测试 |
| 消息列表内存 | 随视口有界 | heap snapshot |

## 6. 风险

- 过早引入 Query 会造成双缓存；必须限定范围。
- 虚拟化会改变滚动/复制/查找行为，需要回归测试。
- Monaco 体积大，非编辑场景不要引入。
- SSE 在代理层可能被缓冲；需要 `Cache-Control: no-cache` 与心跳。

## 7. 官方来源

- AG-UI：https://docs.ag-ui.com/concepts/architecture
- TanStack Virtual：https://tanstack.com/virtual/latest/docs/introduction
- TanStack Query：https://tanstack.com/query/latest
- TanStack AI SSE 参考：https://tanstack.com/ai/latest/docs/reference/functions/toServerSentEventsStream
