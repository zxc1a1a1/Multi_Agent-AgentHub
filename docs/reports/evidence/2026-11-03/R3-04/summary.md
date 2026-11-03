# R3-04 JSON Schema 表单

- task: R3-04（`docs/plans/v3-gap-atomic-plan.md` §4.4）
- 对应缺口：GAP-11；UI-04
- 环境：Windows；Node 20.12.2；`@rjsf/core@6.11.0` + `@rjsf/validator-ajv8@6.11.0`（Apache-2.0）
- overall: **PASS**（41/41 相关 Vitest；前端全量 329/329；构建通过）

## 1. 许可与包体

- `@rjsf/core` Apache-2.0，unpacked 2,473,734 B；`@rjsf/validator-ajv8` Apache-2.0，337,094 B；
  依赖 `ajv@8`、`ajv-formats`、`markdown-to-jsx`。无 AGPL/SSPL。
- 构建产物：`JsonSchemaForm-Cl98065l.js` 394,046 B / gzip 130,592 B，
  为独立懒加载 chunk，`dist/index.html` 未预加载；入口 gzip 160.37 KiB。

## 2. 行为

- `ui.component: json_form` 需要 `runId`、props.toolCallId 与 `user_input` capability，否则降级文本卡片
  （`form_context_missing` / `user_input_capability_missing`）。
- `sanitizeFormSchema` 白名单关键字：`type/title/description/properties/required/enum/default/const/minimum/maximum/minLength/maxLength/items/additionalProperties`；
  拒绝 `$ref/$id/$schema/format/oneOf/anyOf/allOf/patternProperties` 等。
- 限制：深度 ≤5、字段 ≤30、enum ≤50、string maxLength ≤4096（缺省自动补 4096）；
  拒绝 `__proto__/constructor/prototype`、`javascript:`/`data:` 默认值与 enum。
- 只使用 RJSF core + ajv8 validator，无自定义 widget；不执行 schema 中的表达式。
- 提交调用既有 `sendToolResult`（`/api/runs/{id}/tools/result`），由服务端按工具 schema 再校验；
  取消提交 `status=cancelled`。

## 3. 测试

- sanitizer：合法 schema、`$ref`、`__proto__`、`javascript:` 默认值、`format: data-url`、深度/字段/enum 上限、非对象根。
- 表单：渲染字段、填写后提交（runId/taskId/toolCallId/data）、取消、恶意 schema 不渲染 input、
  缺 toolCallId 不提交。
- `MessageBubble`：合法上下文渲染 RJSF；缺 toolCallId/capability 降级。
- 全量 329 测试通过；`npm run build` 通过。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 表单渲染/提交/取消 | events.jsonl | 41 | yes |
| 工具结果 API | events.jsonl（mock 断言） | 2 | yes |
| 用户/SSE/HITL/记忆/MQ | - | 0 | not_applicable: 组件级 |

## 5. 边界

- 服务端工具 schema 校验由既有 tool-result 路径负责；本任务未新增后端测试。
- props.toolCallId 只作为标识符使用，服务端必须校验它属于该 run（既有安全边界）。
- 未启用 RJSF 自定义 widget / 远程 schema / 文件上传。

## 6. 回滚

- registry `json_form` renderer 改回 null，MessageBubble 删除分支；依赖可删除。

## 7. 证据

- `docs/reports/evidence/2026-11-03/R3-04/`（commands.txt、events.jsonl、summary.md、logs/）
