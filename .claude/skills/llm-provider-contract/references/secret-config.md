# Secret 配置规则

API key 只能来自：

- environment variable。
- secret manager。
- runtime injected secret。
- 本地开发专用 `.env`，但不得提交到仓库。

配置文件中只能写 env var 名称：

```yaml
providers:
  - name: anthropic-main
    type: anthropic
    apiKeyEnv: ANTHROPIC_API_KEY
```

不得写：

```yaml
apiKey: sk-...
```

API key 不得写入：

- config.yaml。
- AgentCard。
- ExecutionPlan。
- Artifact。
- Prompt。
- 日志。
- 错误信息。
- 前端响应。
- 数据库普通字段。
- test snapshot。
- README 示例中的真实值。

日志中如需记录 secret 状态，只能记录 present / missing。
