# Agent Config Contract

## 1. 目的

`config.yaml` 是 Child Agent 的本地能力声明，用于生成或校验 AgentCard，并为 Runtime 初始化提供配置。

## 2. 推荐 Schema

```yaml
name: example-agent
displayName: Example Agent
description: What this agent can do.
version: "0.1.0"
url: "http://example-agent:8080"

runtime:
  streaming: true
  artifacts: true
  llm:
    enabled: true
    providerRef: default
    timeoutSeconds: 120
  tools:
    enabled: false

agentCard:
  inputModes:
    - text
  outputModes:
    - text
    - code
  skills:
    - id: example_skill
      name: Example Skill
      description: What this skill does.
      inputTypes:
        - text
      outputTypes:
        - text
        - code

permissions:
  network: false
  filesystem: false
  shell: false
  browser: false
  deploy: false
```

## 3. 校验规则

- `name` 必填。
- `description` 必填。
- `version` 必填。
- `url` 是 Agent 自身地址，供 Registry / Orchestrator 内部使用。不得通过 Public API 暴露给 Frontend。
- `runtime.streaming` 必填。
- `runtime.artifacts` 必填。
- `agentCard.inputModes` 必填。
- `agentCard.outputModes` 必填。
- `agentCard.skills` 必填。
- permissions 必填且默认最小权限。
- 文件不得包含 secret。

## 4. AgentCard 映射

| config 字段 | AgentCard 字段 |
|---|---|
| name | name |
| description | description |
| url | url（供 Registry / Orchestrator 内部使用，非公开） |
| version | version |
| runtime.streaming | capabilities.streaming |
| runtime.artifacts | capabilities.artifacts |
| agentCard.inputModes | inputModes |
| agentCard.outputModes | outputModes |
| agentCard.skills | skills |

## 5. Review Checklist

- [ ] config 与 Handler 能力一致。
- [ ] config 与 AgentCard 一致。
- [ ] outputModes 没有虚假声明。
- [ ] permissions 没有过度授权。
- [ ] 无 secret。
