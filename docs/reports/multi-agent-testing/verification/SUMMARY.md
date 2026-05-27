# AgentHub 子 Agent 冒烟验证结果

每个 Agent 加载真实 `config.yaml`，通过 httptest 启动内存 server，
抓取 `/health` 和 `/.well-known/agent.json` 的实际响应内容。

## 验证清单

| Agent | 端口 | /health | AgentCard 必填字段 | 安全扫描 |
|---|---|---|---|---|
| agent-builder-agent | 8100 | ✅ OK | ✅ OK | ✅ CLEAN |
| artifact-agent | 8096 | ✅ OK | ✅ OK | ✅ CLEAN |
| code-agent | 8081 | ✅ OK | ✅ OK | ✅ CLEAN |
| context-agent | 8091 | ✅ OK | ✅ OK | ✅ CLEAN |
| custom-agent | 8084 | ✅ OK | ✅ OK | ✅ CLEAN |
| deploy-agent | 8104 | ✅ OK | ✅ OK | ✅ CLEAN |
| diff-agent | 8093 | ✅ OK | ✅ OK | ✅ CLEAN |
| document-agent | 8083 | ✅ OK | ✅ OK | ✅ CLEAN |
| file-agent | 8101 | ✅ OK | ✅ OK | ✅ CLEAN |
| ppt-agent | 8105 | ✅ OK | ✅ OK | ✅ CLEAN |
| qa-acceptance-agent | 8103 | ✅ OK | ✅ OK | ✅ CLEAN |
| release-agent | 8106 | ✅ OK | ✅ OK | ✅ CLEAN |
| review-agent | 8094 | ✅ OK | ✅ OK | ✅ CLEAN |
| security-agent | 8097 | ✅ OK | ✅ OK | ✅ CLEAN |
| test-agent | 8095 | ✅ OK | ✅ OK | ✅ CLEAN |
| version-agent | 8099 | ✅ OK | ✅ OK | ✅ CLEAN |
| vision-agent | 8092 | ✅ OK | ✅ OK | ✅ CLEAN |
| web-agent | 8082 | ✅ OK | ✅ OK | ✅ CLEAN |
| web-research-agent | 8102 | ✅ OK | ✅ OK | ✅ CLEAN |
| **cross-agent: vision→web** | — | — | — | ✅ 2/2 PASS |
| **cross-agent: vision→code** | — | — | — | ✅ 2/2 PASS |
| **cross-agent: file→document** | — | — | — | ✅ 2/2 PASS |
| **cross-agent: security→review** | — | — | — | ✅ 2/2 PASS |
| **concurrent: ADK servers** | — | — | — | ✅ 9/9 PASS (zero races) |
| **concurrent: agent parsers** | — | — | — | ✅ 4/4 PASS (zero races) |
| **stream-replay: event order** | — | — | — | ✅ 4/4 PASS |
| **stream-replay: error & fault safety** | — | — | — | ✅ 3/3 PASS |
| **stream-replay: cancel & interruption** | — | — | — | ✅ 3/3 PASS |
| **stream-replay: malformed & edge cases** | — | — | — | ✅ 2/2 PASS |
| **stream-replay: multi-agent & metadata** | — | — | — | ✅ 2/2 PASS |

## 输出目录

```
verification/
├── agent-builder-agent/
│   ├── health.json
│   └── agent-card.json
├── artifact-agent/
│   ├── health.json
│   └── agent-card.json
├── code-agent/
│   ├── health.json
│   └── agent-card.json
├── context-agent/
│   ├── health.json
│   └── agent-card.json
├── custom-agent/
│   ├── health.json
│   └── agent-card.json
├── deploy-agent/
│   ├── health.json
│   └── agent-card.json
├── diff-agent/
│   ├── health.json
│   └── agent-card.json
├── document-agent/
│   ├── health.json
│   └── agent-card.json
├── file-agent/
│   ├── health.json
│   └── agent-card.json
├── ppt-agent/
│   ├── health.json
│   └── agent-card.json
├── qa-acceptance-agent/
│   ├── health.json
│   └── agent-card.json
├── release-agent/
│   ├── health.json
│   └── agent-card.json
├── review-agent/
│   ├── health.json
│   └── agent-card.json
├── security-agent/
│   ├── health.json
│   └── agent-card.json
├── test-agent/
│   ├── health.json
│   └── agent-card.json
├── version-agent/
│   ├── health.json
│   └── agent-card.json
├── vision-agent/
│   ├── health.json
│   └── agent-card.json
├── web-agent/
│   ├── health.json
│   └── agent-card.json
├── web-research-agent/
│   ├── health.json
│   └── agent-card.json
├── cross-agent/
│   ├── vision-to-web.json
│   ├── vision-to-code.json
│   ├── file-to-document.json
│   └── security-to-review.json
├── concurrent/
│   ├── adk-concurrency.json
│   └── agent-parser-concurrency.json
├── stream-replay/
│   └── stream-replay.json
```

每个 `health.json` 和 `agent-card.json` 即为对应 Agent A2A 端点的实际 HTTP 响应内容。
