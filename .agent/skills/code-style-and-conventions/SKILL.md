---
name: code-style-and-conventions
description: "用于定义 AgentHub 全项目代码风格、命名规范、格式化规则、测试约定、质量门禁、注释习惯、错误处理和提交信息规范，适用于 Go、TypeScript、React、JSON、YAML、Markdown、Contract 文档和 AI 生成代码 Review。"
---

# code-style-and-conventions

## 1. Skill 目的

本 Skill 用于统一 AgentHub 项目的代码风格、命名规范、工程质量门禁和 Review 标准。

它适用于：

- Go 后端代码。
- TypeScript 前端代码。
- React 组件与 Hook。
- JSON / YAML / Markdown 文档。
- Contract 文档。
- 测试文件。
- 构建脚本。
- AI 生成代码的质量检查。

一句话：

**本 Skill 只规定代码与文档如何保持一致、可读、可维护、可测试；不定义任何业务协议或产品功能。**

---

## 2. 独立性原则

本 Skill 必须独立可读。

本 Skill 不依赖其他 Skill 才能理解。

本 Skill 不定义：

- 后端内部协议字段。
- 前端实时事件字段。
- 子 Agent 运行时接口。
- 产物 schema。
- 数据库表结构。
- Docker 服务拓扑。
- 具体前端组件参数。
- 业务编排算法。

如果某个任务同时涉及专业协议或业务契约，应以用户本轮指定的文件或契约为准。本 Skill 只约束代码风格和质量底线。

---

## 3. 当前阶段识别

本 Skill 不写死当前业务阶段。

规则：

- MVP v0.1 已完成时，只作为历史兼容与回归测试基线。
- 当前活跃阶段由用户本轮任务、Sprint 文档、Issue、PR 描述或项目计划决定。
- 风格规则跨阶段保持稳定。
- 不得把历史阶段限制写成当前开发禁令。
- 不得因为旧代码是 MVP 风格，就继续扩大技术债。

示例：

```text
如果当前任务是 v1.0 Sprint，则不能用“旧 MVP 不做多 Agent”来拒绝当前需求。
如果当前任务只是修复一个 UI bug，则不能顺手重构后端目录。
```

---

## 4. 适用场景

当任务涉及以下内容时，应使用本 Skill：

- 编写 Go 代码。
- 编写 TypeScript 代码。
- 编写 React 组件。
- 新增或修改测试。
- 新增或修改 JSON / YAML / Markdown。
- 新增或修改 Contract 文档。
- 新增或修改构建脚本。
- 重命名文件、目录、类型或字段。
- Review AI 生成代码。
- 检查 PR 是否越界、是否缺少测试、是否风格不一致。
- 生成提交说明或变更报告。

---

## 5. 通用代码原则

所有代码必须遵守：

1. **清晰优先**：不要为了“高级写法”牺牲可读性。
2. **小步修改**：只改和当前任务直接相关的内容。
3. **少魔法**：避免隐式全局状态、隐式副作用和隐式类型转换。
4. **边界清楚**：handler、service、store、component、schema、test 各司其职。
5. **错误可见**：错误必须被处理、记录或向上返回，不得静默吞掉。
6. **测试友好**：外部依赖应可 mock，复杂逻辑应可单测。
7. **命名准确**：变量名、函数名、类型名应表达意图，而不是表达实现细节。
8. **默认安全**：不要打印密钥，不要把 token 写入日志，不要把危险行为隐藏在工具函数中。
9. **兼容演进**：公共字段、公共类型和 Contract 的改动必须考虑向后兼容。
10. **不夸大完成度**：未执行测试不得声称测试通过。

---

## 6. Go 风格

### 6.1 格式化

必须使用 `gofmt`。

规则：

- 不手工调整 gofmt 会覆盖的格式。
- 不为了个人偏好改变缩进、空行或 import 顺序。
- 大范围格式化必须和功能变更分开提交。

### 6.2 包与文件命名

Go 包名：

- 使用小写。
- 简短、明确。
- 避免下划线。
- 避免 mixedCaps。
- 避免 `common`、`utils`、`helper` 滥用。

