# 参数 Schema 规则

## 1. 目的

本文定义 Runtime Skill 参数 schema 和校验规则。

## 2. 校验时机

参数必须在完整 Tool Call args 聚合完成后校验。

```text
TOOL_CALL_END
→ parse JSON
→ schema validation
→ execute / render
```

## 3. Schema 要求

每个 Runtime Skill 必须定义 `parametersSchema`。

schema 必须明确：

- `type`
- `required`
- `properties`
- `additionalProperties`
- enum / format / maxLength 等必要约束。

## 4. 拒绝规则

必须拒绝：

- 缺少必填字段。
- 字段类型错误。
- 非法 enum。
- 未知危险字段。
- 超大 inline payload。
- 非法 URL。
- 非法 file reference。
- 违反安全策略的字段。
- 向 render-only Skill 传入可执行指令。

## 5. MVP code_preview 参数

```ts
type CodePreviewParams = {
  code: string;
  language: string;
  filename: string;
};
```

必须：

```text
required = ["code", "language", "filename"]
additionalProperties = false
```

## 6. 禁止事项

不得：

- 将任意 JSON 直接传给 React Component。
- 用 TypeScript 类型代替运行时 schema validation。
- 只在开发环境校验参数。
- 参数校验失败后继续执行。
