# Provider Fallback 规则

MVP 阶段不要求 fallback。

正式开发阶段可以支持：

```text
same model retry
same provider fallback model
cross provider fallback
local model fallback
user-safe failure
```

Fallback 必须显式配置。

Fallback 不得降低安全等级。

Fallback 不得绕过：

- structured output schema validation。
- tool permission。
- content safety。
- secret policy。
- user authorization。

如果主模型用于 structured output，fallback 模型必须也支持所需结构化输出能力，或者必须进入安全失败。

禁止：

- 隐式 fallback。
- 无限 fallback。
- fallback 到未注册 Provider。
- fallback 到 disabled 模型。
- fallback 到低安全等级模型。
- fallback 后跳过 schema validation。
