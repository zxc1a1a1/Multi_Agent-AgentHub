---
name: artifact-contract
description: "用于定义 AgentHub 中所有 Agent 生成产物的独立通用 Artifact 契约，包括 Artifact 类型注册、标准字段、content/contentRef、生命周期、版本、身份关联、预览映射、安全边界、schema 校验和 Review 规则。"
---

# artifact-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 中所有生成产物的标准契约。

Artifact 是 AgentHub 中由 Agent、编排流程或系统能力生成的**产物事实源**。它可以是代码、网页、Markdown、文档、结构化数据、文件引用或其他可被后续查看、预览、下载、复用、版本化的结果。

本 Skill 的核心定位：

```text
Agent / 系统输出 → AgentHub normalized Artifact → identity / content / storage / versioning → previewType / download / reuse
```

一句话：

```text
artifact-contract 只定义 AgentHub 产物事实源，不定义前端组件、不定义实时事件、不定义子 Agent 内部实现。
```

---

## 2. 独立性原则

本 Skill 必须独立可读。

阅读本 Skill 时，不应要求读者先阅读其他 Skill。

本 Skill 可以说明边界，但不得展开其他协议或实现细节。

本 Skill 不负责：

- 子 Agent 如何生成 Artifact。
- 实时事件如何把 Artifact 通知到前端。
- 前端组件如何渲染 Artifact。
- 数据库表如何建。
- 对象存储 SDK 如何接入。
- Docker、CI、E2E 测试如何配置。
- LLM prompt 如何诱导模型生成某类产物。

如果某个实现最终会产生 Artifact，只要进入 AgentHub，就必须归一化为本文定义的 Artifact 对象。

---

## 3. 当前阶段

当前项目已完成 MVP v0.1，正在进入 v1.0 迭代。

MVP v0.1 的 `code Artifact → code_preview → 小型 inline content` 是历史基线，不再作为所有新产物的唯一模型。

v1.0 产物契约目标：

- 支持 2+ Agent 或系统模块生成 Artifact。
- 不固定具体 Agent 名称。
- 支持多类型 Artifact。
- 支持 `content` 与 `contentRef` 的清晰边界。
- 支持 `artifact.type → previewType` 的稳定映射。
- 支持多 Agent / 群聊下的产物归属。
- 支持 schema 校验、版本化、生命周期和安全脱敏。

---

## 4. Artifact 定义

Artifact 是具备以下特征的产物对象：

- 有稳定身份：`artifactId`。
- 有类型：`type`。
- 有标题或文件名：`title`。
- 有 MIME 类型：`mimeType`。
- 有内容或内容引用：`content` / `contentRef`。
- 有来源信息：`source`。
- 有上下文关联：`links`。
- 有状态：`status`。
- 可以被预览、下载、复用、版本化或持久化。

普通文本消息不是 Artifact。

当输出只用于对话阅读，且不需要独立预览、下载、版本化或引用时，应保留为普通 message content。

当输出具备文件、代码、页面、文档、数据、图片、附件、可复用内容等属性时，应建模为 Artifact。

---

## 5. Artifact 与普通消息的区别

| 对比项 | 普通消息 | Artifact |
|---|---|---|
| 主要用途 | 对话阅读 | 产物保存、预览、复用 |
| 身份 | messageId | artifactId + version |
| 内容大小 | 通常较小 | 可小可大 |
| 生命周期 | 跟随消息 | 可独立管理 |
| 版本化 | 通常没有 | 必须支持 |
| 预览 | 文本渲染 | 由 previewType 决定 |
| 下载 | 通常不支持 | 可支持 |
| 引用 | 弱引用 | 可稳定引用 |

规则：

- 不要把大型产物塞进普通消息文本。
- 不要把需要复用的代码、网页、文档只放在 message content 中。
- 不要把 previewType 当作 Artifact 类型。
- 不要把前端组件名当作 Artifact 类型。

---

## 6. Artifact Type Registry

Artifact 类型必须统一注册。

任何新增类型必须先进入 Artifact Type Registry，再允许生成、持久化、预览或下载。

v1.0 推荐注册类型：

| artifact.type | 用途 | 默认 mimeType | 默认 previewType | v1.0 状态 |
|---|---|---|---|---|
| `code` | 代码片段或单文件代码 | `text/plain` | `code_preview` | required |
| `webpage` | HTML 页面或网页片段 | `text/html` | `web_preview` | required |
| `markdown` | Markdown 文档、说明、报告 | `text/markdown` | `markdown_render` | required |
| `document` | 较大文档或非 Markdown 文档 | `application/octet-stream` | `document_preview` | planned |
| `data` | JSON / CSV / 结构化数据 | `application/json` | `data_preview` | planned |
| `image` | 图片产物 | `image/png` | `image_preview` | planned |
| `archive` | zip/tar 等打包文件 | `application/zip` | `download` | planned |