Go 文件名：

- 使用小写。
- 多词使用下划线，例如 `http_client.go`。
- 测试文件使用 `_test.go`。

### 6.3 类型与函数命名

- 导出类型、函数、方法使用 PascalCase。
- 非导出类型、函数、变量使用 camelCase。
- 接口名应表达行为，例如 `Runner`、`Store`、`Client`。
- 不要用 `Manager`、`Processor`、`Handler` 作为万能后缀。
- 返回布尔值的函数可使用 `Is`、`Has`、`Can`、`Should` 开头。

### 6.4 错误处理

必须显式处理 error。

推荐：

```go
if err != nil {
    return fmt.Errorf("load config: %w", err)
}
```

禁止：

```go
_ = err
```

除非有明确注释说明为什么可以忽略。

错误处理规则：

- 不吞掉 error。
- 不随意 panic。
- 内部错误应保留上下文。
- 用户可见错误必须脱敏。
- 外部系统错误不得原样暴露给前端。
- 错误码应稳定，不应把完整错误文本当成协议字段。

### 6.5 context 规则

请求级调用链必须传递 `context.Context`。

规则：

- context 应作为函数的第一个参数。
- 外部调用必须尊重 context cancellation。
- goroutine 必须有退出条件。
- 不得把 context 存到结构体长期保存。
- 不得用 context 传递可选业务参数。

### 6.6 HTTP Client 规则

访问外部服务必须使用带 timeout 的 `http.Client`。

禁止：

```go
http.DefaultClient.Do(req)
```

除非当前代码明确只用于短生命周期测试。

推荐：

```go
client := &http.Client{Timeout: 30 * time.Second}
```

规则：

- LLM、Agent、Webhook、外部 API 调用都必须设置 timeout。
- 可复用 client 应在启动时创建并注入。
- 不要在每个请求中重复创建复杂 client。
- Transport 配置应集中，不要分散在 handler 内。

### 6.7 Handler 规则

HTTP handler 应保持薄层。

Handler 可以：

- 解析请求。
- 校验基础参数。
- 调用 service / orchestrator / store。
- 写 response。

Handler 不应：

- 写复杂业务编排。
- 直接访问多个外部系统并拼装流程。
- 直接处理跨模块状态机。
- 直接保存请求级全局变量。
- 直接拼接复杂协议事件。

### 6.8 并发规则

- goroutine 必须可退出。
- channel 必须有明确关闭方。
- 不要在多个 goroutine 中无保护写 map。
- 使用 `sync.Mutex` 时要缩小锁范围。
- 使用 `context.Context` 控制取消。
- 并发逻辑必须测试错误路径和取消路径。

---

## 7. TypeScript 风格

### 7.1 基础规则

必须优先使用 TypeScript 类型系统表达边界。

规则：

- 开启 `strict`。
- 避免无理由 `any`。
- 避免长期 `@ts-ignore`。
- 公共类型必须集中定义或生成。
- 外部 JSON 必须有安全解析或 fallback。
- 不要在 UI 组件内手写复杂后端返回结构。

允许使用 `unknown` 表达未验证外部输入。

示例：

```ts
function parsePayload(input: unknown): Payload | null {
  if (!input || typeof input !== 'object') return null
  // validate fields
  return input as Payload
}
```

### 7.2 类型命名

- type / interface 使用 PascalCase。
- 变量和函数使用 camelCase。
- 常量枚举值可使用 UPPER_SNAKE_CASE 或稳定字符串字面量。
- Props 类型使用 `XxxProps`。
- Store 状态类型使用 `XxxState`。
- API 返回类型使用 `XxxResponse`。
- API 请求类型使用 `XxxRequest`。

### 7.3 any 与类型断言

禁止：

```ts
const data: any = await res.json()
```

更推荐：

```ts
const data: unknown = await res.json()
```

类型断言规则：

- 类型断言必须靠近外部边界。
- 不要在业务逻辑深处反复 `as Something`。
- 如果无法验证，必须有 fallback 或错误处理。

### 7.4 API 调用规则

