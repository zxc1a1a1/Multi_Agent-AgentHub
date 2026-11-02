# R3-01 交互组件协议与白名单 registry

- task: R3-01（`docs/plans/v3-gap-atomic-plan.md` §4.4）
- testId: UI-06（未知组件 → 降级文本卡片）
- 对应缺口：GAP-11；UI-06
- commit: 未提交（按要求只做实现与验证，由用户 review 后提交）
- branch: dev
- node: v20.12.2（PATH 上的 `node` 不可用，改用 `$NODE_HOME/node.exe` + `$NODE_HOME/node_modules/npm/bin/npm-cli.js`）
- 验证目录: `frontend/`（jsdom + Vitest 2 + React Testing Library）
- started: 2026-11-02T09:17:29Z（基线套件）
- ended: 2026-11-02T09:38:32Z
- overall: **PASS**（新增测试 46 条全通过；前端全量 303 条通过；`npm run build` 通过）

## results

| metric | count |
|---|---|
| 基线（改动前）frontend 测试 | 28 files / 257 tests PASS |
| 改动后 frontend 测试 | 31 files / 303 tests PASS |
| 本次新增测试文件 | 3 |
| 本次新增用例 | 46 |
| 失败 | 0 |
| 跳过 | 0 |
| `npm run build`（tsc -b + vite build） | PASS（exit 0） |
| 变异检查（去掉校验） | 3 轮，分别 5 / 2 / 9 条用例失败（预期失败） |
| events.jsonl 行数 | 30 |

## 本批次实际执行的命令与结果

| 命令 | 结果 |
|---|---|
| `npm test -- --run`（改动前基线） | PASS 28 files / 257 tests |
| `npm test -- --run src/agui/skills.test.ts src/stores/uiComponentEvent.test.ts src/components/MessageBubbleUIComponent.test.tsx`（首轮） | FAIL 4/46 —— 三处都是**测试自身**写错：① helper `rejection()` 被用在「应通过」的 `props: Object.create(null)` 用例上；② 期望字符集安全的未知组件名（`xss_card` / `dangerous_iframe`）也被 redact（实际契约只 redact 不安全字符）；③ no-eval 源码扫描正则 `\(\s*props\s*\)` 误命中纯类型判断 `isPlainObject(props)`。修正的是测试期望与正则，未降低任何校验强度 |
| 同上（修正后） | PASS 3 files / 46 tests |
| `npm test -- --run`（改动后全量） | PASS 31 files / 303 tests |
| `npm run build` | PASS，`✓ built in 19.77s`；入口 chunk `dist/assets/index-rODRSL0_.js` gzip 158.84 kB（R3-07 预算门禁不在本任务范围，见「未验证项」） |
| 密钥扫描（见 commands.txt） | 关键词命中 0；`sk-` 形态命中 0；`git ls-files .env` 空输出（.env 未被跟踪） |

## per test（新增）

### frontend/src/agui/skills.test.ts（25 用例）

