# Profiles 策略

## 推荐 Profile

```text
default
dev
demo
mock
extra-agents
observability
storage
```

## 语义

- `default`：主 Demo 最小必需服务。
- `dev`：热更新、调试、bind mount。
- `demo`：稳定演示环境。
- `mock`：mock LLM / mock Agent。
- `extra-agents`：可选更多 Child Agent。
- `observability`：日志、指标、追踪。
- `storage`：Redis、Object Storage 等可选依赖。

## 规则

- 默认 profile 不应过重。
- 可选 profile 不应影响默认启动。
- 新增 profile 必须说明用途。
- profile 是否需要 secret 必须说明。
- profile 启动和清理命令必须说明。

## 禁止

- 默认启动所有可选组件。
- 隐式启用重型依赖。
- profile 名称含糊不清。
