# runtime-security

## 目的

本文定义 ADK Runtime 的安全边界。

## Secret 规则

不得在以下位置出现 secret：

- `config.yaml`
- AgentCard
- `/health` response
- Artifact metadata
- 用户可见错误
- 文本流
- 前端 Tool Call 参数
- 普通日志

secret 只能来自：

- 环境变量。
- 受控 secret 管理。
- 本地开发 `.env`，且不得提交仓库。

## 权限规则

默认权限：

```yaml
permissions:
  network: false
  filesystem: false
  shell: false
  browser: false
  deploy: false
```

启用任何危险权限前，必须同步 `security-boundary-contract`。

## HTML / webpage Artifact

- Runtime 只输出 `webpage` Artifact。
- Runtime 不决定 iframe sandbox。
- Handler 不直接输出 `web_preview`。
- 前端渲染安全由 `frontend-runtime-skills-contract` 和 `security-boundary-contract` 管。

## 日志规则

日志可以包含：

```text
traceId
runId
taskId
agentName
errorCode
durationMs
artifactType
```

日志不得包含：

```text
API key
token
Authorization header
完整 system prompt
用户隐私明文
未脱敏 provider error
```

## Review Checklist

- [ ] AgentCard 是否无 secret？
- [ ] /health 是否无 secret？
- [ ] 错误是否脱敏？
- [ ] 工具权限是否最小化？
- [ ] 日志是否不打印敏感信息？