规则：

- `artifact.type` 必须是稳定枚举值。
- `artifact.type` 使用小写 snake_case 或单词形式。
- `artifact.type` 不得等于 React 组件名。
- `artifact.type` 不得等于事件名。
- `artifact.type` 不得等于 toolName，除非项目显式声明二者同名只是兼容别名。
- 新类型必须说明 MIME、inline 策略、previewType、存储策略和安全风险。

---

## 7. 标准 Artifact Schema

标准 Artifact 对象应包含以下字段：

```json
{
  "artifactId": "art_01H00000000000000000000000",
  "type": "webpage",
  "title": "landing-page.html",
  "mimeType": "text/html",
  "content": "<!DOCTYPE html><html>...</html>",
  "contentRef": null,
  "summary": "响应式落地页",
  "metadata": {
    "language": "html",
    "entry": "index.html"
  },
  "source": {
    "agentName": "agent-name",
    "taskId": "task_001"
  },
  "links": {
    "conversationId": "conv_001",
    "messageId": "msg_001",
    "runId": "run_001"
  },
  "preview": {
    "previewType": "web_preview",
    "available": true
  },
  "version": 1,
  "status": "ready",
  "createdAt": "2026-05-25T00:00:00Z",
  "updatedAt": "2026-05-25T00:00:00Z"
}
```

字段分组：

| 分组 | 字段 |
|---|---|
| 身份 | `artifactId`, `version` |
| 类型 | `type`, `title`, `mimeType` |
| 内容 | `content`, `contentRef`, `summary` |
| 元数据 | `metadata` |
| 来源 | `source.agentName`, `source.taskId` |
| 关联 | `links.conversationId`, `links.messageId`, `links.runId` |
| 预览 | `preview.previewType`, `preview.available` |
| 状态 | `status`, `createdAt`, `updatedAt` |

必填字段：

```text
artifactId
type
title
mimeType
links.conversationId
links.messageId
links.runId
version
status
createdAt
```

`content` 与 `contentRef` 至少应存在一个。

---

## 8. content / contentRef 规则

`content` 表示小型 inline 内容。

`contentRef` 表示由后端控制的内容引用。

推荐策略：

| 场景 | 使用方式 |
|---|---|
| 小型代码 | `content` |
| 小型 Markdown | `content` |
| 小型单页 HTML | `content` |
| 小型 JSON | `content` |
| 大型代码包 | `contentRef` |
| zip / tar | `contentRef` |
| 图片 / 二进制 | `contentRef` |
| 大型日志 | `contentRef` |
| 私有文件 | `contentRef` |

禁止：

- 把大型文件直接塞进 `content`。
- 把 zip、图片、二进制文件 base64 后无条件 inline。
- 把私有对象存储签名 URL 当成永久 `contentRef`。
- 把本地绝对路径写进 `contentRef`。
- 把内网服务地址写进 `contentRef`。
- 把含 token 的下载链接写进 Artifact。

`contentRef` 必须是后端可验证、可授权、可审计的引用。

示例：

```json
{
  "contentRef": {
    "kind": "object_store",
    "bucket": "agenthub-artifacts",
    "key": "conv_001/art_001/index.html",
    "sizeBytes": 42000,
    "sha256": "..."
  }
}
```

`contentRef` 不应该是裸外链字符串。

---

## 9. metadata 规则

`metadata` 用于保存类型相关但非核心字段。

规则：

- `metadata` 必须是 JSON object。
- `metadata` 不得保存密钥、token、私有 URL、系统 prompt。
- `metadata` 不得保存无法序列化的对象。
- `metadata` 字段应稳定命名。
- 类型相关字段应尽量文档化。

常见字段：

| 字段 | 适用类型 | 说明 |
|---|---|---|
| `language` | code / webpage / markdown | 语言或格式 |
| `filename` | code / document / archive | 原始文件名 |
| `entry` | webpage / archive | 入口文件 |
| `framework` | code / webpage | 框架信息 |
| `sizeBytes` | all | 内容大小 |
| `sha256` | all | 内容摘要 |
| `encoding` | text-like | 编码 |

---

## 10. Preview Mapping

Artifact Contract 只定义：

```text
artifact.type → previewType
```

它不定义前端组件实现，也不定义事件传输字段。

v1.0 推荐映射：