| 组 | 覆盖 |
|---|---|
| ui.component registry (R3-01) 4 | 白名单恰为 `html_sandbox/json_form/chart/code_editor`；registry 与白名单一致；R3-01 所有 `renderer` 为 `null` 且 `interactive=false`、`acceptsStreamingProps=false`；能力名集合冻结 |
| component allowlist 3 | 合法事件通过且 `props` 原样保留；四个组件全部通过；白名单外（含大小写、空串、非字符串）一律 `component_not_allowed` |
| schemaVersion 1 | 严格 `=== 1`：`2`/`'1'`/`0`/缺失/`null` 全部 `schema_version_unsupported` |
| props 6 | 必须纯对象（array/null/字符串/数字/布尔/函数 → `props_invalid`）；非 `Object.prototype`/`null` 原型（class 实例、`new Date()`）拒绝；64KB 边界（65536 通过 / 65537 拒绝）且按 **UTF-8 字节**计数（21842 个 CJK 字符 = 65537 字节，按字符计数会误放行）；函数/symbol/bigint/getter/循环引用/`toJSON` 一律拒绝且断言「未被调用」；深度 70 拒绝、深度 40 通过； hostile markup 作为惰性数据通过 |
| artifactId 2 | 匹配仓库真实形状（`art_`+hex/ULID 风格，生成端 `pkg/runtime/tools/artifact.go`、`services/orchestrator/httpapi/handler_render.go`）；拒绝 `art_`、`artifact_1`、含空格、`art_<script>`、`art_../etc/passwd`、65 字符超长后缀、非字符串 |
| capabilities 2 | 必须是已知能力名子集（`[]` 合法）；`network_access`/`file_write`/`eval`/非数组/含非字符串项/缺失全部拒绝；返回副本而非别名事件载荷 |
| event shape 2 | 非对象/`type` 不为 `ui.component` → `event_shape_invalid`；组件白名单先于其它字段判定 |
| fallback display tokens 3 | token 清洗（安全名保留、含尖括号/空格/超长/对象 → `<redacted>`、数字转字符串）；拒绝描述只含 `reason/component/artifactId/schemaVersion`，不回显 props 原文 |
| no-eval guarantees 2 | 剥离注释后扫描 `skills.ts` 源码：无 `eval(`、`new Function`、`Function(`、`import(`、`require(`、`dangerouslySetInnerHTML`、`document.write`、字符串型 `setTimeout`；不把 props 当可调用值 |

### frontend/src/stores/uiComponentEvent.test.ts（17 用例）

| 组 | 覆盖 |
|---|---|
| reducer 追加卡片 3 | 合法事件追加 1 张 `ui_component` 卡片（component/schemaVersion/artifactId/capabilities/props/eventId/seq/runId 全部来自校验结果）；`TEXT_MESSAGE_END` 后卡片仍在且消息为 `sent`；不同 eventId/seq → 2 张 |
| 幂等（eventId / seq）5 | 同 `eventId` 重复 3 次 → 1 张；无 eventId 时按 `runId+seq+subIndex` 去重 → 1 张；同 seq 不同 subIndex 视为不同事件 → 2 张；被拒事件同样去重 → 1 张；既无 id 又无 seq 时如实追加（契约禁止 Gateway 下发，见 §11.4） |
| 降级文本卡片 7 | 未知组件 → `unknown_skill`（`toolName: ui_component`、`reason: component_not_allowed`）；hostile 组件名 redact 且不回显 `onerror`；schemaVersion 2 / props 非对象 / props 超 64KB / artifactId 非法 / capabilities 越界 → 均降级且断言 `JSON.stringify(args)` 不含 props 原文、`alert(1)`、`passwd` |
| 不影响既有卡片链路 2 | `TOOL_CALL_END chart_render` + `ui.component` 共存为 `['chart_render','ui_component']`；未知工具名仍走原 `unknown_skill` 文本卡片 |

### frontend/src/components/MessageBubbleUIComponent.test.tsx（4 用例）

| 覆盖 |
|---|
| 合法但未实现渲染器的 `ui_component` 卡片渲染成 `UnknownSkillFallback` 文本卡片（`Tool: ui_component` + `raw output`），容器内无 iframe/script/img，无 `onerror`/`alert(1)`/`internal.example` 文本，`window.eval` spy 未被调用 |
| 拒绝原因卡片展示 `component_not_allowed` 与 `<redacted>`，不回显 props |
| 用户动作：`click` 卡片标题、`click` `raw output` 徽标、键盘 `Tab`/`Enter`/`Escape` —— 卡片内部无 `button/a/input/textarea/select/iframe/script`，Tab 后焦点不落在卡片内，交互后仍只有 1 张卡片且 `eval` 未被调用 |
| 去重后的载荷只渲染 1 张卡片（`ui_component` 分支不输出 `pre`，即不展示 props 原文） |

## 冻结的契约

`docs/contracts/rich-artifact-output.md` §11（新增，80 行）：

