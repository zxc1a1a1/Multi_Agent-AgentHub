# P4 阶段报告：Redis relay、事件回放与恢复入口（部分）

- 日期：2026-10-29
- 基线：`dev / bc2acab`（P3b 接线后）
- 状态：**SQL outbox → Redis Streams relay、SQL 事件回放、Gateway 恢复代理与 requestId 幂等已落地；gRPC、投影游标、前端恢复/管理 UI、Memory 缓存未完成**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| Outbox relay | `services/orchestrator/relay/` | 按 outbox id 顺序 XADD 到 run Stream，MAXLEN+TTL；失败记录 attempt 并停止后续 seq，保证同 run 不越序 |
| Redis 启动接线 | `services/orchestrator/cmd/orchestrator/main.go` | `REDIS_URL` 存在时启动 relay；Redis 缺失只告警，SQL 仍权威 |
| Outbox 读取 | `.../persistence/mysql/repository.go` | join executions 获取 workspace；`GetEvent`；按 run 读取 |
| SQL 回放接口 | `services/orchestrator/httpapi/handler_replay.go` | `GET /internal/orchestrator/runs/{id}/status`、`/events?after=&limit=` |
| Gateway 恢复代理 | `services/gateway/httpapi/replay_proxy.go` | `GET /api/runs/{id}/status`、`/events` 转发 Orchestrator |
| Gateway requestId | `gateway/httpapi/server.go`、`orchestratorclient`、`runservice` | 生成 runId/requestId，写入 MySQL chat_requests，重复请求 409；runId/requestId 传给 Orchestrator |
| 断连 durable run | `gateway/httpapi/server.go` | 浏览器断连不取消 runner；继续消费事件并持久化最终 assistant 消息 |
| 公开事件日志 | Gateway `run_public_events`、`/api/runs/{id}/events` | 每个公开事件带 `(seq,subIndex)`，本地日志优先回放 |
| 实时多订阅 | Gateway `/api/runs/{id}/events/stream` | 每个订阅者独立游标 tail SQL 事件；共享投影不抑制其他订阅 |
| 内部 gRPC | `services/orchestrator/grpcapi`、`pkg/runtime/proto` | StartRun/SubscribeRun/Status/Approval/Cancel/ListAgents；订阅读 SQL 公开事件 |
| Gateway gRPC | `orchestratorclient/grpc.go` | `ORCHESTRATOR_GRPC_URL` 时聊天运行走 gRPC；HTTP 保留用于恢复/管理代理 |
| 前端游标恢复 | `frontend/src/agui/client.ts`、`recoverRun` | 记录 runId/lastSeq；流错误时按 after 游标拉取事件，终态停止 |
| Memory 工具 | `pkg/runtime/tools/memory.go` | memory.search/read/propose；scope 由可信代码绑定，不从参数读取 |
| Memory 缓存 | `internal/memory/cache_redis.go` | 键含 scopeID/epoch/limit/policy 摘要；每次召回从 SQL 取 epoch，旧缓存不复活 |
| Memory 管理 API | `httpapi/memory_admin.go` | entries/propose/confirm/update/forget/policy/mutations |
| Gateway Memory 代理 | `httpapi/memory_proxy.go` | `/api/memory/*` 转发 Orchestrator，保留状态码 |
| Memory UI | `frontend/src/components/MemoryPanel.tsx` | 模式/scope/状态/修订/来源/到期；候选确认、纠正、单条/范围遗忘与 purgeStatus |
| 测试 | relay、httpapi、orchestratorclient、MemoryPanel | 真实 Redis/MySQL 测试 + 前端组件测试 |

## 2. 验证

```text
$ AGENTHUB_MYSQL_TEST_DSN=... AGENTHUB_REDIS_TEST_ADDR=... \
  go test ./pkg/adk/... ./pkg/runtime/... ./services/gateway/... ./services/orchestrator/... -count=1
全部包 PASS
```

关键用例：

