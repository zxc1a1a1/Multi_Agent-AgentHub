# security-boundary-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目的安全边界，包括前后端鉴权、A2A 内部通信、API Key 管理、LLM Provider 密钥、Artifact 权限、iframe sandbox、file upload、confirm_action、run_command、XSS 防护和错误信息脱敏。

一句话：

**AgentHub 中 Frontend、Gateway、Orchestrator、Child Agent、LLM、Artifact、文件和部署能力之间必须有明确安全边界；任何危险操作都不能由 AI 直接绕过。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 设计鉴权。
- 设计固定 Token / JWT。
- 设计 A2A 内部鉴权。
- 保存或读取 API Key。
- 调用 LLM Provider。
- 输出错误信息。
- 设计 Artifact 下载。
- 设计 iframe 预览。
- 设计 file_upload。
- 设计 confirm_action。
- 设计 run_command。
- 设计 deploy。
- 设计 sandbox。
- 设计权限校验。
- Review 是否泄漏 token、API key、stack trace。
- Review 子 Agent 是否越权访问 workspace。

---

## 3. 核心原则

AgentHub 安全规则：

```text
Frontend 不可信
Child Agent 不完全可信
用户输入不可信
LLM 输出不可信
Artifact 内容不可信
文件上传不可信
工具执行不可信
错误信息不能泄漏内部细节
高危操作必须 confirm_action
密钥只能来自环境变量或安全密钥管理
```

---

## 4. 四份设计文档解释原则

本 Skill 同时遵守：

1. **v1.1 Skills 设计规范**：要求 P0 中定义安全边界。
2. **PDR**：定义完整目标中的 JWT、A2A internal auth、API Key、sandbox、XSS 等安全需求。
3. **MVP 文档**：允许 v0.1 使用固定 Token，不做完整用户注册登录。
4. **UML 文档**：定义 Frontend → Gateway → Orchestrator → Code-Agent → LLM 的调用链路。

解释规则：

```text
MVP 可以简化鉴权方式。
安全边界不能因为 MVP 而取消。
```

---

## 5. 核心文件

本 Skill 落地后应生成或维护：

```text
docs/contracts/security-boundaries.md
docs/contracts/auth-policy.md
docs/contracts/sandbox-policy.md
```

可选维护：

```text
docs/contracts/secret-management.md
docs/contracts/xss-policy.md
docs/contracts/file-upload-policy.md
docs/contracts/security-review-checklist.md
```

---

## 6. MVP v0.1 安全范围

MVP v0.1 允许：

- 使用固定 Token 或环境变量 Token。
- 不做完整用户注册登录。
- 不做完整 JWT 签发 / 刷新。
- 不做复杂 RBAC。
- A2A 内部通信可以先在 Docker 内网中运行。
- 暂不开放 file_upload。
- 暂不开放 run_command 给真实用户。
- 暂不实现部署发布。
- 暂不使用对象存储公开下载。

MVP v0.1 仍然必须：

- Frontend 只能调用 Gateway。
- Gateway 鉴权 `/api/*`。
- Token 不得出现在 query string。
- 错误信息不得泄漏 stack trace / token / API key。
- LLM API key 只能来自环境变量。
- Child Agent 日志不得打印 API key。
- Artifact 预览必须考虑 XSS。
- `code_preview` 默认只展示代码，不执行代码。
- `web_preview` 如果后续实现，必须 iframe sandbox。
- A2A endpoint 不暴露给 Frontend。
- Gateway handler 不直接调用 A2A endpoint。

---

## 7. Post-MVP 安全目标

Post-MVP 应扩展：

- JWT 登录。
- Refresh token。
- 用户权限。
- A2A service-to-service auth。
- API Key 加密存储。
- workspace sandbox。
- run_command 白名单。
- file_upload 类型 / 大小 / 病毒扫描。
- object storage 私有下载 URL。
- confirm_action 高危操作确认。
- audit log。
- rate limit。
- CSRF / CORS 更严格控制。
- Agent 权限隔离。
- 自建 Agent sandbox。

