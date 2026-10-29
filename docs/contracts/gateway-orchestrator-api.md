# Gateway ↔ Orchestrator API Detail

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：start/subscribe/control 分离、游标与幂等）

## Authoritative source order

1. `docs/contracts/event-replay.md`
2. `docs/contracts/gateway-orchestrator.md`
3. `docs/contracts/runtime-harness.md`

## StartRunRequest

| Field | Required | Description |
|---|---:|---|
| `requestId` | yes | Gateway 生成的稳定幂等键；重复请求返回同一 runId |
| `threadId` / `conversationId` | yes | Gateway 会话 id |
| `runId` | yes | Gateway 预分配运行 id（与 requestId 绑定） |
| `workspaceId` / `principalId` | yes | 服务端身份上下文生成，不接受浏览器任意覆盖 |
| `executionPath` | yes | `single_chat` / `group_chat` / `main_agent_orchestration` |
| `allowedAgents[]` | yes | 当前路径允许集合 |
| `messages[]` | yes | 当前用户轮消息 |
| `history[]` | no | 有界历史 |
| `tools[]` | no | 前端声明的 runtime skills（非执行授权） |
| `metadata` | no | conversationType、traceId 等 |

## StartRunResponse

```text
runId, accepted(bool), deduplicated(bool), status
```

- `requestId` 唯一约束由 Orchestrator 执行；Gateway 超时后按同一 requestId 重试/查询。
- 两次服务事务不保证跨服务原子。

## SubscribeRunRequest

```text
runId, cursor{seq, subIndex, mappingVersion}, consumerKey
```

- 返回内部事件流（`InternalEvent`），Gateway 负责映射为公开 AG-UI。
- 断开订阅不取消 durable run。
- 游标只跳过本订阅已确认的 `(seq, subIndex)`。

## 控制操作

```text
GetRunStatus(runId) → status, lastSeq, terminalReason, pendingApproval
SubmitApproval(approvalId, runId, planId, revision, toolCallId, decision, idempotencyKey)
CancelRun(runId, reason) → 保存 cancelled，尝试取消远端任务
```

## InternalEvent

```text
runId, seq, subIndex, eventType, payload, agentName, createdAt
```

- `(runId, seq)` 唯一且有序；一个内部事件可映射多个公开事件。
- payload 不含密钥、DSN、内部 URL、stack trace。
- 历史水位不是浏览器消费确认。

## ListAgents

Gateway 代理 Agent 列表，避免 Frontend 直连 Child Agents。

## 失败规则

- Orchestrator 不可用 → HTTP 503 或流已开始后的 `RUN_ERROR`。
- 重复 StartRun 不重复创建运行/消息；返回 `deduplicated=true`。
- 审批幂等：同一 `idempotencyKey` 重复提交返回同一结果；不同参数复用旧审批被拒绝。
