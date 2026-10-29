# P6 阶段报告：Eval Harness、故障与性能（部分）

- 日期：2026-10-29
- 基线：`dev / 4dc1127`（P5 部分完成后）
- 状态：**FakeModel Eval Harness、真实 MySQL/Redis 集成、FakeModel 浏览器聊天/审批/拒绝与前端单测/构建已执行；真实模型、独立进程 live、race、负载未完成**。

## 1. 本轮交付

- `services/orchestrator/eval/harness.go`：确定性 FakeModel 场景运行器（选人、answer、repair、拒绝、越界、预算）。
- `services/orchestrator/cmd/eval`：JSON 输出评测入口，失败退出码非 0。
- `planner` 增加 AllowedAgents 二次校验：模型返回越界 Agent 即使 registry 存在也被拒绝。
- 评测集为固定合成输入，不读取 `.env`，不调用真实模型。

## 2. 测试矩阵状态

| 层次 | 状态 | 证据 |
|---|---|---|
| FakeModel 单元/契约 | **PASS** | `go test ./services/orchestrator/eval/...` 6 场景全过 |
| Memory 存储/调用/控制 | **PASS（单元 + 真实 MySQL）** | `internal/memory`、`persistence/mysql` 集成用例 |
| 实际 MySQL/Redis | **PASS** | 全量测试含 MySQL 8.4/Redis 7.2 容器；见 P2/P4 报告 |
| 独立进程测试 | **PARTIAL** | 两个独立 Gateway OS 进程回放同一 run；两个 HTTP 订阅者各自从 SQL 游标收到完整事件；未做 gRPC live/崩溃接管 |
| Race | **SKIP** | 本机 Go race 依赖的 `clang-22` 路径不存在，`gcc/clang` 包装器均指向缺失文件 |
| 浏览器 E2E | **PARTIAL（FakeModel + mock child agent）** | Chat 计划卡→批准→执行→回复、计划拒绝/取消已通过；Memory 管理通过；真实模型与 Stop-during-approval 未通过 |
| 前端单测/构建 | **PASS** | Node v20.12.2：220 tests passed；`npm run build` 成功（chunk 1.43MB 警告） |
| 基准 | **PASS（合成 FakeModel）** | `BenchmarkFakeModelEvaluation` ≈ 77µs/op；不代表真实模型/容量 |
| 真模型合成集 | **NOT RUN** | 需要用户注入 key 与预算授权，报告不宣称质量 |

## 3. 本轮真实命令

```text
$ go test ./services/orchestrator/eval/... -count=1 -v
PASS（6 scenarios）

$ go test ./services/orchestrator/eval/... -run '^$' -bench . -benchmem
BenchmarkFakeModelEvaluation-…  ~77µs/op

$ (frontend) PATH=/e/node_js:$PATH npm test -- --run
20 files / 220 tests passed

$ (frontend) PATH=/e/node_js:$PATH npm run build
tsc -b + vite build 成功

$ CC=gcc go test -race ...
FAIL（build）：clang-22 missing；race 未执行

$ AGENTHUB_MYSQL_TEST_DSN=... go test ./services/gateway/httpapi/ -run TestTwoGatewayProcessesReplaySameRun -v
PASS（2 个 OS 子进程各自收到 seq 1..3）
```

## 4. FakeModel 浏览器 HITL E2E（2026-10-29）

环境（全部为本地一次性测试值，未写入仓库）：

- compose 最小栈：MySQL 8.4、Redis 7.2、code-agent（mock 响应）、orchestrator、gateway；Gateway 使用 `ORCHESTRATOR_GRPC_URL`，聊天运行走 gRPC。
- Orchestrator 模型指向本地 `cmd/fake-llm`（OpenAI 兼容、固定计划 JSON），不是真实模型。
- 前端 Vite dev server `127.0.0.1:5173`。

浏览器实际操作与结果：

```text
1. 新建会话并发送 "Write a Go function that ..." → 计划卡渲染（Single Agent / code-agent / 1 task）
2. 点击 Approve & Execute → 无 501；出现 "Plan confirmed — executing..."
3. code-agent 返回 mock 文本并显示在聊天流 → 批准→执行→SSE 回复链路通
4. 新会话再次发送 → 计划卡点击 Cancel → "Plan cancelled: User cancelled"，Stop generation 消失
5. 网络：POST /api/runs/{id}/confirm = 200（批准与拒绝各一次）
6. MySQL executions：批准 run=completed、拒绝 run=cancelled；Stop 尝试的 run=failed（与 cancel 500 对应）
```

发现的真实缺陷：`GRPCRunService` 未实现 Gateway HITL 所需的 `ConfirmRun`，点击批准返回 `501 HITL confirmation not supported by configured runner`。修复内容：

- proto `SubmitApprovalRequest` 增加 `confirmed`/`reject_reason` 并重新生成 Go 绑定。
- Gateway gRPC 客户端新增 `ConfirmRun`（action 为空时按 confirmed 映射 approve/cancel）与 `CancelRun`。
- Orchestrator gRPC 服务转发 `confirmed`/`rejectReason`。
- 单测：`TestGRPCRunServiceConfirmRunMapsDecision`、`TestSubmitApprovalForwardsConfirmation`。

未通过项：计划待审批时点击 Stop generation（走 `/api/runs/{id}/cancel`）返回 500。原因：该 run 尚无 dispatched child task，Orchestrator cancel 返回 404；HTTP runner 也是同样行为，属于待审批阶段取消语义缺失，不是 gRPC 回归。计划卡内的 Cancel 按钮可正常取消。

## 5. 未覆盖的验收

1. 真模型任务完成率/工具选择/成本：未执行，不能用 FakeModel 结果替代。
2. 已做两个 OS 进程独立回放；live 订阅、崩溃接管、旧 fencing 拒绝、relay 崩溃恢复仍未执行。
3. 真实模型浏览器聊天、断线重连、执行中取消、工具事件：未执行；FakeModel 下的批准/拒绝与 Memory 管理已执行（P4/P6 报告）。
4. 负载矩阵（1/10/50 并发、100/1k/10k 历史）：未执行，不承诺容量。
5. Race：环境编译器缺失，不能声明通过。

## 6. 结论

P6 的 FakeModel/前端/真实 MySQL+Redis/FakeModel 浏览器 HITL 证据已具备；真实模型、独立进程 live、race 与负载门禁未通过，P6 未完成。
