# Go 风格规则

## 目的

本文件定义 AgentHub Go 代码的基础风格、命名、错误处理、并发和测试要求。

## 格式化

- 所有 Go 代码必须使用 `gofmt`。
- 不手工维护 gofmt 会自动调整的格式。
- import 顺序交给 gofmt / goimports 处理。
- 不把大规模格式化和功能修改混在一起。

## 包名

- 包名使用小写。
- 包名应短且能表达领域。
- 避免下划线。
- 避免 mixedCaps。
- 避免无意义目录名，例如 `utils`、`common`、`helpers`。

## 文件名

- 文件名使用小写。
- 多词使用下划线。
- 测试文件使用 `_test.go`。
- 文件名应表达主要职责。

## 类型与函数

- 导出名称使用 PascalCase。
- 非导出名称使用 camelCase。
- 导出类型和函数必须有有意义注释。
- 接口名应表达行为。
- 不要滥用 `Manager`、`Processor`、`Helper`。

## 错误处理

必须显式处理 error。

推荐：

```go
if err != nil {
    return fmt.Errorf("create task: %w", err)
}
```

禁止无说明忽略错误：

```go
_ = err
```

错误规则：

- 内部错误保留上下文。
- 对外错误必须脱敏。
- 不随意 panic。
- 不吞掉错误。
- 不把第三方服务完整错误原样返回给用户。

## context

- 请求级函数应传递 `context.Context`。
- context 作为第一个参数。
- 外部调用尊重 cancellation。
- goroutine 必须能退出。
- 不把 context 存进长期结构体。

## HTTP Client

访问外部服务必须使用带 timeout 的 client。

禁止在生产路径中长期使用：

```go
http.DefaultClient.Do(req)
```

推荐集中初始化并注入：

```go
client := &http.Client{Timeout: 30 * time.Second}
```

## Handler

HTTP handler 只做薄层：

- 解析请求。
- 参数校验。
- 调用业务服务。
- 写响应。

不要在 handler 中写复杂业务编排、跨系统调用流程或协议转换状态机。

## 并发

- goroutine 必须有退出条件。
- channel 必须明确关闭方。
- 多 goroutine 写 map 必须加锁或使用并发安全结构。
- Mutex 锁范围要小。
- 并发逻辑必须考虑取消和超时。

## 测试

- 优先使用 table-driven tests。
- 外部服务使用 fake、mock 或 httptest。
- 不依赖真实 LLM、真实外部网络或真实密钥。
- 覆盖错误路径和取消路径。
