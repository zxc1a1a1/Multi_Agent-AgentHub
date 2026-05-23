# Agent 配置规则

## 1. 目的

本文定义 AgentHub 子 Agent 的 `config.yaml` 规则。

`config.yaml` 是 Agent 身份、AgentCard 生成、Runtime 行为和权限边界的重要输入。

## 2. 基础字段

每个 Agent 的 `config.yaml` 必须包含：

```yaml
name: code-agent
displayName: Code Agent
description: Generate and explain code.
version: 0.1.0
```

字段说明：

| 字段 | 含义 |
|---|---|
| `name` | Agent 内部名称，建议 kebab-case |
| `displayName` | UI 或 AgentCard 中展示名称 |
| `description` | Agent 能力说明 |
| `version` | Agent 配置版本 |

## 3. AgentCard 字段

必须包含：

```yaml
agentCard:
  inputModes:
    - text/plain
  outputModes:
    - text/plain
  skills:
    - id: code_generate
      name: Code Generate
      description: Generate code from user instructions.
```

规则：

- `skills` 必须和 handler 实际能力一致。
- `inputModes` 必须和 Task handler 能处理的输入一致。
- `outputModes` 必须和 Runtime 能输出的内容一致。
- AgentCard 不得包含 secret。
- AgentCard 不得包含内部路径。
- AgentCard 不得包含内部服务地址。

## 4. Runtime 字段

必须包含：

```yaml
runtime:
  streaming: true
  artifacts:
    - code
  tools:
    enabled: false
  permissions:
    network: false
    filesystem: false
    shell: false
```

规则：

- `streaming` 表示是否支持流式输出。
- `artifacts` 声明该 Agent 可输出的 Artifact 类型。
- `tools.enabled` 表示是否启用工具系统。
- `permissions` 声明 Runtime 权限边界。

## 5. MVP code-agent 配置

MVP 阶段 `code-agent` 的最小配置：

```yaml
name: code-agent
displayName: Code Agent
description: Generate and explain code.
version: 0.1.0

agentCard:
  inputModes:
    - text/plain
  outputModes:
    - text/plain
  skills:
    - id: code_generate
      name: Code Generate
      description: Generate code from user instructions.

runtime:
  streaming: true
  artifacts:
    - code
  tools:
    enabled: false
  permissions:
    network: false
    filesystem: false
    shell: false
```

## 6. 禁止事项

不得：

- 在配置中写入 API key。
- 在配置中写入 access token。
- 在配置中写入数据库连接字符串。
- 在配置中写入对象存储私有地址。
- 在 AgentCard 中声明 handler 不支持的 skill。
- 在没有安全审查的情况下打开 `shell`、`filesystem`、`network` 权限。
