# Artifact Schema Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Artifacts are produced by ADK/Child Agents and eventually mapped to frontend runtime skills by Orchestrator/Runtime AG-UI translator.

## Shape

```json
{
  "type": "code|webpage|document|file|diff|terminal|image",
  "title": "main.go",
  "content": "...",
  "metadata": {"language":"go"}
}
```

## Mapping

| Artifact type | Skill |
|---|---|
| `code` | `code_preview` |
| `webpage` | `web_preview` |
| `document` | `markdown_render` |
| `file` | `file_download` |
| `diff` | `diff_preview` |
| `terminal` | `terminal_output` |

## Rules

- Artifact content must not include secrets.
- Large file artifacts should use references/URLs only when storage profile is active.
- Unknown artifact types must degrade to text or file card, not crash the stream.

## Storage metadata（v3 A0-04 冻结）

状态等级：`已实现（渲染制品路径）`。元数据仍由 `LocalFileStore` 持久化；渲染制品额外写入
`storageRef/checksum/contentType/version`，blob 字节通过 `artifacts.ObjectStore` 存取：
本地目录（默认）或 S3/MinIO（`ARTIFACT_S3_ENDPOINT`）。读取入口
`GET /internal/orchestrator/artifacts/{id}/content` 会校验 checksum。

```json
{
  "artifactId": "art_...",
  "storageRef": "local://workspace/run-1/out.pdf",
  "checksum": "sha256:<64 hex>",
  "contentType": "application/pdf",
  "size": 123456,
  "version": 1,
  "createdAt": "2026-10-31T00:00:00Z"
}
```

规则：

1. `storageRef` 是不透明定位符，不向模型/前端暴露宿主绝对路径或内部 URL。
2. `checksum` 为内容 SHA-256；下载/预览前可校验，校验失败视为不可用。
3. `contentType` 必须是 IANA MIME；未知类型降级为 `application/octet-stream` 文本卡片。
4. `size` 以字节计；超过上限转分片/压缩或拒绝，不静默截断。
5. `version` 从 1 开始，同一 artifactId 内容变化必须递增；旧版本保留期由存储 profile 决定。
6. 生产者：Agent `artifact.publish` / Orchestrator artifact 适配器；消费者：Gateway 预览/下载、前端 `file_download`。
7. 缺少 `storageRef` 或 `checksum` 的 artifact 不得标记为可下载；跨节点读取必须由 A6-03 的对象存储适配器保证。

## 渲染制品（v3 A5-05/A5-11）

- 生产者：`POST /internal/orchestrator/render`
  `{runId,taskId,url,format:"png"|"pdf",fullPage}`；web-agent 通过
  `capture_page` 工具（`WEB_AGENT_ENABLE_CAPTURE=true`）调用。
- 渲染器：`PLAYWRIGHT_RENDER_SCRIPT` 指向的 Playwright 脚本；URL 在渲染前
  拒绝非 http(s)、凭据、loopback/私网/链路本地/metadata/CGNAT 目标。
- 输出：PNG → `kind=image`，PDF → `kind=document`；写入 blob 后创建元数据：
  `storageRef=obj://runs/<run>/<artifactId>.<ext>`、`checksum=sha256:<hex>`、
  `contentType`、`size`、`version=1`、`downloadPath`。
- 校验和 Content endpoint：`GET /internal/orchestrator/artifacts/{id}/content`
  读取 blob 并校验 SHA-256；不匹配返回 409，缺失返回 404。
- 跨节点：配置 S3/MinIO 时 blob 可跨 Orchestrator 副本读取；未配置时使用本地
  目录，仅单节点可用。前端自动展示渲染制品事件仍未接入。
