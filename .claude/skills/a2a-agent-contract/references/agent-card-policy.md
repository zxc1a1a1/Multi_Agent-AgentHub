# AgentCard Policy

## 1. 定位

AgentCard 是 Child Agent 的能力声明文件，用于让 Registry、Planner、Orchestrator 理解该 Agent。

AgentCard 不是私有配置，不得存放密钥、内部 prompt、运行时 token。

## 2. 最小字段

```json
{
  "name": "agent-name",
  "description": "说明该 Agent 的能力",
  "url": "http://agent-service:8081",
  "version": "0.1.0",
  "capabilities": {
    "streaming": true,
    "artifacts": true,
    "tools": false,
    "cancellable": false
  },
  "skills": [
    {
      "id": "skill_id",
      "name": "技能名称",
      "description": "技能描述",
      "inputTypes": ["text"],
      "outputTypes": ["text", "code"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

## 3. 规则

- `name` 是唯一标识，不是能力判断依据。
- `skills[].id` 必须稳定。
- `skills[].id` 是 Agent 对外声明的**能力 ID 事实源**。Orchestrator 的 `TaskPlan.capabilityIds` 必须引用此 ID，Platform API 的 `CapabilitySummary.id` 是它的公开摘要投影。
- `skills[].outputTypes` 必须与 `outputModes` 兼容。
- `outputModes` 只声明真实能力。
- `url` 不得包含 token。
- AgentCard 不得暴露 API key、system prompt、内部 token。
- 新增 Child Agent 前必须先定义 AgentCard。

### 3.1 outputModes 三层边界

`outputModes` 是 Agent 声明可产出的语义类别，属于 Agent 视角。它不等于平台归一化后的 `artifact.type`，也不等于前端 `toolName`。

```text
AgentCard.outputModes  = Agent 声明可产出的语义类别（Agent 视角）
Artifact.type          = 平台归一化后的产物类型（平台视角）
Runtime toolName       = 前端执行/渲染能力名（前端视角）
```

三者转换链路：

```text
outputMode → artifact.type → previewType → toolName
```

映射示例：

| outputMode | artifact.type | previewType | toolName |
|---|---|---|---|
| `code` | `code` | `code_preview` | `code_preview` |
| `webpage` | `webpage` | `web_preview` | `web_preview` |
| `document` | `document` 或 `markdown` | `document_preview` 或 `markdown_render` | `document_preview` 或 `markdown_render` |

规则：

- `outputMode` 是粗粒度语义类别，不得直接当作 `artifact.type`。
- `outputMode` 不得直接当作 `toolName`。
- `artifact.type` 由 ArtifactRegistry 归一化时根据 ArtifactDraft、mimeType、metadata、outputMode 等信息确定。
- `toolName` 由 `preview.previewType` 决定，不由 `outputMode` 直接决定。

### 3.2 capabilityId 语义边界

`AgentCard.skills[].id` 是能力 ID 的事实源。其他字段与该 ID 的关系：

| 字段 | 与 capabilityId 的关系 |
|---|---|
| `AgentCard.skills[].id` | **事实源**，Agent 对外声明的能力 ID |
| `TaskPlan.capabilityIds` | 必须引用 `AgentCard.skills[].id` |
| `CapabilitySummary.id` | `AgentCard.skills[].id` 的公开摘要投影 |
| `skillId`（如保留） | 只能作为 `capabilityId` 的历史别名 |

以下字段**不是** capabilityId，不得混用：

- `toolName` — 前端 Runtime Capability，不是 Agent capabilityId
- `artifact.type` — 平台归一化产物类型，不是 capabilityId
- `outputMode` — Agent 语义类别，不是 capabilityId

## 4. 禁止事项

禁止：

- 用 agentName 推断能力。
- 在 AgentCard 中暴露密钥。
- 声明无法真实输出的 outputModes。
- 让 Frontend 直接读取 Child Agent 的 AgentCard。
- 把 AgentCard 当作 OpenAPI 文档。
- 把 `outputMode` 直接当作 `artifact.type`。
- 把 `outputMode` 直接当作 `toolName`。
- 通过 `agentName` 推断 outputMode 或选择 toolName。
- 让 Gateway 直接持有 Child Agent 调用地址或 service name。
- 把 `toolName` 当作 capabilityId。
- 把 `artifact.type` 当作 capabilityId。
- 把 `outputMode` 当作 capabilityId。

## 5. Review 要点

- 字段是否完整。
- skills 是否可被 Planner 理解。
- outputModes 是否能映射 Artifact。
- 是否不包含 secret。
- 是否适合 Registry 缓存。
- `outputModes` 是否只声明语义类别，没有被直接当作 `artifact.type` 或 `toolName`。
- 是否所有 outputMode → artifact.type → toolName 转换都通过映射表表达。
- `skills[].id` 是否是能力 ID 事实源，是否可被 Orchestrator 引用。
