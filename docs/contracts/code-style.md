# AgentHub 代码风格契约

## 1. 文档目的

本文定义 AgentHub 项目的通用代码风格、质量门禁和 AI 生成代码要求。

本文独立于具体业务协议，不定义任何后端或前端协议字段。

## 2. 通用原则

- 代码优先可读。
- 修改保持小步。
- 错误必须处理。
- 外部输入必须校验。
- 日志不得泄密。
- 测试应覆盖错误路径。
- 未执行测试不得声称通过。

## 3. Go

- 使用 gofmt。
- 包名小写、简短。
- error 显式处理。
- context 向下传递。
- 外部 HTTP 调用必须有 timeout。
- handler 保持薄层。
- goroutine 必须可退出。

## 4. TypeScript

- 启用 strict。
- 避免无理由 any。
- 外部 JSON 使用 unknown 后校验。
- API 调用集中在 service/client 层。
- 流式请求必须能取消。

## 5. React

- 组件 PascalCase。
- Hook 使用 useXxx。
- Props 显式类型。
- 通用 UI 组件不直接请求 API。
- 高风险渲染独立封装。

## 6. Markdown / JSON / YAML

- Markdown 标题层级连续。
- JSON 示例必须合法。
- YAML 不写死 secret。
- JSON Schema 使用 `*.schema.json`。

## 7. AI 生成代码

AI 生成代码必须：

- 只修改授权范围。
- 不引入无关依赖。
- 不大规模重排无关文件。
- 不留下不可运行伪代码。
- 不声称未执行测试已通过。

## 8. 完成标准

一次代码变更至少应说明：

- 改了什么。
- 为什么改。
- 修改了哪些文件。
- 执行了哪些验证。
- 哪些验证未执行及原因。
