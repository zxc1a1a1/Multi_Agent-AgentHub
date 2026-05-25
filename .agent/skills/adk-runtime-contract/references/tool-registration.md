# tool-registration

## 目的

本文定义 ADK Runtime 的工具注册与权限规则。

v1.0 中工具系统可以保持最小实现，但 Contract 必须为后续新增 Agent 留出安全边界。

## 默认策略

```text
tools.enabled = false
```

任何工具默认不可用。

## 工具声明

```yaml
tools:
  enabled: true
  items:
    - name: example_tool
      description: What this tool does.
      timeoutMs: 10000
      dangerous: false
      requiresConfirmation: false
      permissions:
        network: false
        filesystem: false
        shell: false
        browser: false
        deploy: false
```

## 必填字段

| 字段 | 要求 |
|---|---|
| `name` | 稳定唯一 |
| `description` | 说明能力和副作用 |
| `timeoutMs` | 必填 |
| `dangerous` | 必填 |
| `requiresConfirmation` | 必填 |
| `permissions` | 必填 |

## 权限规则

- `network` 默认 false。
- `filesystem` 默认 false。
- `shell` 默认 false。
- `browser` 默认 false。
- `deploy` 默认 false。
- 未声明权限不得使用。
- 危险权限必须经过 `security-boundary-contract`。

## 禁止事项

- 不得隐式启用工具。
- 不得把 LLM 输出直接当工具参数执行。
- 不得无 timeout 调用工具。
- 不得通过工具读取 secret。
- 不得通过工具访问未授权网络。
- 不得默认启用 shell。
- 不得默认启用浏览器自动化。
- 不得默认启用部署能力。

## Review Checklist

- [ ] 工具是否显式声明？
- [ ] 是否有 timeout？
- [ ] 是否有权限声明？
- [ ] 危险工具是否 requiresConfirmation？
- [ ] 是否遵守 security-boundary-contract？
