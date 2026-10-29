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
```

## 3. 未完成（P5 门禁未通过）

1. **真实隔离执行**未接入：没有 Docker/容器 executor；sandbox 当前默认不可用，不能在宿主运行不可信脚本。
2. **MCP adapter** 未实现（服务器/工具 allowlist、schema 版本、审计）。
3. **真实 web provider** 未配置；`web.search` 明确不可用，`web.fetch` 仅完成 SSRF 校验与有界取回。
4. 工具尚未接入 Orchestrator 实际运行/Child Agent 工具配置；前端工具结果消费未端到端验证。
5. 同一写调用重放、审批重放、旧 hash、过期授权、未知结果的**真实账本**测试未完成（当前仅单元边界）。
6. 崩溃后非幂等工具 `unknown` 的端到端恢复未验证。

## 4. 结论

可声明“写工具与 Broker 授权门、SSRF 校验、sandbox 接口与 artifact 接口已实现并有单元测试”；不能声明 P5 完成。
