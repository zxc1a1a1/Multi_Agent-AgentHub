# Streaming Event Test Policy

流式事件必须可 replay、可恢复、可取消。

## 必测场景

- SSE block 使用双换行分隔。
- incomplete block buffer。
- malformed JSON 不崩溃。
- message / toolCall 顺序稳定。
- STATE_UPDATE 不污染消息正文。
- 多 Agent messageId / senderName 不混淆。
- cancel 后停止普通 delta。