- API 调用应集中在 service / client 层。
- 组件不直接拼接复杂 URL。
- 组件不直接处理认证 header。
- 错误 response 必须转换为用户可理解错误。
- SSE / streaming client 必须处理粘包、拆包、取消和异常。

### 7.5 异步规则

- async 函数必须处理 reject。
- UI 发起请求必须能取消或忽略过期响应。
- 不要在 `useEffect` 中直接创建无法清理的异步副作用。
- 长流式请求必须支持 AbortController。

---

## 8. React 风格

### 8.1 组件命名

- React 组件使用 PascalCase。
- 文件名与主组件名一致，例如 `MessageBubble.tsx`。
- Hook 使用 `useXxx`。
- Context 使用 `XxxContext`。
- Provider 使用 `XxxProvider`。

### 8.2 组件边界

页面组件负责组合。

通用 UI 组件负责展示。

Store / service 负责状态和数据。

禁止：

- 在通用 UI 组件里直接请求 API。
- 在页面组件里写复杂协议解析。
- 在组件中混合大量数据转换、网络请求和展示逻辑。
- 把所有状态都塞进单个全局 store。

### 8.3 Props 规则

- Props 必须显式定义类型。
- 布尔 props 命名优先使用 `is`、`has`、`can`、`should`。
- 不要把大对象作为 props 层层透传，除非它确实是领域对象。
- 回调 props 使用 `onXxx`。

### 8.4 Hook 规则

- 自定义 Hook 必须以 `use` 开头。
- Hook 只在组件顶层或其他 Hook 中调用。
- Hook 不应隐藏高风险副作用。
- Hook 返回值应稳定、清晰。
- 复杂 Hook 应有单测或最小交互测试。

### 8.5 高风险渲染

以下能力必须封装在独立组件中：

- HTML 预览。
- iframe。
- Markdown 渲染。
- 代码高亮。
- 下载链接。
- 外链打开。

规则：

- 不要在普通文本组件里直接插入 HTML。
- 不要把外部内容无校验地注入 DOM。
- iframe 必须有明确 sandbox 策略。
- 用户可见错误必须友好，不应导致白屏。

---

## 9. JSON / YAML / Markdown 风格

### 9.1 JSON

- 对外 API 字段使用 camelCase。
- JSON Schema 文件使用 `*.schema.json`。
- JSON 示例必须是合法 JSON。
- 不要在 JSON 示例中写注释。
- 不要混用 snake_case 与 camelCase，除非明确是数据库或外部协议兼容层。

### 9.2 YAML

- YAML 用于配置时必须保持字段稳定。
- 不要把 secret 写入 YAML 示例。
- 示例 secret 使用占位符，例如 `${API_KEY}`。
- 布尔值使用 `true` / `false`。
- 列表缩进保持一致。

### 9.3 Markdown

- 标题层级必须连续。
- 一个文档只使用一个 H1。
- 代码块必须标注语言。
- Contract 文档必须包含目的、边界、字段、示例、禁止事项、Review Checklist。
- 历史 Profile 不能写成当前开发禁令。
- 中文文档使用中文标点，英文专有名词保持原文。

---

## 10. 文件与目录命名

### 10.1 通用目录

- 项目文档目录使用 kebab-case。
- Skill 目录使用 kebab-case。
- Contract 文档使用 kebab-case。
- 临时输出包使用清晰版本名和语言标记。

### 10.2 Go 目录

- Go package 目录使用小写短名。
- 不要使用难懂缩写。
- 不要创建万能 `common` / `utils` / `shared` 目录。
- `internal` 下目录应表达业务边界。

### 10.3 前端目录

- 组件文件使用 PascalCase.tsx。
- Hook 文件使用 useXxx.ts。
- Store 文件使用 xxxStore.ts。
- Service / client 文件使用 camelCase 或 kebab-case，但同目录必须一致。
- 类型文件可以使用 `types.ts` 或领域名，例如 `conversation.ts`。

### 10.4 测试命名

- Go 测试使用 `_test.go`。
- TypeScript 测试使用 `.test.ts` / `.test.tsx`。
- E2E 测试使用 `.spec.ts`。
- 测试名称应表达行为，不只写函数名。

