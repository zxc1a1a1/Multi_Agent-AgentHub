# AG-UI Review Checklist

## Contract

- [ ] 是否更新 `docs/contracts/agui-events.md`
- [ ] 是否更新 `docs/contracts/agui-events.schema.json`
- [ ] 是否更新 `docs/contracts/agui-event-review-checklist.md`
- [ ] Skill 是否独立可读
- [ ] 是否避免依赖其他 Skill 才能理解

## SSE

- [ ] 是否按 `\n\n` 解析 event block
- [ ] 是否支持粘包
- [ ] 是否支持拆包
- [ ] 是否保留 incomplete buffer
- [ ] malformed JSON 是否不崩溃

## Message

- [ ] `TEXT_MESSAGE_START / CONTENT / END` 是否成对
- [ ] `messageId` 是否稳定
- [ ] 多 Agent 消息是否独立 `messageId`
- [ ] `sender.name` 是否不写死具体 Agent

## Tool Call

- [ ] `TOOL_CALL_START / ARGS / END` 是否成对
- [ ] `toolCallId` 是否稳定
- [ ] args 是否按 delta/content 聚合
- [ ] `TOOL_CALL_END` 后才解析执行
- [ ] 未知 toolName 是否安全降级

## State

- [ ] 是否支持 `STATE_UPDATE`
- [ ] 是否优先使用 `state` object
- [ ] 是否兼容 `content` JSON 字符串
- [ ] `phase` 是否合法
- [ ] retrying/fallback 是否可展示

## Error

- [ ] `RUN_ERROR` 后是否停止 loading
- [ ] 错误是否脱敏
- [ ] 错误码是否稳定
- [ ] 错误 message 是否适合展示
