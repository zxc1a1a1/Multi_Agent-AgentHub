# 10 外部编码 Agent 接入（Claude Code 等）

- 状态：待评审
- 目标：把专业编码 Agent 作为可选 Child Agent 接入，不污染核心编排与前端
- 默认：**关闭**，需合规、沙箱、预算审批后启用

## 1. 候选方案

| 方案 | 形态 | 优点 | 代价/风险 |
|---|---|---|---|
| 现有 code-agent | 平台内 Go Agent | 可控、易审计 | 编码能力有限 |
| Claude Code / Agent SDK | 本地 CLI 或 Python/TS SDK | 强代码能力、工具生态 | 外部模型、许可/费用、代码出域 |
| 其他 CLI Agent | 子进程 | 接入快 | 版本/协议不稳定、权限难控 |

官方 Agent SDK 提供与 Claude Code 相同的工具、Agent loop 与上下文管理，支持 Python/TypeScript；Go 侧建议用 sidecar/子进程适配，不把 TS SDK 嵌入 Go 服务。

## 2. 推荐接入形态

```text
Orchestrator
  -> A2A Adapter (claude-code-agent)
     -> sidecar process / Agent SDK wrapper
        -> isolated workspace (read-only source + writable temp)
        -> model provider
     <- event stream (text/tool/artifact)
```

适配器必须实现与现有 Child Agent 相同的 A2A 契约：任务输入、流式事件、工具调用、制品、取消、超时、错误脱敏。

## 3. 权限与沙箱

- 工作区：只读挂载源码，独立可写临时目录；禁止宿主 home。
- 网络：默认无网络；模型调用走受控代理/网关。
- 命令：allowlist + 超时 + 输出上限；高风险命令需审批。
- 文件：禁止读取 `.env`、SSH、云凭证；路径穿越/符号链接校验。
- 进程：非 root、资源限制、独立用户；生产用 gVisor/Firecracker。
- 取消：收到 cancel 终止子进程/会话，不继续计费。

## 4. 数据与合规

- 明确代码/仓库内容是否允许发送到外部模型；不允许则只能本地模型。
- 不把平台密钥传入 Agent prompt；由 sidecar 自身环境注入。
- 记录模型、token、费用、工具调用与制品来源。
- 输出必须经过脱敏；不得把原始 provider 错误暴露给前端。
- 外部 Agent 生成的文件先入隔离区，经审批再写回。

## 5. 选型矩阵

| 维度 | 现有 code-agent | Claude Code | 其他 CLI |
|---|---|---|---|
| 可控性 | 高 | 中 | 低 |
| 编码能力 | 中 | 高 | 视产品 |
| 数据出域 | 无 | 有 | 有 |
| 许可证/条款 | 自有 | 需审 | 需审 |
| 运维复杂度 | 低 | 中 | 中 |
| 建议 | 默认 | 可选高级 | 按需 |

## 6. 实施阶段

1. **P0 设计**：A2A 适配契约、沙箱、预算、审计、开关。
2. **P1 PoC**：单仓库只读任务，无网络，人工审批命令。
3. **P2 受限可用**：允许写临时工作区、生成 patch，人工合并。
4. **P3 扩展**：多 Agent、并行任务、成本配额。

## 7. 验收

- [ ] 未启用时 0 外部进程/0 外发
- [ ] 代码/密钥不进入 prompt 日志
- [ ] 无网络时任务明确失败，不绕过
- [ ] 取消后进程终止
- [ ] 产物可追溯（agent/version/model/run）
- [ ] 预算超限终止
- [ ] 错误对外脱敏
- [ ] 与现有 A2A/AG-UI 事件兼容

## 8. 风险

- 供应商条款/许可变化；需定期复核。
- 版本升级导致 CLI/事件格式变化；适配层需契约测试。
- 成本不可控；必须有 token/调用预算。
- 代码出域合规；默认关闭。

## 9. 来源

- Claude Agent SDK：https://code.claude.com/docs/en/agent-sdk/overview
- Claude 平台文档：https://platform.claude.com/docs/en/home
- A2A 契约：`docs/contracts/a2a-task.md`
- 沙箱：https://gvisor.dev/docs/
