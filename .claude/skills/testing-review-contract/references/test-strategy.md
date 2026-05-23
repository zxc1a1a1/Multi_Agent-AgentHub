# 测试策略

## 1. 目的

本文定义 AgentHub 的测试分层。

## 2. 测试层级

长期测试层级：

```text
contract
schema validation
unit
integration
protocol conversion
frontend component
E2E
security
regression
CI quality gate
PR review
```

## 3. MVP 最小测试

MVP 阶段至少需要：

```text
manual demo checklist
minimum backend unit tests
minimum protocol conversion tests
minimum frontend component / hook tests
minimum E2E happy path
docker compose smoke test
secret redaction check
```

## 4. 测试优先级

优先测试：

1. 协议边界。
2. 数据持久化。
3. Artifact 映射。
4. 流式顺序。
5. 错误路径。
6. 安全脱敏。
7. UI happy path。

## 5. 禁止事项

不得只靠手动测试。

不得只测 happy path。

不得让单元测试依赖真实 LLM 或真实外部服务。
