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
| Memory 工具 | `pkg/runtime/tools/memory.go` | memory.search/read/propose；scope 由可信代码绑定，不从参数读取 |
| Memory 管理 API | `httpapi/memory_admin.go` | entries/propose/confirm/update/forget/policy/mutations |
| Gateway Memory 代理 | `httpapi/memory_proxy.go` | `/api/memory/*` 转发 Orchestrator，保留状态码 |
| Memory UI | `frontend/src/components/MemoryPanel.tsx` | 模式/scope/状态/修订/来源/到期；候选确认、纠正、遗忘与 purgeStatus |
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
TestMemoryToolsBindTrustedScope                            PASS
TestMemoryProxyForwards                                    PASS
TestIntegrationMemoryAdminAPI                              PASS（真实 MySQL：候选→确认→更新→冲突→遗忘）
MemoryPanel.test.tsx                                       PASS（6 tests：加载/确认/纠正冲突/遗忘/错误/模式）
```

## 3. 未完成（P4 门禁未通过）

1. **内部 gRPC**（StartRun/SubscribeRun/control）未实现；当前仍是 HTTP/SSE 兼容路径。
2. **Gateway 事件投影与消费位置**：`event_projection_offsets` 表未接线；没有共享投影去重与 subIndex 游标。
3. **浏览器断线恢复**：前端未保存/发送 `(runId, seq, subIndex)` 游标；重连不会自动回放。
4. **两个 Gateway OS 进程独立订阅**、慢订阅者有界队列、Redis 重启/清空恢复测试未做。
5. **Memory Redis 缓存**与失效事件未实现；前端管理 UI 已具备但未做浏览器 E2E。
6. **取消语义**：普通断连仍会取消本地执行（未实现 durable run 继续/恢复）。
7. `Redis Streams` 消费者侧去重与终态对账未接入 Gateway。

## 4. 结论

可声明“SQL→Redis relay、SQL 回放接口、Gateway 恢复代理与 requestId 幂等已实现并在真实 MySQL+Redis 上验证”；不能声明 P4 完成。
