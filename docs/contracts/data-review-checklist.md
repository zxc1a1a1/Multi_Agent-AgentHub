# 数据持久化 Review 契约清单

## 阶段

- [ ] 是否没有把 MVP 历史基线当成当前禁令？
- [ ] 是否支持 2+ Agent？
- [ ] 是否支持 single / group conversation？
- [ ] 是否不固定具体 Agent 名称？

## 表结构

- [ ] 是否有 conversations？
- [ ] 是否有 conversation_participants？
- [ ] 是否有 messages？
- [ ] 是否有 agents？
- [ ] 是否有 agent_health_checks 或等价字段？
- [ ] 是否有 runs？
- [ ] 是否有 run_steps？
- [ ] 是否有 agent_tasks 或兼容表？
- [ ] 是否有 tool_calls？
- [ ] 是否有 artifacts？

## 关联

- [ ] message 是否关联 conversation？
- [ ] run 是否关联 conversation？
- [ ] run 是否能关联多 message？
- [ ] agent_task 是否关联 run？
- [ ] artifact 是否关联 message / run / agent？
- [ ] tool_call 是否关联 message / artifact？
- [ ] 多 Agent 消息是否能追溯 agentName？

## Migration

- [ ] 是否有版本化 migration？
- [ ] 是否有 up migration？
- [ ] 破坏性变更是否使用 expand / migrate / contract？
- [ ] 是否避免手工改库不留记录？
- [ ] 是否更新数据模型文档？

## JSON

- [ ] JSON 字段是否有结构说明？
- [ ] 常用查询字段是否没有藏在 JSON？
- [ ] JSON 是否没有保存 secret？
- [ ] JSON 是否没有保存大文件？

## 安全

- [ ] 是否没有明文 API key / token？
- [ ] 是否没有完整 system prompt？
- [ ] 是否没有未脱敏 LLM 请求响应？
- [ ] 是否没有永久公开文件 URL？
- [ ] error_message 是否脱敏？

## 查询与索引

- [ ] 聊天历史查询是否有索引？
- [ ] run trace 查询是否有索引？
- [ ] Agent 健康状态查询是否有索引？
- [ ] 群聊参与者查询是否有索引？
