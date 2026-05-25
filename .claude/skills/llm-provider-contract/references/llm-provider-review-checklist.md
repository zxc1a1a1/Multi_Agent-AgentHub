# LLM Provider Review Checklist

## 通用性

- 是否没有绑定具体 Agent 名称？
- 是否没有让业务代码直接调用 Provider SDK？
- 是否所有 LLM 调用都经过 Provider Adapter？
- Gateway 是否没有直接调用 LLM Provider？

## Provider / Model

- Provider 是否注册？
- Model 是否注册？
- 能力是否声明？
- disabled Provider / Model 是否不可用？

## 结构化输出

- 是否有 schema？
- 是否本地 JSON Schema validation？
- 校验失败是否不执行下游动作？

## 安全

- API key 是否未进入日志、Prompt、Artifact、数据库普通字段？
- 错误是否脱敏？
- Prompt 模板是否不含 secret？

## 可靠性

- 是否有 timeout？
- retry 是否有上限？
- fallback 是否校验能力？
