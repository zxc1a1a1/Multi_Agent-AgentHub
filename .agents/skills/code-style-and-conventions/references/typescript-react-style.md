# TypeScript 与 React 风格规则

## TypeScript 基础

- 必须启用 strict。
- 避免无理由 `any`。
- 外部输入优先使用 `unknown`，验证后再转换。
- 禁止长期 `@ts-ignore`。
- 公共类型集中定义或由 schema 生成。
- API 边界必须有明确 request / response 类型。

## any 规则

允许短期使用 `any` 的情况：

- 与第三方库兼容，且没有合理类型定义。
- 临时迁移代码，且有 TODO 说明移除条件。
- 测试中构造部分 mock，且不会泄漏到生产代码。

禁止：

- 用 `any` 绕过类型错误。
- 在 store state 中使用大面积 any。
- 在 API response 中直接使用 any。

## 类型命名

- 类型、接口、组件使用 PascalCase。
- 变量、函数使用 camelCase。
- Props 使用 `XxxProps`。
- State 使用 `XxxState`。
- Request / Response 使用 `XxxRequest` / `XxxResponse`。
- 枚举值和稳定常量使用统一风格，不同文件不得混用。

## API 调用

- API 调用集中在 service/client 层。
- 组件不直接拼复杂 endpoint。
- fetch 错误必须处理。
- 外部 JSON 必须有安全解析。
- 流式请求必须支持取消。

## React 组件

- 组件文件使用 PascalCase.tsx。
- 组件 props 必须显式定义。
- 页面组件负责组合。
- 通用 UI 组件不直接请求 API。
- 展示组件不处理复杂协议转换。
- 高风险渲染封装成独立组件。

## Hooks

- 自定义 Hook 必须以 `use` 开头。
- Hook 只能在组件顶层或其他 Hook 中调用。
- Hook 不隐藏危险副作用。
- Hook 返回值应清晰稳定。
- 复杂 Hook 需要测试。

## 状态管理

- Store state 必须有 interface/type。
- 不保存无法序列化的大型运行时对象，除非有明确原因。
- 请求状态、错误状态、loading 状态应可恢复。
- 长流式请求应能按会话或任务隔离状态。

## 错误边界

- 高风险组件应使用 Error Boundary 或局部 fallback。
- 错误提示应用户可理解。
- 不要因为一个预览组件崩溃导致整个应用白屏。
