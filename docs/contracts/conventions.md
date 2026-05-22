# AgentHub Naming and Workflow Conventions

## 1. 目录命名

目录统一使用 kebab-case。

标准 Skill 目录名：

```text
project-architecture
platform-api-contract
agui-event-contract
gateway-orchestrator-contract
a2a-agent-contract
frontend-runtime-skills-contract
artifact-contract
data-persistence-contract
intent-orchestration-contract
adk-runtime-contract
security-boundary-contract
code-style-and-conventions
ai-collaboration-workflow
```

注意：标准名称是 `platform-api-contract`，不建议使用 `platform-contract`。如果历史目录已经使用旧名，应在 README 中说明别名或后续统一迁移。

## 2. React 命名

- 组件：PascalCase，例如 `ChatWindow`、`CodePreview`。
- hooks：`useXxx`，例如 `useAguiRun`。
- store：`useXxxStore`，例如 `useConversationStore`。
- 运行时 Skill 组件放在 `runtime-skills/`。

## 3. Go 命名

常见类型建议：

```text
GatewayServer
AGUIHandler
Orchestrator
OrchestratorRequest
OrchestratorResult
EventSink
A2AClient
A2ATask
AgentCard
Artifact
Store
ConversationStore
MessageStore
```

避免：

```text
Manager
Utils
Common
helpers 大杂烩
service 大杂烩
```

## 4. Commit Message

使用：

```text
feat: 简短描述
fix: 简短描述
docs: 简短描述
refactor: 简短描述
test: 简短描述
chore: 简短描述
```

禁止：

```text
update
fix bug
wip
随便改一下
```

## 5. TODO 注释

推荐格式：

```text
// TODO(mvp): ...
// TODO(contract): ...
// TODO(security): ...
// TODO(post-mvp): ...
```

TODO 必须说明原因和后续动作。

## 6. 测试命名

前端：

```text
*.test.ts
*.test.tsx
```

Go：

```text
*_test.go
```

Contract 测试：

```text
openapi_contract_test.go
agui_event_contract_test.go
a2a_contract_test.go
```
