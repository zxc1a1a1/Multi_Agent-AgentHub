---
name: code-style-and-conventions
description: Use when changing TypeScript, React, Go, JSON, OpenAPI, naming, formatting, commit style, tests, or project conventions.
---

# code-style-and-conventions

## 1. Skill 目的

本 Skill 用于定义 AgentHub 全项目的代码风格、命名规范、格式化规则、目录命名、注释规范、提交规范和基础质量门禁。

它不是 React 或 Go 的深度最佳实践替代品，而是 AgentHub 项目的统一风格兜底规则。

一句话：

**所有由 Codex / Claude Code / OpenCode 生成的 AgentHub 代码，都必须在 TypeScript、Go、JSON、OpenAPI、Contract、目录命名和提交说明上保持一致。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 初始化前端 / 后端 / Agent 代码。
- 编写 React + TypeScript 组件。
- 编写 Go Gateway、Orchestrator、A2A Client、ADK Runtime、Child Agent。
- 编写 OpenAPI、JSON Schema、Contract 文档。
- 编写测试。
- 重构目录结构。
- 命名文件、包、类型、字段、函数、接口。
- 生成提交说明。
- Review AI 生成代码风格。
- 检查是否存在 TypeScript / Go / JSON 命名混乱。
- 检查是否跳过 lint / format / typecheck。

---

## 3. 与外部 Skills 的关系

本 Skill 是 AgentHub 全项目统一风格兜底。

外部 Skills 负责更细的技术栈最佳实践：

| 外部 Skill | 作用 |
|---|---|
| shadcn/ui Skill | 组件组合、UI 样式、表单和弹窗 |
| Vercel react-best-practices | React 性能、组件拆分、长列表、hooks |
| Vercel web-design-guidelines | UX、A11y、移动端、暗色模式 |
| samber/cc-skills-golang | Go 深度最佳实践 |
| Speakeasy Skills | OpenAPI / SDK / Contract Test |
| Trail of Bits Skills | 安全 review、静态分析 |

如果外部 Skill 与本 Skill 冲突：

```text
AgentHub 项目协议和命名规则优先。
通用语言最佳实践其次。
```

---

## 4. 四份设计文档解释原则

本 Skill 必须同时遵守：

1. **v1.1 Skills 设计规范**：定义本 Skill 的必要性和基本规则。
2. **PDR**：定义完整目标架构和长期技术栈。
3. **MVP 文档**：定义当前 v0.1 实施范围和简化实现。
4. **UML 文档**：定义模块关系、时序和关键链路。

解释原则：

```text
PDR 决定长期目标。
MVP 决定当前实现范围。
UML 决定链路和模块命名。
v1.1 Skills 规范决定 AI 开发约束。
```

MVP 可以简化实现，但不能破坏代码风格、命名一致性和 Contract-first 规则。

---

## 5. TypeScript 基础规则

Frontend 必须使用 TypeScript strict mode。

必须遵守：

- `strict: true`
- 禁止无理由使用 `any`
- 禁止长期保留 `// @ts-ignore`
- API response 类型必须来自 OpenAPI 生成结果
- AG-UI Event 类型必须来自 `agui-events.schema.json` 或对应生成类型
- Frontend Runtime Skill 参数类型必须来自 `frontend-runtime-skills.schema.json`
- 组件 props 必须显式定义类型
- 复杂状态必须定义清晰 interface / type
- 不允许把后端 snake_case 字段直接暴露到前端 state 中

允许：

- UI ViewModel 类型由前端自行定义
- 组件内部 props 类型自行定义
- 临时 mock 类型可以存在，但必须标记 `// TODO(contract): replace with generated type`

---

## 6. React 命名规则

React 组件使用 PascalCase：

```text
ChatWindow
ConversationList
MessageBubble
CodePreview
AgentAvatar
```

Hooks 使用 `useXxx`：

```text
useAguiRun
useConversationStore
useStreamingMessage
```

Store 使用 `useXxxStore`：

```text
useMessageStore
useConversationStore
```

文件命名推荐：

```text
PascalCase.tsx       用于组件
camelCase.ts         用于普通工具函数
useXxx.ts            用于 hooks
xxxStore.ts          用于状态 store
```

禁止：

- React 组件小写命名
- 一个文件塞多个大型组件
- 组件里直接散落 API endpoint 字符串
- 组件里手写后端 response 类型
- 组件里混入 A2A / Gateway-Orchestrator 内部协议

---

## 7. React 组件拆分规则

推荐结构：

```text
frontend/src/
  pages/
  components/
    chat/
    agents/
    artifacts/
    runtime-skills/
  hooks/
  stores/
  services/
  types/
```

