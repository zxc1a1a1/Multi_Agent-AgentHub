# agent-config

## 目的

本文定义 Child Agent 的 `config.yaml` 约束。

`config.yaml` 是 Agent 的本地能力声明，不是密钥文件，也不是运行时状态文件。

## 推荐结构

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

## 字段规则

| 字段 | 要求 |
|---|---|
| `name` | 必填，稳定，作为 Agent 唯一标识 |
| `displayName` | 可选，用于 UI 展示 |
| `description` | 必填，描述真实能力 |
| `version` | 必填，用于兼容性排查 |
| `url` | 必填或由环境注入，不得包含 secret |
| `runtime.streaming` | 必填，声明是否支持流式输出 |
| `runtime.artifacts` | 必填，声明是否支持 Artifact |
| `agentCard.inputModes` | 必填 |
| `agentCard.outputModes` | 必填 |
| `agentCard.skills` | 必填 |
| `permissions` | 必填，默认最小权限 |

## 禁止事项

- 不得在 `config.yaml` 写 API key。
- 不得写用户 token。
- 不得写数据库密码。
- 不得写完整 system prompt 中的敏感内容。
- 不得虚假声明 skills。
- 不得虚假声明 outputModes。
- 不得将 Agent 名称作为能力判断依据。

## Review Checklist

- [ ] name 稳定。
- [ ] version 存在。
- [ ] skills 与 Handler 能力一致。
- [ ] outputModes 与 Artifact 能力一致。
- [ ] permissions 默认最小权限。
- [ ] 文件中没有 secret。
