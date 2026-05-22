# Security Review Checklist

## 1. MVP 安全检查

- [ ] Gateway 是否鉴权 `/api/*`？
- [ ] 是否使用 `Authorization: Bearer <token>`？
- [ ] Token 是否不在 query string？
- [ ] LLM API key 是否只来自环境变量？
- [ ] API key 是否没有写进代码？
- [ ] RUN_ERROR 是否不泄漏 stack trace？
- [ ] REST ErrorResponse 是否不泄漏内部错误？
- [ ] code_preview 是否只展示不执行？
- [ ] A2A endpoint 是否没有暴露给 Frontend？
- [ ] Gateway handler 是否没有直接调用 A2A endpoint？

## 2. Secret 检查

- [ ] AgentCard 是否不包含密钥？
- [ ] OpenAPI 示例是否不包含密钥？
- [ ] Contract 示例是否不包含密钥？
- [ ] 日志是否不打印 token / API key？
- [ ] `.env.example` 是否没有真实值？

## 3. Artifact 检查

- [ ] 大 Artifact 是否不塞进文本流？
- [ ] HTML 预览是否 sandbox？
- [ ] 文件下载是否鉴权？
- [ ] 私有 URL 是否短期有效？
- [ ] Artifact metadata 是否不包含密钥？

## 4. 高危能力检查

- [ ] run_command 是否默认未启用？
- [ ] file_upload 是否默认未启用？
- [ ] deploy 是否需要 confirm_action？
- [ ] 文件覆盖是否需要 confirm_action？
- [ ] 外部付费调用是否需要 confirm_action？

## 5. Post-MVP 检查

- [ ] JWT signing secret 是否安全管理？
- [ ] A2A 是否有 service-to-service auth？
- [ ] 自建 Agent 是否有 sandbox？
- [ ] 是否有 audit log？
- [ ] 是否有 rate limit？
