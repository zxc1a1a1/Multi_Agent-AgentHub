# Artifact 交付策略

- Gateway 负责将 A2A Artifact 转为前端 artifact event。
- Gateway 应补充 `artifactId`、`artifactRef`、`previewSkill`、`metadata`。
- Orchestrator 负责聚合多 Agent 产物并维护 `provenance`。
- 大产物应先存储再传引用。
- 失败产物应产生可展示错误状态，不泄露内部 stack trace 或密钥。
