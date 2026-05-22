# AgentHub Contract 文件清单

## 1. 说明

本目录用于存放 AgentHub 所有协议和契约文件。  
当前文件只列出 Contract 清单，不替代具体 Contract 内容。

具体 Contract 应由对应 Skill 生成：

- platform-api-contract
- agui-event-contract
- gateway-orchestrator-contract
- a2a-agent-contract
- frontend-runtime-skills-contract
- artifact-contract
- intent-orchestration-contract
- adk-runtime-contract
- testing-review-contract

---

## 2. 第一阶段必须准备的 Contract

```text
openapi.yaml
agui-events.md
gateway-orchestrator.md
a2a-agent-card.md
a2a-task.md
frontend-skills.md
artifact-schema.md
intent-orchestration.md
```

---

## 3. 完整推荐 Contract 清单

```text
openapi.yaml
agui-events.md
agui-events.schema.json
gateway-orchestrator.md
gateway-orchestrator-events.md
a2a-agent-card.md
a2a-task.md
a2a-errors.md
frontend-skills.md
frontend-skills.schema.json
artifact-schema.md
artifact.schema.json
intent-orchestration.md
execution-plan.schema.json
adk-runtime.md
testing-review.md
```

---

## 4. Contract first 规则

所有跨服务字段变更必须先改 Contract，再改实现。

开发顺序：

```text
Contract
  → Mock
  → Implementation
  → Contract Test
  → Review
```

---

## 5. 暂时不在本文件中展开

以下内容不要在 project-architecture 阶段过度展开：

- REST API schema 细节
- AG-UI event JSON Schema 细节
- A2A Task 完整字段
- Artifact JSON Schema
- Frontend Runtime Skills 参数 schema
- ADK Runtime config.yaml 细节

这些应由后续对应 Skill 单独生成。


## 6. v1.1 Skills 与 Contract 对齐补充

v1.1 项目级 Skills 总数为 17：

P0（13）:

```text
project-architecture
code-style-and-conventions
ai-collaboration-workflow
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
```

P1（4）:

```text
llm-provider-contract
observability-debugging-contract
testing-review-contract
docker-compose-delivery
```

说明：`frontend-runtime-skills-contract`、`artifact-contract`、`intent-orchestration-contract`、`adk-runtime-contract` 如尚未落地具体文件，由后续对应 Skill 细化。

