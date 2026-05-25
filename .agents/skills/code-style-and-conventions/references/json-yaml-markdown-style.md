# JSON / YAML / Markdown 风格

## JSON

- 示例必须是合法 JSON。
- 不在 JSON 中写注释。
- 对外字段使用 camelCase。
- schema 文件使用 `*.schema.json`。
- 必填字段和可选字段必须清楚。
- 示例不要包含真实密钥、真实 token 或真实内部地址。

## JSON Schema

- 推荐使用 Draft 2020-12。
- 顶层声明 `$schema`。
- 顶层声明 `title` 和 `type`。
- 对核心字段写 description。
- 使用 `required` 明确必填字段。
- 对枚举字段使用 `enum`。
- 尽量避免无限宽松的 `additionalProperties: true`。

## YAML

- 用于配置时必须稳定。
- 不把 secret 写死到示例。
- secret 使用环境变量占位。
- 缩进保持一致。
- 布尔值使用 `true` / `false`。
- 列表项保持同一层级。

## Markdown

- 一个文档只使用一个 H1。
- 标题层级不要跳跃。
- 代码块必须标注语言。
- 表格列数保持一致。
- 中文文档使用中文标点。
- 中英文混排时保留英文专有名词。
- Contract 文档应包含：目的、边界、字段或规则、示例、禁止事项、Review Checklist。

## 历史 Profile

如果文档包含历史 Profile，必须明确它是历史基线。

禁止把历史 Profile 写成当前开发禁令。