1. §11.1 事件形状冻结为
   `{"type":"ui.component","component":"html_sandbox|json_form|chart|code_editor","props":{},"schemaVersion":1,"artifactId":"art_...","capabilities":["read_only"]}`；
   生产者=Gateway，消费者=Frontend `messageStore` reducer；不是 A2A 原始事件透传，不新增公开 REST endpoint。
2. §11.2 字段规则表：component 白名单；`schemaVersion` 严格 1；`props` 纯对象且 UTF-8 ≤65536 字节、深度 ≤64、禁 function/symbol/undefined/bigint/访问器属性；`artifactId` 匹配 `^art_[A-Za-z0-9_-]{1,64}$`（与 Go 生成端和 `artifact_id VARCHAR(64)` 对齐，未自造格式）；`capabilities` 必须是 `read_only|user_input|sandboxed_script` 的子集（`[]` 合法），并列出永久禁止的能力名。
3. §11.3 降级与拒绝：未知/非法 → 既有文本卡片；已注册但无渲染器（R3-01 的四个全部如此）→ 同样文本卡片；降级参数只含 4 个清洗后的字段，不含 props 原文；禁止 `eval`/`new Function`/props URL 动态 `import`/`dangerouslySetInnerHTML`。
4. §11.4 幂等与重放：去重键 `id` 优先，否则 `runId+seq+subIndex`；重放命中同键直接丢弃；无游标事件无法去重（Gateway 不得下发）；只覆盖 `ui.component` 路径。
5. §11.5 回滚：移除 reducer 分支与 registry 即可，旧卡片路径不变。
6. 未触碰 `docs/contracts/00-INDEX.md`、`redis-usage.md`、`event-replay.md`，未创建任何 event-bus 文件（属其它任务）。

## 新增/改动文件

- `docs/contracts/rich-artifact-output.md`（+80，§11）
- `frontend/src/types/uiComponent.ts`（新增：`UIComponentName`/`UIComponentCapability`/`UIComponentAGUIEvent`/`UIComponentCardData`/`UIComponentValidation`/`UIComponentRejectionReason`/`UIComponentFallbackArgs`）
- `frontend/src/types/index.ts`（+22：re-export 上述类型；`AGUIEvent` 增加 `component/props/schemaVersion/artifactId/capabilities` 可选字段并注明「未校验输入，必须先过 validator」；`SkillCardData` 并入 `UIComponentCardData`）
- `frontend/src/agui/skills.ts`（+241：`uiComponentAllowlist`、`uiCapabilityAllowlist`、`uiComponentRegistry`、`UI_COMPONENT_PROPS_MAX_BYTES=64*1024`、`UI_COMPONENT_ARTIFACT_ID_PATTERN`、`validateUIComponentEvent`、`describeUIComponentRejection`、`safeComponentToken`）
- `frontend/src/stores/messageStore.ts`（+60：`appliedUIComponentKeys` 去重集合、`uiComponentIdempotencyKey`、`handleUIComponentEvent`、`case 'ui.component'`）
- 新增测试：`frontend/src/agui/skills.test.ts`、`frontend/src/stores/uiComponentEvent.test.ts`、`frontend/src/components/MessageBubbleUIComponent.test.tsx`
- 证据：`docs/reports/evidence/2026-11-02/R3-01/{commands.txt,summary.md,events.jsonl}`

## 未改文件（确认）

- 未改 `MessageBubble.tsx`：`ui_component` 卡片走其既有 `default` 分支 → `UnknownSkillFallback`；未新增渲染器组件。
- 未改 `UnknownSkillFallback.tsx`、`CodePreview.tsx`、`WebPreview.tsx`、`FormInputCard.tsx`、`ArtifactCard.tsx`、`agui/client.ts`。
- 未改 Gateway/Orchestrator/Child Agent/数据库/compose；未引入任何新依赖；未新增 feature-flag 基础设施。
- 未改既有测试，也未删改任何既有断言（改动前 257 条 → 改动后 303 条，全绿）。
- 工作区中其它文件（`docs/contracts/README.md`、`redis-usage.md`、`event-replay.md`、`docs/contracts/event-bus*.md`、docker-compose、R0-04/R1-01/R2-01 证据等）是并行任务的改动，本任务未写入。