---

## 8. Frontend 安全边界

Frontend 允许：

- 调用 Gateway REST API。
- 连接 Gateway AG-UI SSE。
- 执行已注册的 Frontend Runtime Skill。
- 展示 `code_preview`。
- 在 sandbox 中展示网页预览。

Frontend 不允许：

- 直接调用 Orchestrator。
- 直接调用 Child Agent。
- 直接调用 A2A endpoint。
- 直接访问 LLM Provider。
- 保存 LLM API key。
- 在 query string 中传 token。
- 执行 LLM 生成的任意 JS。
- 自动执行危险 Tool Call。
- 绕过 `confirm_action`。

---

## 9. Gateway 安全边界

Gateway 负责：

- 对外鉴权。
- CORS。
- requestId / traceId。
- Rate limit。
- REST API 权限校验。
- AG-UI endpoint 鉴权。
- 用户资源访问控制。
- 错误脱敏。
- 向 Orchestrator 传递安全上下文。

Gateway 不负责：

- 保存明文 API key。
- 直接执行 LLM 生成命令。
- 直接调用 Child Agent。
- 直接信任 Frontend 传来的 AgentName / toolName。
- 在错误中输出内部堆栈。

MVP v0.1：

```text
Authorization: Bearer <fixed-token>
```

Post-MVP：

```text
Authorization: Bearer <jwt>
```

---

## 10. Orchestrator 安全边界

Orchestrator 负责：

- 校验 Agent 是否来自 Registry / 配置白名单。
- 控制可调用的 Agent。
- 控制可传给 Agent 的上下文。
- 校验 Frontend 声明的 Tools / Skills 是否白名单。
- 将 A2A 错误脱敏后转为 RUN_ERROR。
- 不把敏感 system prompt 传给不可信 Agent。
- 不把用户 token 传给 Child Agent。

Orchestrator 不允许：

- 直接暴露给 Frontend。
- 直接处理浏览器鉴权。
- 直接信任 LLM 输出的 AgentName。
- 让 LLM 编排不存在的 Agent / skill。
- 把 Gateway Authorization token 放进 A2A metadata。
- 把完整内部错误传给 Frontend。

---

## 11. Child Agent 安全边界

Child Agent 不完全可信。

Child Agent 必须：

- 只通过 A2A endpoint 被 Orchestrator 调用。
- 暴露 AgentCard，但不泄漏密钥。
- 不打印 LLM API key。
- 不打印完整敏感 prompt。
- 不访问超出 workspace 的文件。
- 不执行未授权命令。
- 不返回恶意 HTML / JS 作为可信内容。
- Artifact 输出必须经过 Orchestrator / Artifact Contract / Frontend Runtime Skill 处理。

Post-MVP 自建 Agent：

- 必须有权限隔离。
- 必须有 sandbox。
- 必须有限制的工具权限。
- 必须有 resource limit。
- 必须有审计日志。

---

## 12. A2A 内部通信安全

MVP v0.1：

- A2A 可运行在 Docker 内部网络。
- 不暴露给公网。
- 不暴露给 Frontend。
- Gateway handler 不直接调用。

Post-MVP：

- A2A 调用必须使用 service-to-service token 或 mTLS。
- A2A metadata 不得携带用户 token。
- A2A 请求必须携带 traceId。
- A2A endpoint 必须限流。
- A2A 错误必须脱敏。
- AgentCard 不得泄漏密钥或内部敏感配置。

---

## 13. API Key / Secret 管理

密钥包括：

- LLM API key。
- JWT signing secret。
- Service token。
- Object storage credentials。
- Database password。
- OAuth secret。
- Deploy token。

规则：

- 密钥只能来自环境变量或安全密钥管理。
- 不得写进 Git。
- 不得写进 AgentCard。
- 不得写进 OpenAPI 示例。
- 不得写进 Contract 示例。
- 不得打印到日志。
- 不得回传给 Frontend。
- `.env.example` 只能写变量名，不写真实值。
- AI 生成代码不得硬编码密钥。

