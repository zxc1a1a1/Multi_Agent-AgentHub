# A2A AgentCard Contract

## 1. 文档目的

本文档定义 AgentHub 中 Child Agent 的 AgentCard 契约。

AgentCard 是 Child Agent 的能力声明，用于告诉 Orchestrator：

- 这个 Agent 是谁。
- 这个 Agent 的 A2A Server 地址是什么。
- 这个 Agent 支持哪些输入类型。
- 这个 Agent 可能输出哪些产物类型。
- 这个 Agent 具备哪些 skills。
- 这个 Agent 是否支持 streaming、artifacts、cancel 等能力。

一句话：

> Child Agent 必须暴露 AgentCard；Orchestrator 必须通过 AgentCard 理解和校验 Agent 能力。

---

## 2. 协议边界

AgentCard 属于：

```text
Orchestrator ↔ Child Agent
```

AgentCard 不属于：

```text
Frontend ↔ Gateway REST API
Frontend ↔ Gateway AG-UI Event Stream
Gateway ↔ Orchestrator Internal Contract
Artifact Schema
Frontend Runtime Skills Schema
ADK Runtime 内部实现
```

规则：

- Frontend 不直接读取 Child Agent 的 AgentCard。
- Gateway handler 不直接读取 Child Agent 的 AgentCard。
- Orchestrator 或 Agent Registry 可以读取 AgentCard。
- AgentCard 不得写入 Frontend REST API 的 `openapi.yaml` 作为前端直连接口。
- AgentCard 的完整结构由本文件维护。

---

## 3. MVP v0.1 范围

MVP v0.1 只强制实现一个 Child Agent：

```text
code-agent
```

MVP v0.1 的 `code-agent` 必须：

- 暴露 `GET /.well-known/agent.json`。
- 返回合法 AgentCard。
- `name` 固定为 `code-agent`。
- `inputModes` 至少包含 `text`。
- `outputModes` 至少包含 `text` 和 `code`。
- `capabilities.streaming = true`。
- `capabilities.artifacts = true`。
- `skills` 至少包含代码生成能力。
- 不泄漏 API key、token、完整 system prompt。

MVP v0.1 暂不强制：

- `web-agent`。
- `doc-agent`。
- `custom-agent`。
- Agent Registry。
- 动态 Agent 发现。
- 复杂 Agent capability 匹配。
- Agent 市场展示。

---

## 4. Post-MVP 扩展方向

Post-MVP 可以扩展：

- `web-agent`
- `doc-agent`
- `custom-agent`
- Agent Registry
- AgentCard 动态拉取
- Agent 健康检查
- Agent capability 匹配
- Agent skill 路由
- 多 Agent ExecutionPlan
- fallback / retry
- Agent outputModes → Frontend Runtime Skills 自动映射
- Agent 版本管理

扩展时必须保持 AgentCard 向后兼容。

---

## 5. AgentCard Endpoint

Child Agent 必须暴露：

```text
GET /.well-known/agent.json
```

响应：