规则：

- Chat 页面不直接处理 SSE 解析细节。
- SSE 解析放在 AG-UI client / hook 中。
- `code_preview` 等运行时 Skill 放在 `runtime-skills/` 目录。
- API 调用集中到 `services/`。
- 前端不能直接调用 Orchestrator / Child Agent。
- 前端不能引用 A2A 类型作为 API response 类型。
- 前端不能把大 Artifact 直接塞进普通 message state，必须按 Artifact / Skill 契约处理。

---

## 8. Go 基础规则

Go 代码必须遵守：

- 使用 `gofmt` 或 gofmt-compatible formatting。
- 包名使用小写。
- 包名不使用驼峰。
- 包名不使用下划线，除非极少数生成代码需要。
- 文件名使用小写加下划线。
- 导出类型使用 PascalCase。
- 非导出类型使用 camelCase。
- 错误必须显式处理。
- context 必须向下传递。
- 不吞掉 error。
- 不随意 panic。
- 不把复杂业务逻辑写进 handler。
- 必须通过 `go test`。
- 正式代码必须通过 `golangci-lint`。

---

## 9. Go 包边界规则

MVP v0.1 推荐：

```text
server/
  internal/
    handler/
    orchestrator/
    a2a/
    store/
    config/
    middleware/
    model/
```

规则：

- `handler` 只处理 HTTP / SSE / auth / request validate。
- `orchestrator` 处理路由、A2A 调度、ProtocolConverter。
- `a2a` 封装 A2A Client 类型和调用。
- `store` 处理 MySQL / PostgreSQL 持久化。
- `middleware` 处理 auth、requestId、traceId、CORS。
- `model` 放内部数据模型。
- `handler` 不直接调用 Child Agent。
- `handler` 不直接解析 A2A Stream。
- `orchestrator` 不直接写 HTTP response。
- `a2a` 不直接输出 AG-UI Event。

Post-MVP 拆分服务时：

```text
gateway/
orchestrator/
agents/
```

但包边界规则不变。

---

## 10. Go 命名建议

常见类型命名：

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

禁止：

- `Manager` 泛滥
- `Utils` 大杂烩
- `Common` 大杂烩
- `service` 里塞所有逻辑
- `handler` 里直接写 Orchestrator 逻辑
- 包名叫 `helpers` 后塞进所有工具函数

---

## 11. JSON 字段命名规则

所有对外 JSON 字段使用 camelCase。

适用范围：

- REST API response
- AG-UI Event
- A2A Task request / response
- Artifact JSON
- Frontend Runtime Skill args
- JSON Schema
- Contract 示例

推荐：

```json
{
  "runId": "run-001",
  "threadId": "conv-001",
  "messageId": "msg-001",
  "toolCallId": "tc-001",
  "pageSize": 20,
  "createdAt": "2026-05-22T10:00:00Z"
}
```

数据库列名可以使用 snake_case，但 API / Contract / JSON response 不得混用 snake_case。

---

## 12. OpenAPI / JSON Schema 风格

OpenAPI：

- 文件为 `docs/contracts/openapi.yaml`
- 所有 operationId 必须稳定
- component schema 使用 PascalCase
- JSON 字段使用 camelCase
- 错误响应统一
- 分页格式统一
- 鉴权 header 统一
- API 变更先改 OpenAPI

JSON Schema：

- schema ID / title 使用 PascalCase
- event type / toolName / artifact type 使用稳定字符串
- required 字段明确列出
- enum 必须显式声明
- 不允许使用过宽的 `additionalProperties: true`，除非有明确理由
- 不允许 schema 与 markdown contract 脱节

---

## 13. Contract 文档风格

Contract 文档统一放在：

```text
docs/contracts/
```

文档命名：

```text
kebab-case.md
kebab-case.schema.json
```

每个 Contract 文档建议包含：

```text
文档目的
协议边界
MVP v0.1 范围
Post-MVP 扩展方向
字段 / 事件 / endpoint 定义
禁止事项
Review Checklist
```

禁止：

- Contract 文档只写示例不写规则
- Markdown Contract 和 JSON Schema 不一致
- OpenAPI 与 Contract 文档冲突
- 修改协议不更新对应文档

---

## 14. 目录命名规则

目录统一使用 kebab-case。

推荐：

```text
project-architecture
platform-api-contract
agui-event-contract
gateway-orchestrator-contract
a2a-agent-contract
frontend-runtime-skills-contract
artifact-contract
data-persistence-contract
security-boundary-contract
```

注意：

```text
platform-api-contract
```