## 变异检查（校验被删除时测试必须失败）

按要求真实执行了三轮，跑完立即用备份还原并复扫（`grep -nE "MUTATION|POSITIVE_INFINITY|&& false"` 无命中），还原后重跑全量 303 条与 build 均通过。

| 轮次 | 临时破坏点 | 真实结果 |
|---|---|---|
| 1 | `skills.ts` 组件白名单判定短路（任何组件名都算合法） | **5 条失败**：`rejects components outside the allowlist`、`checks component before every other field`、`de-duplicates rejected events too`、`unknown component falls through to UnknownSkillFallback data`、`hostile component names are redacted instead of rendered` |
| 2 | props 字节上限比较改成恒不成立（等效删除 64KB 限制） | **2 条失败**：`enforces the 64KB limit on serialized UTF-8 bytes, not characters`、`rejects props above the 64KB limit` |
| 3 | `UI_COMPONENT_ARTIFACT_ID_PATTERN` 改成恒匹配 + `isUIComponentCapability` 只判字符串 + `messageStore` 幂等早退禁用 | **9 条失败**：能力集合冻结、artifactId 形状（skills 2 条 + store 1 条）、capabilities（2 条）、eventId 去重、`runId+seq+subIndex` 去重、被拒事件去重 |

结论：四类校验（白名单、props 上限/形状、artifactId 格式、capabilities 子集）与幂等去重都有真实断言承载，删除即失败。

## redaction（计划 §5.7）

- 证据只写变量名（`$NODE_HOME`、`$NODE_BIN`、`$NPM_CLI`、`$TMP_DIFF`），无任何值。
- 未读取、未打印 `.env`；`git ls-files .env` 空输出确认未被跟踪。
- `events.jsonl` 不含 `prompt/content/text/headers/body/token/dsn/password/cookie/authorization/blob` 等禁止键；props 原文一律不入证据，只记字节数与 id/kind/size。
- 用例中的 artifactId、props、组件名全部为合成值；hostile 样本（`<img onerror>`、`javascript:`、`../../etc/passwd`）为合成攻击面样例，且断言它们不会进入 DOM 文本。

## 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作（click / press_key） | events.jsonl | 6 | yes（jsdom + React Testing Library：点击卡片标题与 `raw output` 徽标，Tab/Enter/Escape） |
| 用户操作（fill） | - | 0 | not_applicable: R3-01 无输入型组件被渲染（`json_form`/`code_editor` 的渲染器属 R3-04/R3-05），本阶段组件事件一律文本卡片，页面内不存在可填充控件 |
| API 请求/响应 | - | 0 | not_applicable: 纯前端 reducer/组件测试，未启动 HTTP 服务；R3-01 不新增 endpoint |
| SSE / AG-UI 事件 | events.jsonl | 14 | yes（`ui.component` 接收/拒绝/重放、`TOOL_CALL_END`、终态 `TEXT_MESSAGE_END`；均带 seq/subIndex/cursor/eventType/idempotencyKey/decision） |
| HITL 请求/信号/Claim | - | 0 | not_applicable: 本任务不涉及审批链路 |
| 工具授权/结果 | - | 0 | not_applicable: 只做未知工具降级回归；组件事件的回传链路属 R3-04（工具输入校验） |
| 记忆召回 | - | 0 | not_applicable: 不涉及记忆 |
| Artifact | events.jsonl | 1 | partial/yes：只记录 `detail.artifact`（合成 id/kind/size）与卡片事件 seq；生成与内容读取 not_applicable——R3-01 只引用 `artifactId`，不读写 artifact blob 或 content 端点 |
| 多实例 | - | 0 | not_applicable: 单前端 reducer 实例（行内 `instanceId=frontend-single` / `jsdom-renderer`）；跨实例事件重放验收属 R1-01/R0-06 |
| 中间件（Redis/MQ）/MySQL | - | 0 | not_applicable: 无后端改动 |
| 真实浏览器（browser 层）/ 截图 / trace | - | 0 | not_applicable: 未运行 Playwright；R3-07 负责键盘与移动端门禁 |
| 测试用例结果 / 变异检查 / 构建 | events.jsonl | 15 | yes（基线、首轮失败、修正后通过、全量 303、build、3 轮变异、还原后复跑、密钥扫描） |

