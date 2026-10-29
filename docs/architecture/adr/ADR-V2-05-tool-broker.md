# ADR-V2-05：Tool Broker、原生工具优先与受控 MCP

- 日期：2026-10-29
- 状态：已采纳（P0）
- 背景：现有工具多为确定性片段生成，缺工作区读取、外部检索与隔离执行；模型可见工具与实际执行权限未统一治理。
- 决策：所有副作用经 Tool Broker；ToolDescriptor 声明 schema/读写/超时/审批/输出上限；ToolCallEnvelope 绑定 run/task/toolCall/argsHash/workspace/approval/fencingVersion。先交付只读工作区工具，再按 P5 开放补丁、沙箱测试、 web 检索与 artifact.publish。MCP 作为受控扩展，annotations 不是授权。
- 备选：直接给模型任意 shell/公网工具（不可控）；把前端 CodePreview 当执行环境（XSS/浏览器执行风险）；MCP 全量自动接入（供应链与权限风险）。
- 后果：写工具需要审批、幂等键、hash/版本检查与 unknown 状态；未批准的调用执行次数必须为 0；沙箱需独立资源限制。
- 同步契约：`tool-broker.md`、`security-boundaries.md`、`adk-runtime.md`、`agent-config.md`、`artifact-schema.md`。
