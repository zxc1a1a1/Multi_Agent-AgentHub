# Event Replay & Internal gRPC Contract

**Status:** Active (v2, P0 frozen)
**Owner:** AgentHub Orchestrator / Gateway
**Last updated:** 2026-10-29

## 1. 权威顺序

```text
SQL execution_events（权威、单调 seq）
  → outbox（同事务）
  → Redis Streams（短期副本、可裁剪）
  → Gateway 投影（幂等）
  → 浏览器 AG-UI SSE（可恢复游标）
```

- Redis Stream ID 不是业务版本；业务版本是 `(runId, seq)`。
- 未提交的文本增量不向浏览器确认可重放。
- Pub/Sub 只能做可丢失的唤醒提示，不能承载审批结果、终态或恢复日志。

## 2. seq 与事件

- 每个 run 分配单调递增 `seq`（从 1 开始），SQL 唯一约束 `(runId, seq)`。
- 状态/checkpoint 变化在同一个 Orchestrator 事务内更新执行版本、写 `execution_events` 与 outbox。
- 文本增量可按小批次持久化（初值 50 ms 或 32 KiB 先到者）；生命周期事件、工具派发、终态立即冲刷。
- 一个内部事件可映射多个公开 AG-UI 事件，公开事件 ID 使用稳定 `(runId, seq, subIndex)`。
- 中段重连要重新取当前 seq 并映射，只跳过该订阅已确认的 subIndex，不能只按 seq 去重。

## 3. 内部 gRPC（冻结操作面）

```protobuf
service OrchestratorService {
  rpc StartRun(StartRunRequest) returns (StartRunResponse);        // 幂等：requestId
  rpc SubscribeRun(SubscribeRunRequest) returns (stream InternalEvent);
  rpc GetRunStatus(GetRunStatusRequest) returns (RunStatus);
  rpc SubmitApproval(SubmitApprovalRequest) returns (ApprovalResult);
  rpc CancelRun(CancelRunRequest) returns (CancelRunResponse);      // 显式取消
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
}
```

- `StartRun` 以 `requestId` 唯一约束幂等接入；重复请求返回同一 runId，不重复创建运行/消息。
- `SubscribeRun` 携带订阅游标（最后确认的 `seq/subIndex`）；断开订阅不取消 durable run。
- 审批通过 `SubmitApproval`，绑定 `runId/planId/revision/toolCallId/idempotencyKey`。
- 所有操作携带内部鉴权 token 与 traceId；不暴露给浏览器。
- 迁移期 HTTP/SSE 是兼容实现，文档不得写成目标架构；Gateway 目标启动不得静态直连 Child Agent。

## 4. Outbox relay

1. 事务提交 outbox（仅 ID/版本/事件类型，不含敏感正文或凭据）。
2. relay 在事务外按 run 保序投递 Redis Stream。
3. 投递后确认前崩溃可能重复发送；消费者按 `(runId, seq)` 去重。
4. relay 记录 Redis 尾游标 R，再读取 SQL 水位 H 回放；之后从 R 续读。
5. 缺号与定期 SQL 水位/终态核验共同负责补齐，包括没有后继事件的尾部缺失。
6. 重复投递仍可能发生；不得因重复而重复追加文本或重复执行工具。

## 5. Gateway 投影与订阅

- Gateway 先幂等更新消息投影，再发送公开事件；发送完成终态前确认投影已提交。
- 投影消费位置与每个订阅的发送游标是两套状态：已投影事件仍要发送给尚未收到的订阅者。
- 两个 Gateway OS 进程订阅同一 run，各自收到完整事件；共享投影更新不得抑制另一订阅发送。
- 同一 seq 的中间 subIndex 断开重连不漏片段、不重复应用投影。
- 慢订阅者：有界发送队列 + 写 deadline；队列满断开订阅并允许恢复，不阻塞其他订阅、不删除已提交日志。

## 6. 恢复与快照

