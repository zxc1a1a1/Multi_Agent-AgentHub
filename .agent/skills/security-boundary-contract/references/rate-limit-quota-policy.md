# Rate Limit / Quota Policy

## 原则

所有消耗型能力都必须有上限。

## 规则

- Gateway public API 必须有基础 rate limit。
- SSE / stream 必须有连接数和超时限制。
- Orchestrator run 必须有最大 task 数。
- Planner 必须有 timeout。
- LLM 请求必须有 token、timeout、retry 上限。
- fallback / retry 必须有最大次数。
- Child Agent 调用必须有 timeout。
- Artifact 大小必须有限制。
- file_upload 必须有限制。

## 禁止

- 无限 retry。
- 无限 fallback。
- 无限 stream。
- 无限 Artifact。
- 不限制 LLM token 消耗。