---

## 11. 错误处理风格

### 11.1 后端错误

后端错误分三层：

1. 内部错误：包含详细上下文，用于日志。
2. 协议错误：稳定错误码，用于前后端交互。
3. 用户错误：脱敏、可理解，用于 UI 展示。

规则：

- 内部错误可以 wrap。
- 用户错误不能包含密钥、堆栈、连接串、内部路径。
- API 错误必须有稳定 code。
- 可重试错误应标记 retryable。
- 不要把第三方服务原始错误直接暴露给前端。

### 11.2 前端错误

前端错误必须：

- 停止 loading。
- 给出可理解提示。
- 不导致白屏。
- 保留开发调试信息，但不暴露给普通用户。
- 流式请求失败时应清理 abort controller 或 streaming 状态。

---

## 12. 日志与注释风格

### 12.1 日志

日志必须有上下文。

推荐字段：

```text
requestId
runId
conversationId
messageId
agentName
operation
errorCode
durationMs
```

禁止记录：

- API key。
- Authorization token。
- Cookie。
- 数据库密码。
- 完整 system prompt。
- 用户私密内容的全文。

### 12.2 注释

注释解释“为什么”，不是重复“是什么”。

应该注释：

- 兼容逻辑。
- 安全边界。
- 不直观的算法。
- 临时折中及移除条件。
- 外部协议兼容原因。

不应该注释：

```go
// increment i
i++
```

### 12.3 TODO

TODO 必须包含原因或后续动作。

推荐：

```text
TODO(v1.1): 将临时 inline 存储迁移到对象存储。
```

禁止：

```text
TODO: fix later
```

---

## 13. 测试风格

### 13.1 测试原则

测试应该覆盖：

- happy path。
- 错误路径。
- 空输入。
- 非法输入。
- 取消 / 超时。
- 兼容字段。
- 多实体场景。

### 13.2 Go 测试

- 使用 table-driven tests 测多 case。
- 测试名使用 `TestXxx`。
- 子测试使用 `t.Run`。
- 外部依赖使用 fake / mock / test server。
- 不依赖真实 LLM 或真实外部网络。

### 13.3 TypeScript / React 测试

- 纯函数优先单测。
- Store reducer / event reducer 必须测试关键事件。
- 组件测试关注用户可见行为。
- 不过度测试实现细节。
- 异步 UI 必须等待状态变化。

### 13.4 测试数据

- fixture 必须小而清晰。
- 不要在测试中使用真实 secret。
- 大型 fixture 应放在明确目录。
- 测试名称应说明 fixture 的意图。

---

## 14. 质量门禁

优先执行仓库已有命令。

如果命令不存在，不得声称已执行或已通过。

### 14.1 后端建议命令

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

如果项目配置了额外工具，可以执行：

```bash
golangci-lint run
```

### 14.2 前端建议命令

```bash
npm run lint
npm run typecheck
npm run test
npm run build
```

如果项目使用其他包管理器，应按实际 lockfile 和脚本执行。

### 14.3 项目级建议命令

```bash
docker compose config
./smoke-test.sh
```

### 14.4 文档类任务检查

如果本轮只修改文档或 Skill，至少检查：

- 路径是否正确。
- 文件是否为中文。
- frontmatter 是否合法。
- Markdown 标题层级是否合理。
- 链接或引用路径是否存在。
- 是否没有混入未授权业务代码。

---

## 15. Commit Message 规范

提交信息使用 Conventional Commits 风格。

格式：

```text
<type>(<scope>): <summary>
```

常用 type：

```text
feat      新功能
fix       修复问题
docs      文档变更
refactor  重构，不改变行为
test      测试
chore     工程配置或杂项
style     格式、命名，不改变行为
ci        CI/CD
build     构建系统或依赖
```

示例：

```text
feat(agent): add generic child agent registry
fix(frontend): handle split SSE event blocks
docs(skills): rewrite code style convention in Chinese
refactor(server): extract planner error handling
```

禁止：

```text
update
fix bug
wip
final
随便改一下
```