最低要求核对：≥1 条用户动作（6）、≥1 条 SSE 事件（14）、≥1 条终态（`TEXT_MESSAGE_END`）均满足；未运行的层全部显式标注 `not_applicable` + 原因，无留空。

## 已知限制 / 未验证项

1. Gateway/Orchestrator 尚未产生 `ui.component` 事件：本任务只冻结协议与前端消费侧；端到端（真实 SSE 流）未验证，需在渲染器任务里补 E2E。
2. `resumeActiveRun`（页面重载后的 `recoverRun` 重放路径）只重建文本与 activity 状态，不重建 skillCards；`ui.component` 的幂等集合是每次 `sendMessage` 闭包内的，跨重载不共享。属既有架构边界，R3-01 未改写该路径。
3. 无游标（既无 `id` 又无 `seq`）的 `ui.component` 事件无法去重，已在 §11.4 规定为 Gateway 禁止下发，并有测试记录该行为；未在协议层强制拒收（避免拒掉合法的内联卡片），如需更强约束应在评审后调整。
4. 首屏预算：`vite build` 入口 chunk gzip 158.84 kB（含本任务改动，未与改动前做同条件对比，因为工作区含并行任务的在途修改）。R3-01 未新增依赖、未新增懒加载 chunk，预算门禁属 R3-07。
5. R3-01 阶段四个白名单组件均无渲染器，因此「合法组件」当前也渲染为文本卡片；渲染器接入在 R3-02（html_sandbox）、R3-04（json_form）、R3-05（chart），届时只改 registry 的 `renderer` 字段。

## 回滚方式

由评审者执行（本任务未运行任何回滚/丢弃命令）：还原 `frontend/src/agui/skills.ts`、`frontend/src/stores/messageStore.ts`、`frontend/src/types/index.ts`、`docs/contracts/rich-artifact-output.md` 四个文件，删除新增的 `frontend/src/types/uiComponent.ts`、三个测试文件与本证据目录即可。
- 无数据库、无配置、无依赖变化；`ui.component` 分支移除后，未知事件类型不会命中任何 `case`，既有卡片链路行为不变。

## 审核方独立复跑（2026-11-02T09:46Z，提交前）

- `npm test -- --run`（frontend，PATH 前置 /e/node_js）→ **PASS：31 test files / 303 tests passed**
- `npm run build` → **PASS：built in 15.77s**（首屏产物体积见 logs；R3-07 预算门禁负责自动判定）
- `git diff --stat`：`rich-artifact-output.md +80`、`skills.ts +241`、`messageStore.ts +60`、`types/index.ts +22`，**0 删除行**，未触碰无关文件、未格式化既有代码
- 禁路径复核：新增代码中无 `eval(`、无 `new Function`、无 `dangerouslySetInnerHTML`、无以 props URL 为参的 `import()`
  （`messageStore.ts:19-20` 的 `import('../types')` 是既有类型导入，非动态加载）
- 审核方复跑未覆盖：组件渲染器为 `renderer: null`（R3-01 只冻结协议与 registry，实际渲染在 R3-02/R3-04/R3-05），
  Gateway/Orchestrator 尚未发出 `ui.component` 事件，因此本任务**没有端到端证据**，不声称 UI 已可用。
- 变异检查（由实现方执行并记录在 commands.txt：移除 allowlist→5 fail、移除 64KB 限制→2 fail、
  移除 artifactId/capabilities/dedupe→9 fail）。审核方未重复执行破坏性变异，
  但已核对 46 个新用例确实断言这些边界（见 frontend/src/agui/skills.test.ts、stores/uiComponentEvent.test.ts）。
