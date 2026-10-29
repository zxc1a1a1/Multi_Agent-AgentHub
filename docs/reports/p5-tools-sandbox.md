# P5 阶段报告：写工具、Sandbox 接口与 SSRF（部分）

- 日期：2026-10-29
- 基线：`dev / 7682048`（P4 部分完成后）
- 状态：**patch/sandbox/web/artifact 工具与 Broker 强制审批已实现；真实隔离执行、MCP、真实检索后端与 Agent 接线未完成**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| 受控补丁 | `pkg/runtime/tools/workspace_write.go` | `workspace.apply_patch`：期望 SHA-256、陈旧 hash 拒绝、并发二次校验、原子 rename、根目录/凭据/符号链接检查 |
| Broker 执行门 | `pkg/runtime/tools/broker.go` | `Execute` 先 Authorize 再执行；写工具无审批时执行次数为 0 |
| Sandbox 接口 | `pkg/runtime/tools/sandbox.go` | 结构化 executable/argv、allowlist、60s/输出上限；无可信 executor 时明确不可用，**不退回宿主执行** |
| Docker 隔离执行 | `pkg/runtime/tools/sandbox_docker.go` | 非 root 容器、只读根/工作区、tmpfs、内存/CPU/PID/超时、默认 network=none |
| MCP adapter | `pkg/runtime/tools/mcp.go` | 服务器工具 allowlist、schema 校验、平台策略覆盖 annotations、同一 Broker 审批 |
| Web 工具 | `pkg/runtime/tools/web.go` | `web.fetch` SSRF 校验（协议/用户信息/私网/loopback/link-local/metadata/每次重定向）、大小限制；`web.search` 未配置 provider 明确不可用 |
| Artifact | `pkg/runtime/tools/artifact.go` | `artifact.publish` 内部 artifactId + 1MiB 限制；内存 store 供测试 |
| 测试 | `workspace_write_test.go`、`sandbox_test.go`、`web_test.go`、`artifact_test.go` | 见下 |

## 2. 验证

```text
$ go test ./pkg/runtime/tools/ -count=1
PASS

覆盖：
- 正确 hash 补丁成功；陈旧/缺失 hash 不修改文件；新建嵌套文件
- 路径穿越、凭据路径拒绝；Windows 符号链接测试 SKIP（权限不足）
- 未审批的 patch/sandbox/artifact 调用不触达工具实现（calls=0）
- sandbox 拒绝未列 executable、路径形式 executable、超时夹紧、无 executor 不可用
- SSRF 拒绝 127.0.0.1、::1、10/8、169.254、file/ftp、userinfo；重定向到私网被拒绝
- fetch 输出截断；web.search 无 provider 不生成假结果

$ AGENTHUB_DOCKER_SANDBOX_TEST=1 go test ./pkg/runtime/tools/ -run TestDockerSandbox
PASS（真实 Docker：alpine:3 echo 输出；超时 1s 被终止；network=true 默认拒绝）

$ go test ./pkg/runtime/tools/ -run TestMCP
PASS（白名单、schema、annotations 不作授权、未审批写工具 0 调用）
```

## 3. 未完成（P5 门禁未通过）

1. Docker executor 已实现并验证 benign command；尚未验证恶意代码逃逸、资源极限、挂载逃逸，不能声明完整沙箱安全。
2. MCP adapter 已实现但未接真实 MCP 服务器/安装流程；无服务器注册、审计持久化。
3. **真实 web provider** 未配置；`web.search` 明确不可用，`web.fetch` 仅完成 SSRF 校验与有界取回。
4. 工具尚未接入 Orchestrator 实际运行/Child Agent 工具配置；前端工具结果消费未端到端验证。
5. 写工具与 ledger 的“消费审批+预算+派发意图同事务”尚未接到真实工具执行路径（Broker 内存 Guard 先行）。
5. 同一写调用重放、审批重放、旧 hash、过期授权、未知结果的**真实账本**测试未完成（当前仅单元边界）。
6. 崩溃后非幂等工具 `unknown` 的端到端恢复未验证。

## 4. 结论

可声明“写工具/Broker、SSRF、Docker 隔离执行、MCP allowlist、artifact 已实现并验证”；不能声明 P5 完成（真实 MCP/恶意代码/账本接线缺失）。
