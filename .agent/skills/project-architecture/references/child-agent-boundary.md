# Child Agent Boundary

Child Agent 是通用能力提供服务，不以具体名称定义架构。

## 负责

- 声明 identity。
- 声明 capabilities。
- 暴露 health。
- 接收 task。
- 返回文本、状态、Artifact 或 ToolCall 引用。

## 禁止

- 自己决定全局编排。
- 直接访问 Frontend。
- 直接访问 Gateway 用户 API。
- 控制前端组件。
- 返回未声明能力的产物。

## 示例

代码类 Agent、网页类 Agent、文档类 Agent 只能作为示例，不是长期固定枚举。
