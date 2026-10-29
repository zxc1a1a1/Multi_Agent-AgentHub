# Security Boundaries Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Boundary rules

- Browser talks only to Gateway.
- Gateway never exposes internal Orchestrator/Agent URLs to browser.
- Orchestrator never receives browser tokens directly; Gateway authenticates public traffic.
- Child Agent AgentCards must not leak secrets.
- ADK must not contain concrete API keys or provider secrets.
- Runtime provider configs use env references.

## Secret ban

Forbidden in contracts, AgentCards, events, logs, and skills:

```text
API keys, DSNs, bearer tokens, stack traces with env, localhost admin URLs, system prompt secrets
```

## v2 扩展边界

- **Memory 授权**：scope 由服务端身份生成；固定 Token 不宣称个人隔离；模型不能自报 principal/scope，不能直接 approve/delete 或修改策略。
- **Tool Broker**：所有副作用经 Broker；审批绑定 toolCallId/参数 hash/工作区/版本/有效期，一次性消费；未批准的写调用实际执行次数必须为 0。
- **工作区隔离**：只读工具限制在受控根目录；路径规范化、符号链接检查；拒绝凭据文件（`.env*`、`*.pem`、`*.key`、`id_rsa`）。
- **Sandbox**：结构化 executable/argv、非 root、受限挂载/网络/资源；不在宿主机直接运行不可信项目脚本；不开放任意 shell/SQL/公网部署。
- **SSRF**：`web.fetch` 解析并校验目标 IP，拒绝 private/loopback/link-local/metadata 网段；每次重定向重新校验，不能只检查原始 URL。
- **MCP**：只接入审核后的服务器/工具；annotations 不是授权；返回内容按不可信数据处理；不自动安装第三方服务。
- **提示注入**：网页/源码中的指令不得触发追加授权、读取凭据、修改策略或提升权限。
- **前端**：CodePreview 只展示文本；禁止 `eval`/`new Function`/执行用户生成脚本。
- **错误脱敏**：不返回内部 URL、stack trace、DSN、provider 原始错误全文或凭据。