- 前端记录最后实际消费游标；重连从该位置续传，不创建新任务。
- 游标包含 `(runId, seq, subIndex)` 与映射版本。
- 映射版本或 SQL 保留窗口不兼容时返回明确 snapshot/resync 要求，不静默重启任务。
- SQL 历史切换到 Redis 实时流：先捕获 Redis 尾游标 R，读取 SQL 已提交水位 H 回放到 H，再从 R 开始读取，按业务 seq/公开游标过滤重复。
- Redis 不可用或被清空：直接走 SQL 补读；SQL 仍能恢复审批、终态和允许窗口内的事件。
- 历史水位 H 不是浏览器消费确认；写入 socket 不等于浏览器已处理。

## 7. Redis 键与 TTL

```text
agenthub:{env}:{workspaceId}:run:{runId}:events     # Stream，短期副本
agenthub:{env}:{workspaceId}:run:{runId}:state      # 可重建视图，短 TTL
agenthub:{env}:{workspaceId}:memory:{scope}:{epoch} # 版本化记忆缓存
agenthub:{env}:catalog:{revision}                   # 能力缓存
agenthub:{env}:instance:{id}:heartbeat              # 心跳
```

- 缓存 TTL 上限 5 分钟且不超过来源剩余有效期；Memory 缓存键含 scope/epoch/revision。
- 权限撤销/记忆删除立即作用于新授权，不等待 TTL。
- Redis 不保存 API key、token 明文、完整敏感 prompt、DSN。
- 多订阅广播使用独立游标的 `XREAD`，不把多个订阅者放进同一 `XREADGROUP`。

## 8. 门禁

- 两个独立 Gateway 进程各自收到完整事件。
- relay 在 publish 后确认前崩溃，重试不重复追加文本；序号缺口可恢复。
- SQL 回放切换 Redis 窗口只产生一条终态；Redis 重启后 SQL 能恢复审批/终态/事件。
- Gateway 投影提交前/后崩溃均可恢复；同 seq 中间 subIndex 断开不漏不重。
- 浏览器验证断连重连、手动取消、审批、历史回放、Agent 归属与控制台/网络异常。

## 9. 群聊 AGENT_TURN 投影（v3 A0-05）

- 公开事件按 `(runId, seq, subIndex)` 写入 `run_public_events`；同一 turn 的
  content 事件共享 `messageId/turnIndex`。
- 投影 consumer 固定为 `assistant_message.v1`，游标表
  `event_projection_offsets(run_id, consumer_key)` 记录 `last_seq/last_sub_index`。
- 投影单位是“完整 turn”：`AGENT_TURN_FINISHED`（或终态）到达后，将该 turn
  的全部 content 合成一条 assistant 消息；`AGENT_TURN_STARTED` 本身不落消息。
- 崩溃恢复：Gateway 在 journal 已提交、投影未提交时退出，另一个实例的投影
  worker 从 offsets 续读并补齐；同一 `(seq, subIndex)` 不重复插入。
- 投影偏移只用于避免重复写库，不用于抑制其他订阅者；每个 SSE 订阅仍有独立
  发送游标。
- 多 Agent run 刷新后必须是多条消息，按 `turnIndex` 排序；合并成一条即视为
  投影缺陷。
- 证据：`services/gateway/store/mysql_integration_test.go`
  `TestIntegrationMySQLProjectionOffsetIsIdempotent`；
  `services/gateway/projection/worker_test.go` 覆盖多 Agent 每轮独立投影、
  未完成轮次跳过、重复恢复不重复写入；`worker_integration_test.go` 在注入 DSN
  时验证真实 MySQL 恢复（CI 门禁 A0-07 执行）。
- 限制：worker 只能恢复已写入 `run_public_events` 的轮次；若 Gateway 在
  `AGENT_TURN_FINISHED` 之前死亡，该轮按“未完成”不落库，需由 A1-09c 的
  终态判定/取消墓碑补齐。
