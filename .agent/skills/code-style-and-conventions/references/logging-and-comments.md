# 日志与注释规则

## 日志目标

日志用于排查问题和观察运行状态，不用于泄漏内部实现或用户隐私。

## 推荐日志字段

```text
requestId
traceId
runId
conversationId
messageId
agentName
operation
errorCode
durationMs
```

字段是否存在取决于当前模块上下文，不要求所有日志都包含全部字段。

## 日志级别

- debug：开发调试，默认生产可关闭。
- info：关键生命周期事件。
- warn：可恢复异常或降级。
- error：失败且需要排查的问题。

## 禁止记录

- API key。
- token。
- Cookie。
- 数据库密码。
- 私有对象存储签名 URL。
- 完整用户敏感输入。
- 完整 system prompt。

## 注释原则

注释解释“为什么”，不是重复“是什么”。

应该注释：

- 兼容逻辑。
- 临时折中。
- 安全边界。
- 不直观状态机。
- 外部协议兼容原因。

不需要注释：

```go
// return error
return err
```

## TODO 规则

TODO 必须包含上下文：

```text
TODO(v1.1): 将临时 inline 内容迁移到后端受控存储。
```

禁止：

```text
TODO: later
TODO: fix
```
