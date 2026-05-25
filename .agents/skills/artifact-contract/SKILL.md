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

## 6. ArtifactDraft 与 Core Artifact 的归一化边界

### 6.1 两个概念

AgentHub 中有两种 Artifact 形态，必须严格区分：

| 形态 | 来源 | 包含字段 |
|---|---|---|
| **ArtifactDraft** | Child Agent / ADK Handler 输出 | `type`、`title`、`content`（或 `contentRefDraft`）、`metadata` |
| **Core Artifact** | Orchestrator / ArtifactRegistry 归一化后 | 完整标准字段（`artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 等） |

ArtifactDraft 不是标准 Core Artifact。Core Artifact 只能由 ArtifactRegistry 归一化生成。

### 6.2 归一化流程

```text
Child Agent / ADK Handler
    → ctx.AddArtifact(ArtifactDraft)
    → A2A artifact event（ArtifactDraft）
    → Orchestrator 接收
    → ArtifactRegistry.Normalize(ArtifactDraft, context)
    → Core Artifact（完整字段）
```

### 6.3 ArtifactDraft 最小字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| `type` | 是 | Artifact 类型枚举值 |
| `title` | 是 | 产物标题或文件名 |
| `content` | 二选一 | 小型 inline 内容 |
| `contentRefDraft` | 二选一 | 大型内容的内部引用（非 Core contentRef 对象） |
| `metadata` | 推荐 | 类型相关元数据，推荐包含 `language`、`mimeType` 提示 |

规则：

- Child Agent / ADK Handler 不得在 ArtifactDraft 中提供 `artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt`。
- `contentRefDraft` 是 Child Agent 内部引用，不是 Core Artifact 的 `contentRef` 对象。
- Orchestrator 负责将 `contentRefDraft` 转换为平台可授权的 `contentRef`。

### 6.4 归一化时平台生成或补齐的字段

以下字段由 Orchestrator / ArtifactRegistry 在归一化时生成：

| 字段 | 生成者 | 生成方式 |
|---|---|---|
| `artifactId` | ArtifactRegistry | 分配全局唯一 ID |
| `mimeType` | ArtifactRegistry | 根据 `type` + metadata 推断或取默认值 |
| `source.agentName` | Orchestrator | 已知调用目标 Agent |
| `source.taskId` | Orchestrator | 已知 A2A task id |
| `links.conversationId` | Orchestrator | 已知当前会话 |
| `links.messageId` | Orchestrator / Gateway | 消息创建后回填 |
| `links.runId` | Orchestrator | 已知当前 Run |
| `preview.previewType` | ArtifactRegistry | 根据 `type` 查 Artifact Type Registry |
| `preview.available` | ArtifactRegistry | 根据内容大小、安全策略决定 |
| `version` | ArtifactRegistry | 默认 1，重复归一化时递增 |
| `status` | ArtifactRegistry | 初始 `pending`，归一化完成后 `ready` |
| `createdAt` | ArtifactRegistry | 归一化时间 |
| `updatedAt` | ArtifactRegistry | 归一化/更新时间 |

### 6.5 归一化示例

ArtifactDraft 输入：

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main\n\nfunc main() {}",
  "metadata": {
    "language": "go"
  }
}
```

归一化后 Core Artifact（新增字段标注 ★）：

```json
{
  "artifactId": "art_01H00000000000000000000000",   // ★ 生成
  "type": "code",
  "title": "main.go",
  "mimeType": "text/x-go",                          // ★ 推断
  "content": "package main\n\nfunc main() {}",
  "contentRef": null,
  "summary": null,
  "metadata": {"language": "go"},
  "source": {                                       // ★ 补齐
    "agentName": "code-agent",
    "taskId": "task_001"
  },
  "links": {                                        // ★ 补齐
    "conversationId": "conv_001",
    "messageId": "msg_001",
    "runId": "run_001"
  },
  "preview": {                                      // ★ 补齐
    "previewType": "code_preview",
    "available": true
  },
  "version": 1,                                     // ★ 生成
  "status": "ready",                                // ★ 生成
  "createdAt": "2026-05-25T00:00:00Z",             // ★ 生成
  "updatedAt": "2026-05-25T00:00:00Z"              // ★ 生成
}
```

### 6.6 禁止事项

- 不得把 ArtifactDraft 当作 Core Artifact。
- Child Agent / ADK Handler 不得在 Draft 中提供平台字段。
- 不得跳过归一化直接使用 ArtifactDraft 作为公开 API 响应。
- Public API 只能暴露 Core Artifact 的 DTO 投影，不得暴露 ArtifactDraft。

---

## 7. Artifact Type Registry

Artifact 类型必须统一注册。

`artifact.type` 是平台归一化后的产物类型。ArtifactRegistry 在归一化时根据 ArtifactDraft 中的 `type` 字段、`metadata`（如 `language`、`format`）、推断的 `mimeType`、以及 Agent 声明的 `outputMode` 等信息综合确定最终的 `artifact.type`。

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
| `archive` | zip/tar 等打包文件 | `application/zip` | `file_download` | planned |

