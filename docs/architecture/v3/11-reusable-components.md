# 11 可复用现成组件清单

- 状态：待评审
- 原则：**能复用协议/库，不直接复用整套平台**；先核验许可证、维护状态、安全公告与 Go/TS 支持

## 1. 协议与接口

| 组件 | 用途 | 采用方式 | 风险/备注 |
|---|---|---|---|
| AG-UI | 前端 Agent 事件协议 | 继续使用；固定事件 schema | 自研客户端需兼容 seq/subIndex |
| A2A | Orchestrator↔Child Agent | 已用；外部 Agent 复用 | 不暴露浏览器 |
| MCP | 工具/数据接入 | adapter allowlist | 固定规范版本；2026-07-28 已发布 |
| OpenAI function calling | 模型工具调用 | provider 已映射 tools | 模型能力需探测；Flash 返回 400 |
| Anthropic tool use | Claude 工具调用 | 独立 schema adapter | 不假设与 OpenAI 完全兼容 |

## 2. 后端与基础设施

| 组件 | 用途 | 现状 | 建议 |
|---|---|---|---|
| grpc-go | 内部 RPC | 已有 | 保留；补拦截器/OTel |
| go-redis | Redis | 已有 | 保留 |
| go-sql-driver/mysql | MySQL | 已有 | 保留；连接池/超时参数化 |
| OpenTelemetry Go | traces/metrics/logs | 未接 | P0 接入 |
| Temporal | durable workflow | 未接 | 条件引入，先评估 |
| gVisor | 沙箱隔离 | 未接 | 写工具前接入 |
| MinIO/S3 | 制品存储 | 未接 | 多节点前接入；注意许可证 |
| Playwright | 浏览器自动化 | 测试已用 | 生产受控容器 |
| Firecrawl 等检索 | 网页抓取 | 未接 | 数据出域/成本评估 |

## 3. 前端

| 组件 | 用途 | 现状 | 建议 |
|---|---|---|---|
| React 18 | UI | 已有 | 暂不升级大版本 |
| Zustand | 本地状态 | 已有 | 保留瞬时状态 |
| TanStack Query | 服务端状态 | 未用 | 渐进引入 |
| TanStack Virtual | 长列表 | 未用 | 消息列表优先 |
| react-markdown + remark-gfm | Markdown | 已有 | 加 XSS 策略 |
| highlight.js | 代码高亮 | 已有 | 可换 Shiki（按需） |
| Monaco | 代码编辑 | 未用 | 仅编辑场景 |
| Playwright | E2E | 已有 | 保留 |

## 4. IM 渠道

| 组件 | 用途 | 建议 |
|---|---|---|
| 企业微信官方文档/API | 回调、应用消息、webhook | 先做 adapter；SDK 需核验 |
| 飞书官方文档/API | 事件订阅、卡片、webhook | 先做 adapter；SDK 需核验 |
| 通用 HTTP/crypto | 验签/解密 | 优先标准库，减少依赖 |

## 5. 许可证与合规

- MinIO 社区版为 AGPL v3（官方仓库描述），商用/分发前必须法务评审；可评估 S3 兼容替代。
- 任何 SDK 引入前检查：许可证、最近提交、CVE、是否内置遥测、是否读取环境变量。
- 禁止引入来源不明的闭源 Agent/插件。
- 版本固定并记录 lockfile/摘要；升级走 ADR。

## 6. 不建议直接复用

| 方案 | 原因 |
|---|---|
| 整套 Agent 平台（Dify/LangFlow 等） | 边界冲突、迁移成本、许可/运维 |
| Kafka | 当前事件量不足 |
| Service Mesh | 服务规模不足 |
| 向量数据库 | 召回评测不足 |
| 通用工作流引擎 | 现有 SQL 账本+lease 已覆盖核心 |

## 7. 组件准入清单

- [ ] 许可证可接受
- [ ] 维护活跃（近 6 个月提交/发版）
- [ ] 有安全公告渠道
- [ ] Go/TS 版本兼容
- [ ] 可被 adapter 隔离
- [ ] 不默认外发数据
- [ ] 有卸载/替换路径
- [ ] 有最小 PoC 与压测

## 8. 官方来源

- AG-UI：https://docs.ag-ui.com/concepts/architecture
- MCP：https://modelcontextprotocol.io/docs/2026-07-28/getting-started/intro
- OpenTelemetry：https://opentelemetry.io/docs/
- Temporal：https://docs.temporal.io/
- gVisor：https://gvisor.dev/docs/
- MinIO：https://github.com/minio/minio
- TanStack Virtual：https://tanstack.com/virtual/latest/docs/introduction
- 企业微信：https://developer.work.weixin.qq.com/document/path/90930
- 飞书：https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
- Claude Agent SDK：https://code.claude.com/docs/en/agent-sdk/overview
