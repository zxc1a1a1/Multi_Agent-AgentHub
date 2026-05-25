# AgentHub 代码 Review 清单

## 通用

- 是否只修改授权范围内文件？
- 是否没有引入无关依赖？
- 是否没有大规模无关格式化？
- 是否没有泄漏 secret？

## Go

- 是否 gofmt？
- error 是否显式处理？
- context 是否传递？
- HTTP client 是否有 timeout？
- goroutine 是否可退出？

## TypeScript

- 是否符合 strict？
- 是否没有无理由 any？
- 外部 JSON 是否安全处理？
- API 调用是否集中？

## React

- 组件是否 PascalCase？
- Hook 是否 useXxx？
- Props 是否显式类型？
- 高风险渲染是否封装？

## 测试

- 是否覆盖错误路径？
- 是否覆盖边界输入？
- 是否说明未执行测试原因？

## 文档

- 是否中文一致？
- 标题层级是否合理？
- 示例是否可复制？
- 历史 Profile 是否没有变成当前禁令？

## 交付

- 是否有文件清单？
- 是否有变更说明？
- 是否说明使用方式？