规则：

- `artifact.type` 必须是稳定枚举值。
- `artifact.type` 使用小写 snake_case 或单词形式。
- `artifact.type` 不得等于 React 组件名。
- `artifact.type` 不得等于事件名。
- `artifact.type` 不得等于 toolName，除非项目显式声明二者同名只是兼容别名。
- `artifact.type` 不得直接等于 `outputMode`，除非映射表显式声明兼容。
- 新类型必须说明 MIME、inline 策略、previewType、存储策略和安全风险。

---

## 8. 标准 Artifact Schema

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

## 9. content / contentRef 规则

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

## 10. metadata 规则

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

## 11. Preview Mapping

Artifact Contract 定义三层映射：

```text
outputMode → artifact.type → previewType → Frontend Runtime toolName
```

其中：
- `outputMode`（上游）是 Agent 声明的语义类别，不属于 Artifact Contract 定义范围，但 artifact.type 的确定会参考它。
- `artifact.type` 是平台归一化后的产物类型。
- `previewType` 应优先等于 Frontend Runtime Registry 中的 `toolName`。

它不定义前端组件实现，也不定义事件传输字段。

v1.0 推荐映射：

| artifact.type | previewType | toolName | v1.0 状态 |
|---|---|---|---|
| `code` | `code_preview` | `code_preview` | required |
| `webpage` | `web_preview` | `web_preview` | required |
| `markdown` | `markdown_render` | `markdown_render` | required |
| `document` | `document_preview` | `document_preview` | planned |
| `data` | `data_preview` | `data_preview` | planned |
| `image` | `image_preview` | `image_preview` | planned |
| `archive` | `file_download` | `file_download` | planned |

完整三层映射链（含上游 outputMode）：

| outputMode | artifact.type | previewType | toolName | 说明 |
|---|---|---|---|---|
| `code` | `code` | `code_preview` | `code_preview` | 代码类产物 |
| `webpage` | `webpage` | `web_preview` | `web_preview` | 网页类产物 |
| `document` | `document` 或 `markdown` | `document_preview` 或 `markdown_render` | `document_preview` 或 `markdown_render` | 归一化时根据 metadata.format 确定 |
| `text` | 无 Artifact | — | `markdown_render` / StreamingText | 纯文本流，不产生 Artifact |

规则：

- `previewType` 是预览意图，不是 Artifact 类型。
- `previewType` 应优先等于 Runtime toolName。
- `previewType` 不等于 React 组件名。
- `artifact.type` 不是 `toolName`。
- `outputMode` 不是 `toolName`。
- `outputMode` 不是 `artifact.type`（除非映射表显式声明兼容）。
- 不得使用 `download` 作为 previewType（已废弃，统一使用 `file_download`）。
- 一个 `artifact.type` 可以有默认 `previewType`。
- 一个 Artifact 可以因为内容过大、安全策略或权限问题而 `preview.available = false`。
- 未注册的 `artifact.type` 不得预览。
- 未知 `previewType` 应安全降级为下载或纯文本摘要。
- `document_preview` / `data_preview` 当前为 planned/reserved，不要求实现。

---

## 12. 生命周期

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

## 13. 版本化规则

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

## 14. 身份与关联字段

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

## 15. Core Artifact ↔ Public API DTO 字段映射

Core Artifact 是平台内部事实源。Public API Artifact DTO 是对 Core Artifact 的公开投影，不得重新定义另一套事实源。

### 字段映射表

| Core Artifact（事实源） | Public API DTO（公开投影） | 说明 |
|---|---|---|
| `artifactId` | `id` | API 的 `id` 是 `artifactId` 的公开投影。DB `artifacts.id` 存储 `artifactId`。 |
| `type` | `type` | 直接透传。 |
| `title` | `title` | 直接透传。 |
| `mimeType` | `mimeType` | 直接透传。 |
| `content` | N/A | Core 内部字段，不直接暴露给 Public API。 |
| `contentRef`（object） | `contentRef`（string）/ `contentUrl` | Core `contentRef` 是对象型后端引用（`kind`、`bucket`、`key`、`sizeBytes`、`sha256`）。API 的 `contentRef` 字符串是公开授权访问入口 URL，不是 Core contentRef 对象。推荐 Public API 使用 `contentUrl` 字段避免混名。 |
| `summary` | `summary` | 直接透传。 |
| `metadata` | N/A | Core 内部字段，不直接暴露。 |
| `source.agentName` | N/A | Core 内部字段。 |
| `source.taskId` | N/A | Core 内部字段。 |
| `links.conversationId` | `conversationId` | API 扁平投影。Core 中嵌套在 `links` 对象下。 |
| `links.messageId` | `messageId` | API 扁平投影。Core 中嵌套在 `links` 对象下。 |
| `links.runId` | `runId` | API 扁平投影。Core 中嵌套在 `links` 对象下。 |
| `preview.previewType` | `previewType` | API 扁平投影。Core 中嵌套在 `preview` 对象下。 |
| `preview.available` | N/A | Core 内部字段，不直接暴露。 |
| `version`（integer） | `version`（integer） | 必须类型一致。不推荐字符串投影。 |
| `status` | `status` | 推荐直接对齐 Core 枚举：`pending` / `normalizing` / `ready` / `failed` / `superseded` / `deleted`。如需兼容旧 API，映射：`pending`/`normalizing` → `created`；`superseded` → `deleted`。 |
| `createdAt` | `createdAt` | 直接透传。 |
| `updatedAt` | `updatedAt` | 直接透传（如需）。 |
| `supersedes` | N/A | Core 内部版本链字段。 |

