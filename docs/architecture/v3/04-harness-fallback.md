# 04 Harness 兜底机制

- 状态：待评审
- 核心原则：**兜底是 fail-closed + 可审计降级，不是规则回退**

## 1. 不允许的兜底

以下行为一律禁止：

- 模型不可用 → 关键词/正则选 Agent
- 计划解析失败 → 使用上一条缓存计划直接执行
- 工具被拒绝 → 改参数重试或绕过 Broker
- 沙箱不可用 → 退回宿主执行
- Memory 缓存旧版本 → 不核验 SQL epoch 直接注入
- 外部渠道验签失败 → 仍处理消息
- 崩溃/未知工具结果 → 自动生成新调用键重做

## 2. 分层兜底

| 层 | 触发 | 允许动作 | 禁止动作 |
|---|---|---|---|
| L0 配置 | 缺模型/缺 key | 启动失败并给出缺失项 | 静默降级到无模型 |
| L1 规划 | 解析/校验失败 | 一次模型修复；仍失败返回 `ORCHESTRATOR_PLANNER_FAILED` | 规则计划、空计划执行 |
| L2 审批 | 超时/重复/过期 | 过期终态、一次性消费、409/410 | 自动批准 |
| L3 预算 | 超模型/工具调用上限 | 终止并记预算原因 | 重置预算继续 |
| L4 工具 | 未授权/沙箱不可用/崩溃 | 拒绝、`unknown`、保留 run | 宿主执行、自动重放 |
| L5 模型 | 超时/限流/5xx | 有限重试 + 脱敏错误；可选显式 fallback provider | 无审计切换、暴露原始错误 |
| L6 投递 | SSE 断连/慢消费者 | SQL 回放、游标恢复、有界等待 | 把断连当取消、无界队列 |
| L7 Memory | Redis 丢失/旧缓存 | 回源 SQL、按 epoch 失效 | 用旧缓存注入 |
| L8 IM 渠道 | 验签失败/重复/限流 | 拒绝、去重、重试有上限 | 信任未验签 body |

## 3. 预算与租约

- 每次 run 有 `max_model_calls`、`max_tool_calls`、token/时间上限。
- 预算在 SQL 事务内预留；恢复不得重置。
- 租约带 TTL 与 `fencing_version`；旧 worker 被拒。
- checkpoint 只存可恢复状态，不存密钥与完整敏感内容。

## 4. 工具兜底

```text
Descriptor -> Guard -> (write?) Approval+Budget+Fencing -> Dispatch -> Execute -> Result
                                                    \-> crash -> unknown (no re-execute)
```

- `succeeded/failed` 可复用；`unknown` 需人工/状态核对。
- 写工具幂等键与参数哈希绑定；参数变化必须新审批。
- 沙箱不可用：明确 `unavailable`，不执行。
- 网络工具：SSRF 校验每次重定向；无 provider 时明确不可用。

## 5. 模型兜底

- 保留“单次修复 prompt”而非规则替换。
- Provider fallback 必须显式配置、记录原因、保持预算与审计。
- 对外只返回稳定错误码，不返回 provider 原始 body。
- 模型工具能力必须探测；DeepSeek Flash 带 tools 返回 HTTP 400，因此默认关闭工具。

## 6. 投递与恢复

- SQL 事件日志是权威；Redis/内存只做加速。
- 游标包含 `(runId, seq, subIndex)`，同 seq 的剩余片段不能跳过。
- 浏览器断连不取消 run；显式 cancel 才取消。
- 恢复先读状态/checkpoint，再决定是否继续。
- IM 渠道回调必须验签、解密、去重；失败不进入编排。

## 7. 观测与告警

| 信号 | 用途 |
|---|---|
| `run_id/request_id/trace_id` | 串联全链路 |
| 规划失败率/修复率 | 判断模型/prompt 退化 |
| 审批等待时长/超时率 | 判断 HITL 体验 |
| 工具拒绝/unknown/重放 | 判断安全与一致性 |
| SSE 断连/回放次数 | 判断投递质量 |
| 预算终止 | 判断成本异常 |
| Memory 命中/epoch 失效 | 判断缓存正确性 |

## 8. 兜底验收清单

- [ ] 无模型启动失败
- [ ] 计划解析失败不派发
- [ ] 审批重放只生效一次
- [ ] 预算恢复不重置
- [ ] 未授权工具 0 调用
- [ ] 沙箱不可用不宿主执行
- [ ] 未知结果不重做
- [ ] 断连后 run 可恢复
- [ ] 验签失败消息不处理
- [ ] 告警与日志不含密钥

## 9. 当前证据

- `pkg/runtime/harness/budget.go`、`state.go`、`context.go`
- `services/orchestrator/internal/toolruntime/broker.go`
- `docs/reports/p3b-execution-ledger.md`、`p5-tools-sandbox.md`
- 缺口：真实 Agent 崩溃接管、Flash tools、live 推送、race。
