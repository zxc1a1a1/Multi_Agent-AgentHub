# 测试风格

## 测试目标

测试应验证行为，而不是复制实现。

测试必须关注：

- 正常路径。
- 错误路径。
- 空输入。
- 非法输入。
- 取消 / 超时。
- 兼容字段。
- 多实体场景。

## Go 测试

- 使用 `_test.go`。
- 测试函数使用 `TestXxx`。
- 多 case 使用 table-driven tests。
- 子测试使用 `t.Run`。
- 外部 HTTP 使用 `httptest.Server`。
- 不依赖真实外部服务。
- 不使用真实 secret。

## TypeScript 测试

- 使用 `.test.ts` / `.test.tsx`。
- 纯函数优先单测。
- Store / reducer 重点测事件聚合、状态变化和错误恢复。
- API client 使用 mock fetch 或测试 server。
- 流式解析必须测拆包、粘包和 malformed JSON。

## React 测试

- 测用户可见行为。
- 不过度测试内部实现。
- 异步 UI 使用 wait。
- 错误状态、loading 状态、空状态都应覆盖。

## Fixture

- fixture 应小而清晰。
- 大型 fixture 放到明确目录。
- fixture 不包含真实密钥或真实用户隐私。
- fixture 名称表达场景。

## 测试报告

如果测试未执行，必须说明：

- 未执行的命令。
- 未执行原因。
- 建议用户本地执行的命令。
