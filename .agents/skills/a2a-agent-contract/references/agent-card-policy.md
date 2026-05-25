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
- `skills[].outputTypes` 必须与 `outputModes` 兼容。
- `outputModes` 只声明真实能力。
- `url` 不得包含 token。
- AgentCard 不得暴露 API key、system prompt、内部 token。
- 新增 Child Agent 前必须先定义 AgentCard。

## 4. 禁止事项

禁止：

- 用 agentName 推断能力。
- 在 AgentCard 中暴露密钥。
- 声明无法真实输出的 outputModes。
- 让 Frontend 直接读取 Child Agent 的 AgentCard。
- 把 AgentCard 当作 OpenAPI 文档。

## 5. Review 要点

- 字段是否完整。
- skills 是否可被 Planner 理解。
- outputModes 是否能映射 Artifact。
- 是否不包含 secret。
- 是否适合 Registry 缓存。
