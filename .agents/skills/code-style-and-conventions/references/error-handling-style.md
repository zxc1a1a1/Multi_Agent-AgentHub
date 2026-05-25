# 错误处理风格

## 目标

错误处理必须做到：

- 开发者能排查。
- 用户能理解。
- 前端能恢复。
- 日志不泄密。
- 错误码稳定。

## 后端错误分层

后端错误分三层：

1. 内部错误：包含上下文，用于日志。
2. 协议错误：稳定 code，用于前后端交互。
3. 用户错误：脱敏 message，用于 UI。

## 后端规则

- 不吞掉 error。
- 不随意 panic。
- 不把第三方完整错误直接返回给前端。
- wrap error 时保留操作上下文。
- 对外错误必须脱敏。
- 可重试错误应标记 retryable。

## 前端规则

- 请求失败必须停止 loading。
- 流式失败必须清理 streaming 状态。
- 用户提示必须可理解。
- 组件异常不能导致整个应用白屏。
- malformed JSON 应安全跳过或显示受控错误。

## 错误码命名

错误码使用大写蛇形：

```text
BAD_REQUEST
UNAUTHORIZED
NOT_FOUND
TIMEOUT
INTERNAL_ERROR
```

项目级错误可以加前缀：

```text
AGENT_UNAVAILABLE
STREAM_INTERRUPTED
TOOL_ARGS_INVALID
```

## 禁止泄漏

错误不得包含：

- API key。
- Authorization token。
- Cookie。
- 数据库连接串。
- 内部堆栈。
- 本地绝对路径。
- 完整 system prompt。
- 内网拓扑。