---

## 14. 错误信息安全

对外错误不得包含：

- stack trace。
- API key。
- token。
- 数据库连接串。
- 内部路径。
- 内部服务地址。
- 完整 system prompt。
- 未脱敏用户隐私。

AG-UI `RUN_ERROR` 只能包含：

```text
code
message
runId
```

REST ErrorResponse 只能包含：

```text
code
data: null
message
```

详细错误写日志，但日志也必须脱敏。

---

## 15. Artifact 安全

Artifact 不可信。

规则：

- `code_preview` 只展示代码，不执行。
- `web_preview` 必须使用 iframe sandbox。
- HTML 内容必须隔离。
- 文件下载必须鉴权。
- 私有 Artifact URL 必须短期有效。
- 图片 / 文件必须校验 MIME type。
- 大文件必须限制大小。
- Artifact metadata 不得包含密钥。
- deploy Artifact / deploy action 必须走 `confirm_action`。

MVP v0.1：

```text
只实现 code_preview，不执行代码。
```

---

## 16. Frontend Runtime Skill 安全

不同 Skill 风险不同：

| Skill | 风险 | 要求 |
|---|---|---|
| `code_preview` | 低 | 只展示，不执行 |
| `web_preview` | 高 | iframe sandbox |
| `file_download` | 中 | 鉴权 URL / MIME 校验 |
| `terminal_output` | 中 | 只展示输出 |
| `confirm_action` | 高 | 必须用户确认 |
| `file_upload` | 高 | 类型 / 大小 / 扫描 |
| `deploy_status` | 高 | 不得自动部署 |
| `form_input` | 中 | 校验输入 |

规则：

- 交互类 Skill 的 ToolResult 必须校验。
- 高危 Skill 必须阻塞 run 等用户确认。
- LLM 不能绕过用户确认直接执行危险操作。
- `confirm_action` 必须用于部署、命令执行、文件覆盖等操作。

---

## 17. run_command 安全

Post-MVP 如果支持 `run_command`：

必须：

- sandbox。
- workspace 根目录限制。
- 命令白名单。
- 超时。
- 输出长度限制。
- 禁止访问系统敏感路径。
- 禁止网络扫描。
- 禁止读取密钥文件。
- 禁止持久后台进程。
- 必须用户确认或策略允许。

MVP v0.1 不实现 `run_command`。

---

## 18. file_upload 安全

Post-MVP 如果支持 file upload：

必须：

- 文件大小限制。
- MIME type 校验。
- 后缀校验。
- 病毒扫描或安全扫描。
- 私有存储。
- 用户权限校验。
- 不允许直接作为可执行文件运行。
- 不允许直接注入 prompt 而不做过滤。

MVP v0.1 不实现 `file_upload`。

---

## 19. iframe sandbox 规则

`web_preview` 必须使用 sandbox。

建议：

```html
<iframe sandbox="allow-scripts">
```

根据需求谨慎增加：

```text
allow-forms
allow-downloads
allow-same-origin
```

禁止默认允许：

```text
allow-top-navigation
allow-popups
allow-modals
```

规则：

- iframe 内容不得直接访问父页面 token。
- iframe 不得共享主站 localStorage。
- 用户生成 HTML 不能在主 DOM 直接 innerHTML 执行。
- 预览与主应用必须隔离。

---

## 20. CORS / CSRF

MVP v0.1：

- Gateway 只允许前端开发域名。
- AG-UI endpoint 需要鉴权。
- Token 不放 query string。

Post-MVP：

- CORS 白名单。
- SameSite cookie 策略。
- CSRF token 或 Bearer token 明确策略。
- 预检请求正确处理。
- 不允许 `Access-Control-Allow-Origin: *` 配合 credentials。

---

## 21. 日志与审计

日志必须包含：

```text
requestId
traceId
runId
conversationId
messageId
agentName
a2aTaskId
errorCode
```

日志不得包含：

```text
Authorization
API key
service token
database password
full system prompt
raw uploaded secret file
```

Post-MVP 审计日志应记录：

