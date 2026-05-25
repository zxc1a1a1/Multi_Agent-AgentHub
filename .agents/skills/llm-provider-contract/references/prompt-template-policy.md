# Prompt Template Policy

本规则只定义 Prompt 模板边界，不定义具体业务 Prompt。

## 推荐字段

```ts
type PromptTemplate = {
  templateId: string
  version: string
  useCase: string
  requiredVariables: string[]
  owner?: string
  status: 'draft' | 'active' | 'deprecated'
}
```

## 规则

- Prompt 模板必须版本化。
- Prompt 变量必须显式声明。
- 用户输入不得无边界拼接到 system prompt。
- Prompt 中不得包含 API key、内部 token、数据库连接串。
- Prompt 变更影响 structured output 时必须同步 schema。
