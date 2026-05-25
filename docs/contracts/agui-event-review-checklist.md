# AG-UI Event Review Checklist

## Contract 文件

- [ ] 已更新 `docs/contracts/agui-events.md`
- [ ] 已更新 `docs/contracts/agui-events.schema.json`
- [ ] 已更新本 Review Checklist
- [ ] 事件字段说明完整
- [ ] 事件顺序说明完整

## SSE

- [ ] Gateway 每个 AG-UI event 输出一个 SSE block
- [ ] Gateway 每次写入后 flush
- [ ] Frontend 按 `\n\n` 解析 event block
- [ ] Frontend 支持粘包、拆包、半包
- [ ] malformed JSON 不导致前端崩溃

## Run 生命周期

- [ ] 正常流程有 `RUN_STARTED`
- [ ] 正常流程有 `RUN_FINISHED`
- [ ] 错误流程有 `RUN_ERROR`
- [ ] `RUN_FINISHED` 后不再发业务事件
- [ ] `RUN_ERROR` 后不再发业务事件

## Text Message

- [ ] `TEXT_MESSAGE_START / CONTENT / END` 成对
- [ ] `messageId` 稳定
- [ ] 多 Agent 消息不复用 `messageId`
- [ ] `TEXT_MESSAGE_CONTENT` 新实现使用 `delta`
- [ ] 前端兼容旧字段 `content`
- [ ] 大型产物没有塞进 `TEXT_MESSAGE_CONTENT`

## Tool Call

- [ ] `TOOL_CALL_START / ARGS / END` 成对
- [ ] `toolCallId` 稳定
- [ ] `TOOL_CALL_ARGS` 新实现使用 `delta`
- [ ] 前端兼容旧字段 `content`
- [ ] `TOOL_CALL_END` 后才解析 args
- [ ] 未知 `toolName` 安全降级

## STATE_UPDATE

- [ ] 支持 `STATE_UPDATE`
- [ ] 新实现优先使用 `state` object
- [ ] 前端兼容 `content` JSON 字符串
- [ ] `phase` 属于允许枚举
- [ ] `retrying` / `fallback` 可展示
- [ ] `activeAgent` 不写死具体 Agent 名称

## 多 Agent / 群聊

- [ ] 一个 Run 可以有多条 assistant message
- [ ] 每条 message 有独立 `sender`
- [ ] Tool Call 通过 `messageId` 归属
- [ ] ordered-parallel 不要求 token 交错

## 安全

- [ ] `RUN_ERROR` 已脱敏
- [ ] event 不包含 API key
- [ ] event 不包含 Authorization token
- [ ] event 不包含数据库连接串
- [ ] event 不包含内部堆栈
- [ ] HTML 类 Tool Call 不直接注入父页面 DOM
