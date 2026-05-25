# A2A AgentCard Contract

## 1. 目的

本文档定义 AgentHub 中 Child Agent 的 AgentCard 契约。

AgentCard 用于让 Registry、Planner、Orchestrator 发现和理解 Agent 能力。

## 2. 通用原则

- 本契约不固定 Agent 名称。
- Agent 名称只用于唯一标识，不用于能力判断。
- Agent 能力由 `skills`、`inputModes`、`outputModes`、`capabilities` 声明。
- 新增 Agent 前必须先定义 AgentCard。
- AgentCard 不得泄漏敏感信息。

## 3. 最小 Schema

```json
{
  "name": "agent-name",
  "description": "说明 Agent 能做什么",
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

## 4. 字段说明

| 字段 | 必填 | 说明 |
|---|---:|---|
| `name` | 是 | Agent 唯一名称 |
| `description` | 是 | Agent 能力说明 |
| `url` | 是 | A2A Server 地址 |
| `version` | 是 | Agent 版本 |
| `capabilities` | 是 | 能力开关 |
| `skills` | 是 | Agent 技能列表 |
| `inputModes` | 是 | 支持的输入模式 |
| `outputModes` | 是 | 支持的输出模式 |

## 5. capabilities

推荐字段：

```json
{
  "streaming": true,
  "artifacts": true,
  "tools": false,
  "cancellable": false
}
```

说明：

- `streaming`: 是否支持流式输出。
- `artifacts`: 是否支持 Artifact。
- `tools`: 是否支持 Agent 内部工具。
- `cancellable`: 是否支持取消任务。

v1.0 不强制 `tools` 和 `cancellable`。

## 6. skills

每个 skill 至少包含：

```json
{
  "id": "skill_id",
  "name": "技能名称",
  "description": "技能描述",
  "inputTypes": ["text"],
  "outputTypes": ["text", "code"]
}
```

规则：

- `skills[].id` 是 Agent 对外声明的**能力 ID 事实源**。`TaskPlan.capabilityIds` 必须引用此 ID，`CapabilitySummary.id` 是它的公开摘要投影。
- `id` 必须稳定。
- `description` 应便于 Planner 理解。
- `outputTypes` 必须与 AgentCard.outputModes 兼容。
- 不得将 `toolName`、`artifact.type` 或 `outputMode` 作为 capabilityId。

## 7. inputModes / outputModes

### 7.1 三层边界

`outputModes` 是 Agent 声明可产出的**语义类别**，不是平台归一化后的 `artifact.type`，也不是前端 `toolName`。

```text
AgentCard.outputModes  = Agent 声明可产出的语义类别（Agent 视角）
Artifact.type          = 平台归一化后的产物类型（平台视角）
Runtime toolName       = 前端执行/渲染能力名（前端视角）
```

三者必须通过映射表转换，不得直接等同：

```text
outputMode → artifact.type → previewType → toolName
```

### 7.2 推荐支持类型

v1.0 推荐支持类型：

```text
inputModes: text
outputModes: text, code, webpage, document
```

每个 Agent 只能声明自己真实支持的类型。

### 7.3 映射示例

| outputMode | artifact.type | previewType | toolName |
|---|---|---|---|
| `code` | `code` | `code_preview` | `code_preview` |
| `webpage` | `webpage` | `web_preview` | `web_preview` |
| `document` | `document` 或 `markdown` | `document_preview` 或 `markdown_render` | `document_preview` 或 `markdown_render` |
| `text` | 无 Artifact | — | `markdown_render` / StreamingText |

规则：

- `outputMode` 不得直接当作 `artifact.type`。
- `outputMode` 不得直接当作 `toolName`。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。

## 8. 安全要求

AgentCard 不得包含：

- API key
- Authorization token
- 数据库密码
- 完整 system prompt
- 内部网络敏感信息

## 9. 示例

### code-like agent

```json
{
  "name": "code-like-agent",
  "description": "生成、解释和审查代码",
  "url": "http://code-agent:8081",
  "version": "0.1.0",
  "capabilities": {"streaming": true, "artifacts": true},
  "skills": [
    {
      "id": "code_generate",
      "name": "代码生成",
      "description": "根据需求生成代码",
      "inputTypes": ["text"],
      "outputTypes": ["text", "code"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

### web-like agent

```json
{
  "name": "web-like-agent",
  "description": "生成网页和 UI",
  "url": "http://web-agent:8082",
  "version": "0.1.0",
  "capabilities": {"streaming": true, "artifacts": true},
  "skills": [
    {
      "id": "web_generation",
      "name": "网页生成",
      "description": "生成 HTML/CSS/JS 页面",
      "inputTypes": ["text"],
      "outputTypes": ["text", "code", "webpage"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code", "webpage"]
}
```

以上只是示例，不是硬约束。
