# 03 联调、断链与恢复

- 状态：待评审
- 目标：把“联通”与“断链”变成可验证的状态机，而不是靠重启碰运气

## 1. 故障矩阵

| 故障 | 影响 | 当前语义 | 必须验证 |
|---|---|---|---|
| 模型超时/限流 | 规划或 Agent 失败 | 返回脱敏错误；不切规则 | 超时、429、空响应 |
| Child Agent 不可达 | 任务失败/暂停 | 记录失败，保留 run | 重试/取消/预算 |
| 浏览器断连 | SSE 中断 | run 继续；游标可回放 | 断连后批准、重连 |
| 审批后断连 | 输出可能丢失 | 需验证执行与投影 | 批准→输出→历史 |
| Orchestrator 崩溃 | 运行中断 | 租约/fencing/latest checkpoint | 接管、旧 fencing 拒绝 |
| Gateway 崩溃 | 浏览器断流 | 事件在 SQL，重连可回放 | 两进程回放 |
| MySQL 不可用 | 无法持久化 | 启动失败或明确错误 | 启动 fail-fast |
| Redis 不可用 | relay/cache 降级 | SQL 权威仍可读 | 重启/清空 |
| 工具进程崩溃 | 结果未知 | `unknown`，不自动重做 | 真实账本断言 |
| 审批超时 | 计划过期 | 409/410，不执行 | 时钟与一次性 |

## 2. 状态机

```text
accepted -> planning -> awaiting_approval -> executing -> completed
                    \-> failed          \-> cancelled
executing -> paused（断连/可恢复） -> executing（resume）
```

约束：

1. 只有显式 cancel 才能把 awaiting/executing 变为 cancelled。
2. 断连只影响投递，不影响 run 的持久状态。
3. 终态不可逆；恢复必须从 checkpoint/ledger 读取。
4. 旧 `fencing_version` 的 worker 不得写账本。

## 3. 幂等与去重

| 场景 | 键 | 行为 |
|---|---|---|
| 重复聊天请求 | `request_id` | 返回已有 run，不重复执行 |
| 审批重放 | `idempotency_key` + 审批状态 | 同请求返回缓存；已消费拒绝 |
| 写工具重放 | `tool_call_id + args_hash` | 返回账本结果，不重跑 |
| 事件投影 | `(runId, seq, subIndex)` | 已投影 offset 跳过 |
| IM 回调 | 平台事件 ID + 时间窗 | 去重后入队 |

## 4. 联调步骤（本机）

```bash
# 1. 健康
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8090/health

# 2. 查 run
curl -fsS http://localhost:8080/api/runs/<runId>/status
curl -fsS 'http://localhost:8080/api/runs/<runId>/events?after=0&subIndex=0&limit=200'

# 3. 审批
curl -fsS -X POST http://localhost:8080/api/runs/<runId>/confirm \
  -H 'Content-Type: application/json' \
  -d '{"runId":"<runId>","actionId":"<planId>","planId":"<planId>","action":"approve"}'

# 4. 账本核对（测试库）
# executions / execution_events / plan_revisions / tool_calls / messages
```

## 5. 现有测试与缺口

| 能力 | 现有证据 | 缺口 |
|---|---|---|
| 两 Gateway 进程回放 | `TestTwoGatewayProcessesReplaySameRun` | 真实 Redis live 推送 |
| SQL 多订阅游标 | `TestEventStreamDeliversToTwoSubscribers` | 慢订阅者有界背压 |
| 断连继续持久化 | `TestChatContinuesAndPersistsAfterDisconnect` | 审批中/工具中断连 |
| 审批一次性 | 真实 MySQL 集成用例 | Flash 刷新→批准复验 |
| 工具 unknown | `TestLedgerBrokerMarksUnknownAndDoesNotReexecute` | 真实 Agent 崩溃接管 |
| Relay 顺序 | `TestRelayPublishesInOrder` | publish 后 ack 前崩溃 |
| 浏览器刷新恢复 | `resume.test.ts` + 真模型浏览器（deepseek-chat） | Flash 刷新路径 |

## 6. 已观察问题（必须保留）

- **Flash + tools = HTTP 400**：DeepSeek Flash 当前不接受 tools；code-agent 工具默认关闭，需 `CODE_AGENT_ENABLE_TOOLS=true` 且模型支持。
- **Flash 刷新恢复未复验**：刷新→批准→输出已在 `deepseek-chat`（Pro）验证；`deepseek-flash` 只验证单轮主路径。
- **gRPC SubscribeRun 仍轮询 SQL**：不是实时推送，延迟约 200ms 量级；live 推送是后续项。
- **审批后断连输出**：历史曾出现空输出；根因之一是 Gateway 未收集 `AGENT_TURN_CONTENT`，已修复；仍需 Flash 复验。
- **race 未通过**：本机缺少 clang-22，不能声明 race 通过。

## 7. 恢复检查清单

- [ ] 断连后 run 仍是 durable，不是 cancelled
- [ ] 重连后文本无重复、无缺口
- [ ] 审批只消费一次
- [ ] 工具重放不重跑
- [ ] MySQL/Redis 重启后历史可读
- [ ] 旧 fencing 写入被拒绝
- [ ] 所有错误对外脱敏
- [ ] traceId/runId/taskId 可串联

## 8. 联调记录要求

每次联调必须记录：命令、runId、时间、模型、结果、失败分类（模型/网络/存储/协议/UI），不得只写“已通过”。
