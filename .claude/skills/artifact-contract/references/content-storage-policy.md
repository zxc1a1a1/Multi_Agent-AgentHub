# Content Storage 规则

## 1. 目的

本文定义 Artifact content、contentRef、数据库和对象存储的边界。

## 2. 长期 baseline

长期规则：

```text
AG-UI 不传大内容。
AG-UI 只传 artifactId + summary 或最小 preview args。
完整内容通过 REST API 查询。
大代码包 / HTML / 图片 / 文档 / zip / 日志进入 Object Storage。
Artifact metadata 进入数据库。
Redis 不得作为 Artifact 唯一存储。
```

## 3. content

`content` 适用于：

- 小型文本。
- MVP small code。
- 可安全 inline 的结构化数据。

## 4. contentRef

`contentRef` 适用于：

- 大型代码包。
- HTML 包。
- 图片。
- 文档。
- zip。
- 日志。
- 私有文件。
- 需要鉴权下载的内容。

推荐结构：

```ts
type ContentRef = {
  kind: "object_storage" | "database" | "external_url" | "generated";
  uri?: string;
  objectKey?: string;
  mimeType?: string;
  sizeBytes?: number;
  checksum?: string;
};
```

## 5. MVP 例外

MVP 允许：

```text
small code artifact inline
message.artifacts JSON 临时保存
不要求 Object Storage
不要求独立 Artifact 表
```

MVP inline 只允许：

```text
code
```

## 6. 私有 URL

不得把私有对象存储签名 URL 长期写入 Artifact 并直接暴露给前端。

下载鉴权和签名 URL 生成属于：

```text
platform-api-contract
security-boundary-contract
```

## 7. 禁止事项

不得：

- 大型 Artifact 进入 AG-UI token 流。
- 图片 inline 进 TOOL_CALL_ARGS。
- zip inline 进 TOOL_CALL_ARGS。
- HTML 包 inline 进 TOOL_CALL_ARGS。
- 私有文件 inline 进 TOOL_CALL_ARGS。
- Redis 作为 Artifact 唯一事实源。
- MVP inline 规则扩展到所有 Artifact 类型。