| artifact.type | previewType |
|---|---|
| `code` | `code_preview` |
| `webpage` | `web_preview` |
| `markdown` | `markdown_render` |
| `document` | `document_preview` |
| `data` | `data_preview` |
| `image` | `image_preview` |
| `archive` | `download` |

规则：

- `previewType` 是预览意图，不是 Artifact 类型。
- `previewType` 不等于 React 组件名。
- 一个 `artifact.type` 可以有默认 `previewType`。
- 一个 Artifact 可以因为内容过大、安全策略或权限问题而 `preview.available = false`。
- 未注册的 `artifact.type` 不得预览。
- 未知 `previewType` 应安全降级为下载或纯文本摘要。

---

## 11. 生命周期

Artifact 生命周期状态：

```text
pending
normalizing
ready
failed
superseded
deleted
```

推荐状态机：

```text
pending → normalizing → ready
pending → normalizing → failed
ready → superseded
ready → deleted
```

规则：

- 只有 `ready` 状态可以预览。
- `failed` 状态不得进入预览映射。
- `deleted` 状态应保留最小 tombstone 信息，避免历史消息断链。
- 重新生成不应覆盖旧 Artifact，应创建新版本或新 artifactId。
- 生命周期变更必须可审计。

---

## 12. 版本化规则

Artifact 必须支持版本化。

最小规则：

- `version` 从 1 开始。
- 同一 `artifactId` 的新内容必须增加 `version`。
- 如果无法稳定管理版本，应创建新的 `artifactId`。
- 新版本不得破坏历史消息里的旧引用。
- `superseded` 表示旧版本被新版本替代，但不是删除。

推荐字段：

```json
{
  "artifactId": "art_001",
  "version": 2,
  "parentArtifactId": "art_001",
  "supersedes": {
    "artifactId": "art_001",
    "version": 1
  }
}
```

---

## 13. 身份与关联字段

多 Agent、群聊和多产物场景下，Artifact 必须可追溯。

必需关联：

```text
artifactId
conversationId
messageId
runId
```

推荐关联：

```text
source.agentName
source.taskId
source.stepId
parentArtifactId
groupId
```

规则：

- 同一个 Run 可以生成多个 Artifact。
- 同一个 message 可以关联多个 Artifact。
- 不同 Agent 生成同名文件时不得冲突。
- Artifact 必须能追溯到生成它的 Agent 或系统模块。
- Artifact ID 不得由 `title` 或 `filename` 单独派生。
- 用户可见名称使用 `title`，系统引用使用 `artifactId`。

---

## 14. 存储策略

v1.0 可以采用渐进式存储：

| 内容类型 | 建议策略 |
|---|---|
| 小型文本 | 数据库 JSON / message artifacts |
| 小型代码 | 数据库 JSON / artifact table |
| 小型 Markdown | 数据库 JSON / artifact table |
| 小型单页 HTML | 数据库 JSON / artifact table |
| 大型文件 | contentRef |
| 二进制 | contentRef |
| 打包文件 | contentRef |

规则：

- inline 只是预览优化，不是长期存储模型。
- 大型 Artifact 不应进入实时 token 流。
- 大型 Artifact 不应进入普通 message content。
- 私有文件必须通过后端授权访问。
- Redis、内存缓存、临时 map 不得作为唯一事实源。

---

## 15. 安全与脱敏

Artifact 不得包含：

- API key。
- Authorization token。
- Cookie。
- 数据库连接串。
- 私有对象存储签名 URL。
- 本地绝对路径。
- 内网服务拓扑。
- 未脱敏用户隐私。
- 完整 system prompt。
- 模型供应商原始敏感错误。

对 `webpage` / HTML 类 Artifact：

- 保存 HTML 不代表可信执行。
- HTML 只能作为待预览内容。
- 是否允许脚本执行由运行环境决定。
- 本文件不定义 iframe、sandbox 或浏览器权限细节。

对 `contentRef`：

- 必须可授权。
- 必须可审计。
- 不得是永久公开私有 URL。
- 不得携带临时 token。

---

## 16. MVP v0.1 Historical Profile

MVP v0.1 已完成，仅作为回归测试基线。

MVP 历史能力：

- `code` Artifact。
- small code inline。
- `metadata.language`。
- `title` 作为 filename。
- 临时 `message.artifacts` JSON 保存。
- completed 后映射到 `code_preview`。

MVP 例外不得反向污染 v1.0 通用契约。

尤其不得把以下规则扩展为通用规则：

- 只有 `code` 才是 Artifact。
- 所有 Artifact 都可以 inline。
- `title` 可以替代 `artifactId`。
- preview 参数可以替代 Artifact schema。

---

## 17. v1.0 Required Artifact Types

v1.0 当前 required 类型：

