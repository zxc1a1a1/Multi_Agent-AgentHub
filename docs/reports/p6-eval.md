# P6 阶段报告：Eval Harness、故障与性能（部分）

- 日期：2026-10-29
- 基线：`dev / 4dc1127`（P5 部分完成后）
- 状态：**FakeModel Eval Harness、真实 MySQL/Redis 集成与前端单测/构建已执行；独立进程故障、真实模型、浏览器 E2E、race 未完成**。

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
| 浏览器 E2E | **PARTIAL** | Memory 管理全流程已通过浏览器验证；聊天/审批/重连因无真实模型未执行 |
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

## 4. 未覆盖的验收

1. 真模型任务完成率/工具选择/成本：未执行，不能用 FakeModel 结果替代。
2. 已做两个 OS 进程独立回放；live 订阅、崩溃接管、旧 fencing 拒绝、relay 崩溃恢复仍未执行。
3. 浏览器聊天主路径、断线重连、审批、取消、工具事件：未执行；Memory 管理已执行（P4 报告第 3 节）。
4. 负载矩阵（1/10/50 并发、100/1k/10k 历史）：未执行，不承诺容量。
5. Race：环境编译器缺失，不能声明通过。

## 5. 结论

P6 的 FakeModel/前端/真实 MySQL+Redis 证据已具备；独立进程、真模型、浏览器、race 与负载门禁未通过，P6 未完成。