```text
TestRelayPublishesInOrder                                  PASS（真实 Redis：seq 1,2 有序；二次运行不重发）
TestRelayFailureMarksAttemptWithoutPublishingLaterSeqs     PASS（失败后不处理后续 seq）
TestIntegrationOutboxRelayMySQLRedis                       PASS（SQL outbox→Redis Stream；outbox=published）
TestRunLedgerPersistsConversationalLifecycle               PASS
TestChatRequestIdempotency                                 PASS（重复 requestId：409、不重复消息、不重复执行）
TestReplayMethods                                          PASS
TestReplayProxyEndpoints                                   PASS
TestIntegrationRedisRecallCacheUsesEpoch                   PASS（真实 MySQL+Redis：更新/删除后 epoch 变化绕过旧缓存）
TestChatContinuesAndPersistsAfterDisconnect                PASS（写 socket 失败后仍完成并保存 assistant 消息）
TestChatJournalsPublicEventsAndReplays                     PASS（seq 1..n、本地 after 游标回放）
TestIntegrationPublicEventJournal                          PASS（真实 MySQL：重复 seq 更新不重复）
frontend recoverRun tests                                  PASS（3 tests：回放/状态终态/503）
TestEventStreamDeliversToTwoSubscribers                    PASS（两个订阅者各自完整收到 seq 1..3）
TestEventStreamHonorsAfterCursor                           PASS（after 游标过滤）
TestOrchestratorGRPCControlPlane                           PASS（真实 MySQL：Start dedup、Subscribe seq、Status/Approval/Cancel/List）
TestGRPCRunServiceStartsAndMapsEvents                      PASS（Gateway gRPC Start/Subscribe 映射）
TestMemoryToolsBindTrustedScope                            PASS
TestMemoryProxyForwards                                    PASS
TestIntegrationMemoryAdminAPI                              PASS（真实 MySQL：候选→确认→更新→冲突→单条/范围遗忘）
Browser Memory E2E                                         PASS（见第 4 节）
MemoryPanel.test.tsx                                       PASS（7 tests：加载/确认/纠正冲突/单条/范围遗忘/错误/模式）
```

## 3. 浏览器 Memory E2E（本轮执行）

环境：compose 最小栈（MySQL/Redis/Orchestrator/Gateway，`MEMORY_MODE=scoped_long_term`）+ Vite dev server；浏览器 `127.0.0.1:5173`。

实际步骤与结果：

```text
1. Memory 页加载：scope=dev/default/demo-user，mode=Scoped long-term，candidate 可见
2. 点击 Confirm memory → "Memory confirmed and active."，status=active
3. 纠正内容 → rev=2，内容更新，提示 "Memory corrected with a new revision."
4. Forget memory → "Recall disabled: yes; purge status: pending."，status=revoked
5. 模式切换 Off → UI 显示 Off / "No past session reuse"，再恢复 scoped
6. 控制台 error/warn：无
7. MySQL：entry revoked rev2；mutations propose/confirm/update=completed，forget purge_status=pending
```

发现的真实缺陷：`memory.Policy` 缺少 JSON tags，前端读取到 `Mode/Read/...` 导致模式显示 Off、PUT 失败；已补 camelCase tags 并重建验证。

## 4. 未完成（P4 门禁未通过）

1. 内部 gRPC 已实现并被 Gateway 聊天运行使用；审批/取消已通过 gRPC（`ConfirmRun`/`CancelRun`）从 Gateway 转发，回放/管理仍走 HTTP 兼容接口。
2. `event_projection_offsets` 表仍未被消费；公开事件日志已支持游标，但投影/消费位置尚未独立记录。
3. **浏览器断线恢复**：前端未保存/发送 `(runId, seq, subIndex)` 游标；重连不会自动回放。
4. **两个 Gateway OS 进程独立订阅**、慢订阅者有界队列、Redis 重启/清空恢复测试未做。
5. Memory Redis 召回缓存已实现；缓存失效事件未单独发布，靠 SQL epoch 复核保证旧值不复用。
6. 普通断连已不取消 run；前端 `recoverRun` 按 seq 游标回放，但未在真实模型聊天中做浏览器断线演练。
7. `Redis Streams` 消费者侧去重与终态对账未接入 Gateway。
8. 浏览器已用 FakeModel + mock child agent 验证聊天计划卡批准/拒绝（见 P6 报告第 4 节）；真实模型聊天、断线重连、执行中停止按钮与工具事件未做浏览器 E2E。待审批时 Stop 的 500 已修复并有单测，浏览器未复验。

## 4. 结论

可声明“SQL→Redis relay、SQL 回放接口、Gateway 恢复代理与 requestId 幂等已实现并在真实 MySQL+Redis 上验证”；不能声明 P4 完成。