```text
code
webpage
markdown
```

v1.0 当前 required 能力：

- 标准 Artifact schema。
- Artifact Type Registry。
- `content` / `contentRef` 边界。
- previewType 映射。
- 多 Agent 归属字段。
- 生命周期状态。
- schema 校验。
- 安全脱敏。

v1.0 不固定具体 Agent 名称。

任何 Agent 或系统模块只要生成符合本文约束的 Artifact，都可以接入。

---

## 18. Post-v1.0 Planned Types

Post-v1.0 可扩展：

```text
document
data
image
audio
video
archive
diff
terminal_log
deploy_bundle
```

新增类型前必须补充：

- type 名称。
- mimeType。
- content / contentRef 策略。
- previewType。
- 安全风险。
- 大小限制。
- 版本化策略。
- schema 示例。
- Review checklist。

---

## 19. Contract-first 规则

任何新增或修改以下内容前，必须先更新项目级契约：

```text
docs/contracts/artifact-schema.md
docs/contracts/artifact.schema.json
docs/contracts/artifact-type-registry.md
```

必须先更新契约的变更包括：

- 新增 Artifact 类型。
- 修改标准字段。
- 修改 `content` / `contentRef` 规则。
- 修改 previewType 映射。
- 修改生命周期状态。
- 修改版本化规则。
- 修改安全规则。

不得先改实现，再事后补 Artifact contract。

---

## 20. 禁止事项

Coding Agent 不得：

- 未注册 Artifact type 就生成或持久化。
- 把 previewType 当成 artifact.type。
- 把 React 组件名写入 Artifact type。
- 把事件名写入 Artifact type。
- 把 toolName 当成 Artifact 事实源。
- 用 agentName 判断 Artifact 类型。
- 让 Artifact 缺少 `artifactId`。
- 让 Artifact 缺少 `conversationId / messageId / runId`。
- 把大型文件塞进 `content`。
- 把私有对象存储签名 URL 写进 Artifact。
- 把本地路径或内网地址写进 Artifact。
- 重新生成时覆盖旧版本。
- 把 MVP inline 规则扩展到所有类型。
- 在本 Skill 中定义前端组件实现。
- 在本 Skill 中定义实时事件字段。

---

## 21. Review Checklist

审查 Artifact 相关变更时，必须检查：

### Schema

- 是否更新 `artifact.schema.json`？
- 是否使用 JSON Schema 2020-12？
- 是否定义 `artifactId / type / title / mimeType / status / version`？
- 是否定义 `content` / `contentRef` 边界？
- 是否保留 `metadata` 扩展能力？

### Type

- `artifact.type` 是否已注册？
- `previewType` 是否已注册？
- 是否没有把组件名当类型？
- 是否没有把 toolName 当事实源？

### Content

- 小内容是否允许 inline？
- 大内容是否使用 `contentRef`？
- 是否没有 inline 二进制、zip、大日志、大 HTML 包？
- 是否没有裸露私有 URL？

### Identity

- 是否有 `artifactId`？
- 是否有关联 `conversationId / messageId / runId`？
- 多 Agent 场景是否有来源字段？
- 重名文件是否不会冲突？

### Lifecycle

- 是否有 `status`？
- 是否只有 `ready` 可预览？
- 重新生成是否新版本？
- 删除是否不破坏历史消息？

### Security

- 是否没有 secret？
- 是否没有本地路径？
- 是否没有内网地址？
- 是否没有 system prompt？
- contentRef 是否可授权、可审计？

---

## 22. 完成定义

本 Skill 视为完成，当且仅当：

```text
.claude/skills/artifact-contract/SKILL.md
```

已经明确：

- Artifact 是 AgentHub 的产物事实源。
- 本 Skill 独立可读。
- 不固定具体 Agent 名称。
- MVP v0.1 是历史基线。
- v1.0 支持多类型 Artifact。
- `code / webpage / markdown` 为 v1.0 required 类型。
- Artifact Type Registry 明确。
- 标准 Artifact schema 明确。
- `content` / `contentRef` 边界明确。
- previewType 映射明确。
- 生命周期明确。
- 版本化规则明确。
- 多 Agent / 群聊的身份关联明确。
- 安全脱敏规则明确。
- Review Checklist 明确。

---

## References

- `references/artifact-type-registry.md`
- `references/artifact-schema-policy.md`
- `references/content-and-reference-policy.md`
- `references/lifecycle-policy.md`
- `references/preview-mapping-policy.md`
- `references/identity-linking-policy.md`
- `references/versioning-policy.md`
- `references/storage-policy.md`
- `references/artifact-security-policy.md`
- `references/artifact-review-checklist.md`