- 高危 Tool Call。
- confirm_action。
- file_upload。
- deploy。
- run_command。
- 自建 Agent 修改。
- API key 更新。

---

## 22. Contract Test / Security Test

必须测试：

- 未鉴权 REST API 返回 401。
- 无权限资源返回 403。
- Token 不在 query string。
- 错误不包含 stack trace。
- AG-UI `RUN_ERROR` 不泄漏内部错误。
- A2A endpoint 不可从 Frontend 访问。
- AgentCard 不包含密钥。
- Artifact preview 不执行代码。
- iframe 使用 sandbox。
- `confirm_action` 高危操作需要用户确认。
- 文件下载需要权限。

MVP v0.1 至少检查：

- 固定 Token 有效。
- 无 Token 请求失败。
- API key 不在代码中。
- RUN_ERROR 不泄漏 stack trace。
- code_preview 不执行代码。

---

## 23. 与其他 Skills 的协作

- REST API 必须声明鉴权，错误响应必须脱敏。
- AG-UI `RUN_ERROR` 不能暴露 stack trace / token / 内部地址。
- Gateway 不把用户 token 传给 Orchestrator / Child Agent，除非 Contract 明确需要并做脱敏。
- A2A endpoint 不暴露给 Frontend，AgentCard 不泄漏密钥。
- 高危 Frontend Runtime Skill 必须定义确认、阻塞、ToolResult 和失败规则。
- Artifact 预览和下载必须权限校验，HTML 必须 sandbox。
- 敏感数据不得明文入库，密钥不得进入 JSON 字段。

---

## 24. 硬性规则

1. Frontend 不可信。
2. Child Agent 不完全可信。
3. Frontend 不能直接调用 Orchestrator。
4. Frontend 不能直接调用 Child Agent。
5. Frontend 不能直接调用 A2A endpoint。
6. Gateway 必须鉴权对外 API。
7. Token 不得放 query string。
8. LLM API key 只能来自环境变量或安全密钥管理。
9. API key 不得写进 Git。
10. AgentCard 不得泄漏密钥。
11. A2A metadata 不得携带用户 token。
12. 错误不得泄漏 stack trace / token / API key。
13. code_preview 只展示代码，不执行代码。
14. web_preview 必须 iframe sandbox。
15. file_upload 必须大小 / 类型 / 权限校验。
16. run_command 必须 sandbox / whitelist / timeout。
17. deploy / 文件覆盖 / 命令执行必须 confirm_action。
18. Artifact 下载必须鉴权。
19. 日志必须脱敏。
20. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 25. 必须维护的文件

本 Skill 本体：

```text
skills/security-boundary-contract/SKILL.md
```

正式 Contract：

```text
docs/contracts/security-boundaries.md
docs/contracts/auth-policy.md
docs/contracts/sandbox-policy.md
```

可选：

```text
docs/contracts/secret-management.md
docs/contracts/xss-policy.md
docs/contracts/file-upload-policy.md
docs/contracts/security-review-checklist.md
```

---

## 26. Review Checklist

- [ ] 是否明确 Frontend 不可信？
- [ ] 是否明确 Child Agent 不完全可信？
- [ ] 是否明确 MVP 固定 Token？
- [ ] 是否保留 Post-MVP JWT？
- [ ] 是否禁止 token query string？
- [ ] 是否禁止 Frontend 直连 Orchestrator / Child Agent？
- [ ] 是否禁止 Gateway handler 直连 A2A？
- [ ] 是否定义 A2A 内部通信安全？
- [ ] 是否定义 API Key / Secret 管理？
- [ ] 是否定义错误脱敏？
- [ ] 是否定义 Artifact 安全？
- [ ] 是否定义 code_preview 不执行代码？
- [ ] 是否定义 web_preview iframe sandbox？
- [ ] 是否定义 confirm_action？
- [ ] 是否定义 run_command sandbox？
- [ ] 是否定义 file_upload 安全？
- [ ] 是否定义日志脱敏？
- [ ] 是否定义 Security Test？