AI 生成提交说明必须准确描述实际变更，不得夸大完成度。

---

## 16. AI 生成代码规则

AI 生成代码必须：

- 遵守当前目录已有风格。
- 只修改用户授权范围内文件。
- 不引入无关依赖。
- 不大规模重排无关文件。
- 不把示例代码伪装成生产实现。
- 不留下不可运行伪代码。
- 不用 TODO 代替必要实现，除非用户明确要求草稿。
- 修改公共类型时同步更新调用方。
- 生成 mock 时标注 mock 范围。
- 不声称测试通过，除非实际执行或用户提供结果。
- 输出文件包时必须包含清单和变更说明。

AI 生成代码不得：

- 偷偷改变业务范围。
- 偷偷升级框架版本。
- 偷偷安装依赖。
- 删除用户代码而不说明。
- 将 secret 写入示例。
- 把其他任务的重构夹带进当前任务。

---

## 17. 禁止事项

以下行为禁止：

- 把历史 MVP 限制当成当前开发禁令。
- 在风格 Skill 中展开专业业务协议。
- 为了统一风格而重写大量无关文件。
- 使用 `any` 规避类型错误。
- 使用 `panic` 规避错误处理。
- 使用全局变量保存请求状态。
- 使用无 timeout 的外部 HTTP 调用。
- 在日志中打印 secret。
- 在 Markdown 中放不合法 JSON 示例。
- 在 Review 中只检查 happy path。
- 未执行测试却声称测试通过。

---

## 18. Review Checklist

### 通用

- 是否遵守当前文件夹已有风格？
- 是否没有引入无关依赖？
- 是否没有大规模无关格式化？
- 是否没有越出用户授权范围？
- 是否没有把历史阶段写成当前禁令？

### Go

- 是否已 gofmt？
- 包名是否小写简短？
- error 是否显式处理？
- context 是否向下传递？
- HTTP client 是否有 timeout？
- handler 是否保持薄层？
- goroutine 是否可退出？
- 外部依赖是否可测试？

### TypeScript

- 是否符合 strict？
- 是否没有无理由 any？
- 是否没有长期 @ts-ignore？
- 外部 JSON 是否有安全处理？
- API 调用是否集中？
- 异步请求是否能取消或安全忽略过期响应？

### React

- 组件是否 PascalCase？
- Hook 是否 useXxx？
- Props 是否显式类型？
- 页面是否只负责组合？
- 高风险渲染是否封装？
- 错误是否不会导致白屏？

### 命名

- 对外 JSON 字段是否 camelCase？
- 数据库字段是否没有直接泄漏到 API？
- 目录是否 kebab-case 或符合语言习惯？
- Contract 文件是否 kebab-case？
- 测试文件命名是否正确？

### 测试

- 是否补充必要测试？
- 是否覆盖错误路径？
- 是否没有只测 happy path？
- 是否说明未执行测试原因？

### 文档

- 是否中文一致？
- 是否标题层级合理？
- 是否代码块标注语言？
- 是否没有未授权引用其他 Skill 的详细规则？

### 提交

- commit message 是否符合 Conventional Commits？
- scope 是否准确？
- summary 是否准确描述实际变更？

---

## 19. 完成定义

本 Skill 视为完成，当且仅当：

- `SKILL.md` 使用标准多行 YAML frontmatter。
- 文件内容为中文。
- 不写死当前业务阶段。
- 不固定具体 Agent 名称。
- 不展开其他专业协议。
- Go、TypeScript、React、JSON、YAML、Markdown 规则明确。
- 命名、错误、日志、测试、质量门禁规则明确。
- AI 生成代码规则明确。
- Review Checklist 明确。
- references 与 docs/contracts 内容同步。

---

## References

- `references/go-style.md`
- `references/typescript-react-style.md`
- `references/naming-conventions.md`
- `references/json-yaml-markdown-style.md`
- `references/error-handling-style.md`
- `references/logging-and-comments.md`
- `references/test-style.md`
- `references/quality-gates.md`
- `references/commit-message-policy.md`
- `references/review-checklist.md`