是标准名称，不建议使用：

```text
platform-contract
```

如果历史目录已经叫 `platform-contract`，后续应统一迁移或在 README 中明确别名。

---

## 15. Commit Message 规范

提交信息使用：

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

AI 生成的提交说明必须准确概括变更，不得夸大完成度。

---

## 16. 注释规范

推荐：

- 对复杂协议转换写注释。
- 对安全边界写注释。
- 对非显然的 MVP 临时简化写注释。
- 对 TODO 标注来源和后续动作。

TODO 格式：

```text
// TODO(mvp): ...
// TODO(contract): ...
// TODO(security): ...
// TODO(post-mvp): ...
```

禁止：

- 给显而易见代码写废话注释。
- 用注释掩盖混乱逻辑。
- TODO 不说明原因。
- 长期保留临时 mock 而不标注。

---

## 17. 测试命名规则

前端测试：

```text
*.test.ts
*.test.tsx
```

Go 测试：

```text
*_test.go
```

Contract 测试命名：

```text
openapi_contract_test.go
agui_event_contract_test.go
a2a_contract_test.go
```

测试原则：

- Contract Test 优先于真实集成。
- Mock-first 阶段必须测 mock 是否符合 Contract。
- 业务实现不能只依赖手动测试。
- AI 生成代码必须补必要测试或说明未补原因。

---

## 18. MVP v0.1 特别规则

MVP v0.1 允许：

- Orchestrator 嵌入 Gateway 进程。
- 后端使用 `server/` 单服务目录。
- MySQL 8 作为当前数据库。
- 固定 Token 或环境变量 Token。
- 配置文件静态注册 `code-agent`。
- 只实现 `code-agent`。
- 只实现 `code_preview`。
- 只实现最小 REST API。
- Mock A2A Agent。
- Mock LLM 输出。

但代码风格不能因此放松：

- 仍然必须保持模块边界。
- 仍然必须遵守 Contract。
- 仍然必须使用 camelCase JSON。
- 仍然必须处理 error。
- 仍然必须传递 context。
- 仍然必须避免 token 泄漏。
- 仍然必须保持可测试。

---

## 19. 硬性规则

1. TypeScript 必须启用 strict。
2. 前端必须通过 ESLint / Prettier。
3. React 组件使用 PascalCase。
4. React hooks 使用 `useXxx`。
5. Go 必须使用 gofmt。
6. Go 包名使用小写。
7. Go 必须通过 go test。
8. Go 正式代码必须通过 golangci-lint。
9. JSON API 字段使用 camelCase。
10. 数据库列名可以使用 snake_case，但 API JSON 不得混用。
11. REST API 类型来自 OpenAPI。
12. AG-UI Event 类型来自 AG-UI Contract。
13. A2A 类型来自 A2A Contract。
14. 不允许 handler 塞复杂编排逻辑。
15. 不允许前端直接调用 Orchestrator / Child Agent。
16. 不允许 Gateway handler 直接调用 A2A endpoint。
17. 不允许把大 Artifact 塞进文本流。
18. 不允许跳过 lint / typecheck / go test 直接宣称完成。
19. 不允许生成未格式化代码。
20. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 20. 必须维护的文件

本 Skill 本体：

```text
skills/code-style-and-conventions/SKILL.md
```

后续可选 references：

```text
skills/code-style-and-conventions/references/frontend-style.md
skills/code-style-and-conventions/references/go-style.md
skills/code-style-and-conventions/references/json-style.md
skills/code-style-and-conventions/references/commit-style.md
```

正式项目规则可选落地：

```text
docs/contracts/code-style.md
docs/contracts/conventions.md
```

---

## 21. Review Checklist

- [ ] TypeScript 是否 strict？
- [ ] 是否没有滥用 `any`？
- [ ] React 组件是否 PascalCase？
- [ ] React hooks 是否 `useXxx`？
- [ ] API response 类型是否来自 OpenAPI？
- [ ] Go 代码是否 gofmt？
- [ ] Go 包名是否小写？
- [ ] Go error 是否显式处理？
- [ ] context 是否向下传递？
- [ ] handler 是否没有复杂编排逻辑？
- [ ] JSON 字段是否 camelCase？
- [ ] Contract 示例是否和 schema 一致？
- [ ] 是否没有把 A2A / AG-UI / REST 类型混用？
- [ ] 是否没有跳过 lint / typecheck / go test？
- [ ] commit message 是否规范？


## References

- `references/go-style.md`
- `references/typescript-react-style.md`
- `references/naming-conventions.md`
- `references/commit-message-policy.md`
- `references/test-style.md`
