# Agent Config Contract

版本：v0.1-mvp  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP + 后续正式开发演进  

## 1. 目的

本文定义 AgentHub 子 Agent 的 `config.yaml` 契约。

`config.yaml` 是 Agent 身份、AgentCard 生成、Runtime 行为和权限边界的重要输入。

## 2. 基础字段

每个 Agent 必须声明：

```yaml
name: code-agent
displayName: Code Agent
description: Generate and explain code.
version: 0.1.0
```

## 3. AgentCard 配置

每个 Agent 必须声明：

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
- `inputModes` 必须和 handler 支持的输入一致。
- `outputModes` 必须和 Runtime 输出能力一致。
- AgentCard 不得包含 secret、内部路径或内部服务地址。

## 4. Runtime 配置

每个 Agent 必须声明：

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

## 5. MVP code-agent 配置

MVP 最小配置：

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

## 6. 正式开发阶段扩展

新增 Agent 时，必须补充：

- 支持的 Artifact 类型。
- 支持的 skills。
- 是否支持 streaming。
- 是否启用工具。
- 工具权限。
- 安全限制。
- 版本信息。
- 对应 handler 能力。

## 7. 禁止事项

不得：

- 在配置中写入 API key。
- 在配置中写入 token。
- 在配置中写入数据库连接字符串。
- 在配置中写入对象存储私有地址。
- 声明 handler 不支持的 skill。
- 未经安全审查打开 shell、filesystem、network 权限。
