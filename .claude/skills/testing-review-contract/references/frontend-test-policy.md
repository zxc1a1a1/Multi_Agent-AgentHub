# Frontend Test Policy

前端测试应验证用户可见行为。

## Component Tests

- MessageBubble senderName。
- AgentAvatar 通用 fallback。
- WebPreview iframe。
- StreamingText markdown。
- ErrorBoundary safe error。

## Store Tests

- per-conversation streaming。
- abortControllers per conversation。
- TOOL_CALL assembly。
- STATE_UPDATE activeAgent / retrying。