```json
{
  "name": "code-agent",
  "description": "负责生成、解释和审查代码的 Agent",
  "url": "http://code-agent:8081",
  "version": "0.1.0",
  "capabilities": {
    "streaming": true,
    "artifacts": true,
    "tools": false,
    "cancellable": false
  },
  "skills": [
    {
      "id": "code_generate",
      "name": "代码生成",
      "description": "根据用户需求生成代码",
      "outputTypes": ["code", "text"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

---

## 6. 必需字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `name` | string | 是 | Agent 唯一名称 |
| `description` | string | 是 | Agent 简短说明 |
| `url` | string | 是 | A2A Server 地址 |
| `version` | string | 是 | Agent 版本 |
| `capabilities` | object | 是 | 能力声明 |
| `skills` | array | 是 | Agent 技能列表 |
| `inputModes` | array | 是 | 支持的输入类型 |
| `outputModes` | array | 是 | 可能输出的类型 |

规则：

- `name` 必须稳定。
- `name` 是 Orchestrator 路由 Agent 的关键字段。
- `url` 必须指向 A2A Server。
- `version` 必须可用于排查兼容性。
- `skills[].id` 必须稳定。
- `outputModes` 必须真实反映 Agent 能力。
- AgentCard 不得声明 Agent 实际不支持的能力。
- AgentCard 不得包含 API key、token、内部服务密钥、完整 system prompt。

---

## 7. capabilities

推荐结构：

```json
{
  "streaming": true,
  "artifacts": true,
  "tools": false,
  "cancellable": false
}
```

| 字段 | 类型 | MVP 要求 | 说明 |
|---|---|---|---|
| `streaming` | boolean | true | 是否支持流式输出 |
| `artifacts` | boolean | true | 是否可能输出 Artifact |
| `tools` | boolean | false 或省略 | 是否支持外部工具调用 |
| `cancellable` | boolean | false 或省略 | 是否支持取消 task |

MVP v0.1：

```text
streaming = true
artifacts = true
tools = false 或暂不声明
cancellable = false 或暂不强制
```

Post-MVP 可以扩展：

- `multiModal`
- `memory`
- `toolCalling`
- `structuredOutput`
- `healthCheck`
- `authRequired`

---

## 8. skills

推荐结构：

```json
{
  "id": "code_generate",
  "name": "代码生成",
  "description": "根据用户需求生成代码",
  "outputTypes": ["code", "text"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `id` | string | 是 | 稳定技能 ID |
| `name` | string | 是 | 技能展示名 |
| `description` | string | 是 | 技能描述 |
| `outputTypes` | string[] | 是 | 该 skill 可能输出的类型 |

规则：

- `id` 必须稳定。
- `outputTypes` 必须是系统认可的产物类型。
- MVP v0.1 的 `code-agent` 至少包含 `code_generate` 或等价代码生成能力。
- Post-MVP 可以加入 `code_review`、`code_explain`、`test_generate` 等能力。
- `skills` 不能伪造 Agent 实际不支持的能力。

---

## 9. inputModes

推荐全集：

```text
text
file
image
url
```

MVP v0.1 只强制：

```text
text
```

规则：

- `code-agent` 必须支持 `text`。
- MVP 不强制文件、图片、URL 输入。
- Post-MVP 引入文件或图片输入前，必须同步 Artifact / Frontend Runtime Skills / ADK Runtime 约束。

---

## 10. outputModes

推荐全集：

```text
text
code
webpage
file
image
document
diff
terminal
chart
```

MVP v0.1 只强制：

```text
text
code
```

映射关系建议：

| outputMode | Artifact type | Frontend Skill |
|---|---|---|
| `code` | `code` | `code_preview` |
| `webpage` | `webpage` | `web_preview` |
| `file` | `file` | `file_download` |
| `image` | `image` | `image_preview` |
| `document` | `document` | `markdown_render` |
| `diff` | `diff` | `diff_preview` |
| `terminal` | `terminal` | `terminal_output` |
| `chart` | `chart` | `chart_render` |

规则：

- AgentCard 只声明 outputModes。
- Artifact 具体 schema 由 `artifact-contract` 定义。
- Frontend Skill 参数由 `frontend-runtime-skills-contract` 定义。
- Agent 不直接决定前端组件，只声明输出能力。

---

## 11. code-agent MVP AgentCard

MVP v0.1 的 `code-agent` AgentCard 必须满足：

```text
name = code-agent
inputModes 包含 text
outputModes 包含 text, code
capabilities.streaming = true
capabilities.artifacts = true
skills 至少包含 code_generate 或等价能力
```

推荐示例：

```json
{
  "name": "code-agent",
  "description": "负责生成、解释和审查代码的 Agent",
  "url": "http://code-agent:8081",
  "version": "0.1.0",
  "capabilities": {
    "streaming": true,
    "artifacts": true,
    "tools": false,
    "cancellable": false
  },
  "skills": [
    {
      "id": "code_generate",
      "name": "代码生成",
      "description": "根据用户需求生成代码",
      "outputTypes": ["code", "text"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

---

## 12. Orchestrator 校验规则

Orchestrator 在调用 Agent 前应校验：

- AgentCard 是否存在。
- `name` 是否与配置 / registry 一致。
- `url` 是否存在。
- `capabilities.streaming` 是否满足当前调用需求。
- MVP 中 `code-agent.outputModes` 是否包含 `code`。
- `inputModes` 是否包含 `text`。
- `skills` 是否包含目标能力。
- AgentCard 是否没有明显敏感信息。

如果 AgentCard 无效：

```text
不得调用该 Agent
应返回 A2A / Orchestrator 错误
最终映射为 AG-UI RUN_ERROR
```

MVP v0.1 可先通过配置文件静态注册 Agent，但仍应保留 AgentCard endpoint 和校验逻辑。

---

## 13. 安全规则

AgentCard 不得包含：

- LLM API key。
- 用户 token。
- 服务间 token。
- 数据库连接串。
- 完整 system prompt。
- 内部私有 IP 拓扑。
- 不应公开的工具配置。

AgentCard 可以包含：

- Agent 名称。
- Agent 描述。
- A2A Server URL。
- 版本号。
- 能力声明。
- skills 摘要。
- inputModes / outputModes。

---

## 14. Review Checklist

- [ ] `GET /.well-known/agent.json` 是否存在？
- [ ] AgentCard 是否包含 `name`？
- [ ] AgentCard 是否包含 `description`？
- [ ] AgentCard 是否包含 `url`？
- [ ] AgentCard 是否包含 `version`？
- [ ] AgentCard 是否包含 `capabilities`？
- [ ] AgentCard 是否包含 `skills`？
- [ ] AgentCard 是否包含 `inputModes`？
- [ ] AgentCard 是否包含 `outputModes`？
- [ ] `code-agent` 的 `inputModes` 是否包含 `text`？
- [ ] `code-agent` 的 `outputModes` 是否包含 `text` 和 `code`？
- [ ] `capabilities.streaming` 是否为 true？
- [ ] `capabilities.artifacts` 是否为 true？
- [ ] `skills` 是否包含代码生成能力？
- [ ] AgentCard 是否不泄漏 API key / token / system prompt？
- [ ] AgentCard 是否没有声明 Agent 实际不支持的能力？


## 15. v1.1 对齐补充

### 15.1 AgentCard 路径兼容

- MVP / PDR 使用 `/.well-known/agent.json`。
- Post-MVP 可兼容 `/.well-known/agent-card.json`。
- 两个路径返回的 AgentCard 语义应保持一致。

### 15.2 Frontend Runtime 细化边界

AgentCard 只声明 outputModes 与能力；Frontend Runtime Skills 参数与组件映射由后续 `frontend-runtime-skills-contract` 细化。