### ID 三层映射

Core Artifact.artifactId、DB artifacts.id、Public API Artifact.id 是**同一个系统 ID 在不同层级的命名**，不是三套不同 ID：

| 层级 | 字段名 | 说明 |
|---|---|---|
| Core Artifact（内部 JSON / schema） | `artifactId` | 事实源字段名，如 `art_01H...` |
| DB（MySQL `artifacts` 表） | `id`（或 `artifact_id`） | 存储 Core `artifactId` 值，不另行生成独立的 public id |
| Public API DTO（对外 JSON） | `id` | `artifactId` 的公开投影 |

命名边界：

- **schema 字段**：使用 `artifactId`。
- **DB 字段**：使用 `id` 或 `artifact_id`，二者必须映射到同一个 `artifactId` 值。
- **API DTO 字段**：使用 `id`，其值为 `artifactId` 的公开投影。
- 不强制修改数据库字段名，不新增迁移代码。
- 不得为 Public API 另行生成与 `artifactId` 不同的 public id。
- 文档中不应暗示 `artifactId`、DB `id`、API `id` 是三套不同 ID。

### 规则

- Core Artifact 的 `contentRef` 必须是对象型后端引用，不得是裸 URL 字符串。
- Public API 的 `contentRef` 字符串仅作为授权读取入口，不是 Core Artifact.contentRef。
- 推荐 Public API 新增 `contentUrl` / `downloadUrl` 字段用于公开访问地址，避免和核心 `contentRef` 混名。
- DB `artifacts.content_ref` 列存储 Core contentRef 对象的序列化 JSON，不是裸 URL。
- `links` 在 Core 中是嵌套对象，在 Public API 中扁平化是合法的 API 投影行为。
- `previewType` 在 Core 中嵌套于 `preview` 对象，在 Public API 中扁平化是合法的 API 投影行为。
- Gateway Handler 输出 Public API DTO 时负责将 Core Artifact 结构投影为 Public API 扁平结构。

---

## 16. 存储策略

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

## 17. 安全与脱敏

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

## 18. MVP v0.1 Historical Profile

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

## 19. v1.0 Required Artifact Types

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

## 20. Post-v1.0 Planned Types

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

## 21. Contract-first 规则

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

## 22. 禁止事项

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
- 把 ArtifactDraft 当作 Core Artifact。
- 在 ArtifactDraft 中提供 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段。
- 跳过归一化直接使用 ArtifactDraft 作为公开 API 响应。
- 把 artifact.type 直接当 toolName。
- 把 outputMode 直接当 toolName。
- 把 outputMode 直接当 artifact.type（除非映射表显式声明兼容）。
- 使用 `download` 作为 previewType（已废弃，统一使用 `file_download`）。
- 通过 agentName 推断 outputMode 或选择 toolName。

---

## 23. Review Checklist

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
- `previewType` 是否等于 Runtime toolName？
- 是否没有把组件名当类型？
- 是否没有把 `artifact.type` 直接当 toolName？
- 是否没有把 `outputMode` 直接当 toolName？
- 是否没有把 `outputMode` 直接当 `artifact.type`（除非映射表显式声明兼容）？
- 是否没有使用 `download` 作为 previewType？
- 是否没有把 toolName 当事实源？
- 是否所有 outputMode → artifact.type → toolName 转换都通过映射表表达？

### Content

- 小内容是否允许 inline？
- 大内容是否使用 `contentRef`？
- 是否没有 inline 二进制、zip、大日志、大 HTML 包？
- 是否没有裸露私有 URL？

### Identity

- 是否有 `artifactId`？
- 是否有关联 `conversationId / messageId / runId`？
- Core `artifactId` = DB `artifacts.id` = Public API `id` 是否是同一 ID？
- 是否没有为 Public API 另行生成与 `artifactId` 不同的 public id？
- 文档中是否没有暗示 artifactId、DB id、API id 是三套不同 ID？
- 多 Agent 场景是否有来源字段？
- 重名文件是否不会冲突？

### Normalization

- Child Agent / ADK 输出的是否是 ArtifactDraft（非 Core Artifact）？
- ArtifactDraft 是否只包含 `type`、`title`、`content`（或 `contentRefDraft`）、`metadata`？
- ArtifactDraft 是否没有 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段？
- 是否有归一化流程将 ArtifactDraft 转为 Core Artifact？
- 是否没有跳过归一化直接使用 ArtifactDraft？

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

## 24. 完成定义

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
