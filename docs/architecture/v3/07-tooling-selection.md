# 07 工具、MCP 与沙箱选型

- 状态：待评审
- 目标：统一“模型工具调用”和“平台工具执行”的边界，保证未授权调用为 0

## 1. 现有实现

| 能力 | 位置 | 状态 |
|---|---|---|
| Tool 接口 | `pkg/adk/tool.go` | 已有 |
| Descriptor/Guard/Broker | `pkg/runtime/tools` | 已有 |
| Workspace 读写 | `runtime/tools/workspace*.go` | 读工具 + apply_patch |
| Docker 沙箱 | `sandbox_docker.go` | 已用 alpine 验证 benign command |
| SSRF 防护 | `web.go` | 校验协议/私网/重定向 |
| Artifact | `artifact.go` | 内存/单机存储 |
| MCP adapter | `mcp.go` | allowlist/schema；未接真实服务器 |
| 账本 Broker | `orchestrator/internal/toolruntime` | 审批/预算/派发/unknown |

## 2. 模型工具调用

- OpenAI 兼容请求已支持 `tools`（function/schema）。
- Anthropic 使用独立 tool schema；不能假设完全兼容。
- **DeepSeek Flash 实测**：带 `tools` 返回 HTTP 400；code-agent 工具默认关闭，需 `CODE_AGENT_ENABLE_TOOLS=true`。
- 选择原则：先探测模型能力，再暴露工具；工具失败不能降级为“直接回答”。

## 3. MCP 选型

| 问题 | 建议 |
|---|---|
| 协议版本 | 固定并记录版本；2026-07-28 规范已发布，需核验 SDK 支持 |
| 传输 | stdio 优先（本地受控），HTTP 仅内网/allowlist |
| 注册 | 服务器/工具 allowlist + schema/version 校验 |
| 授权 | 与原生工具同一 Broker、同一审批/预算 |
| 审计 | 记录 server/tool/version/args_hash/结果 |
| 安装 | 不自动安装第三方 MCP 程序 |

## 4. 沙箱选型

| 方案 | 隔离强度 | 成本 | 建议 |
|---|---|---|---|
| 宿主进程 | 无 | 低 | **禁止** |
| Docker 限权 | 中 | 低 | 当前演示 |
| gVisor | 高 | 中 | **生产写工具首选** |
| Firecracker | 很高 | 高 | 多租户/强隔离时评估 |

要求：非 root、只读根/工作区、tmpfs、CPU/内存/PID/时间限制、默认无网络、输出上限、可审计 argv。

## 5. 外部检索与浏览器

- `web.fetch`：SSRF 校验每次重定向；大小/超时限制。
- `web.search`：无 provider 时明确不可用，不生成假结果。
- 浏览器自动化：Playwright（受控容器）；不得直连生产账号。
- 内容抓取服务（如 Firecrawl）需评估数据出域、许可证与成本，默认关闭。

## 6. 制品

- 元数据入 MySQL；内容入对象存储（MinIO/S3）。
- 单机卷仅演示；多节点必须共享存储。
- 发布前做类型/大小/恶意内容检查；下载使用签名 URL。

## 7. 工具接入矩阵

| 工具 | 读写 | 审批 | 沙箱 | 当前 |
|---|---|---|---|---|
| workspace.list/read/search | 读 | 否 | 工作区 | 已有 |
| workspace.apply_patch | 写 | 是 | 工作区+hash | 已有 |
| sandbox.run_tests | 执行 | 是 | Docker/gVisor | 接口已有 |
| web.fetch/search | 外网 | 是（出域） | 网络策略 | SSRF 已有 |
| artifact.publish | 写 | 是 | 存储 | 已有（单机） |
| MCP tools | 取决于 | 是（写） | 取决于 server | adapter 已有 |

## 8. 验收

- [ ] 未审批写工具 0 调用
- [ ] 旧 hash/过期审批拒绝
- [ ] 工具重放不重跑，unknown 不重做
- [ ] 沙箱不可用不宿主执行
- [ ] SSRF 私网/重定向拒绝
- [ ] MCP 未注册工具拒绝
- [ ] 工具参数/结果不含密钥
- [ ] Flash 等不支持 tools 的模型默认不暴露工具

## 9. 来源

- MCP：https://modelcontextprotocol.io/docs/2026-07-28/getting-started/intro
- OpenAI function calling：https://developers.openai.com/api/docs/guides/function-calling
- Anthropic tool use：https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview
- gVisor：https://gvisor.dev/docs/
- MinIO：https://min.io/
