# Fixture / Mock / Fake Policy

## Fixture

- 必须脱敏。
- 必须覆盖 valid 和 invalid。
- 不得包含真实 API key、token、用户隐私。

## Mock / Fake

- Fake LLM 必须确定性。
- Fake Agent 必须通用命名。
- Fake Registry 必须支持 status（enabled/disabled）与 health（healthy/unhealthy）。
- Replay fixtures 必须包含中断和 malformed event。
