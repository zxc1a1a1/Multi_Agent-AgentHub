# Artifact API 策略

Artifact API 返回平台资源形态，不定义前端组件实现。

推荐资源：

- `GET /api/artifacts/{artifactId}`
- `GET /api/artifacts/{artifactId}/content`
- `GET /api/artifacts/{artifactId}/preview`
- `GET /api/artifacts/{artifactId}/download`

大内容必须使用 `contentRef` 或下载入口，下载 URL 必须短期有效，不返回对象存储原始凭证或内部路径。
