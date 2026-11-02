# AgentHub Prometheus / Alerts

- `prometheus.yml`：抓取 Gateway `/internal/metrics` 与 Orchestrator
  `/internal/orchestrator/metrics[/routing]`，带 `format=prometheus&window=1h`。
- `alerts.yml`：可用性、run/approval/tool/SSE/路由决策告警规则。
- Orchestrator 指标需要服务 token：把 `INTERNAL_SERVICE_TOKEN` 写入
  `/etc/prometheus/orchestrator-token`（0600），不要提交到仓库。
- 挂载示例（宿主机启动 Prometheus）：

```bash
docker run --rm -p 9095:9090 \
  --network agenthub_default \
  -v "$PWD/deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  -v "$PWD/deploy/prometheus/alerts.yml:/etc/prometheus/alerts.yml:ro" \
  -v /etc/agenthub/prometheus-token:/etc/prometheus/orchestrator-token:ro \
  prom/prometheus:v3.1.0
```

- 本仓库未实际启动 Prometheus；规则文件用 YAML 解析验证，告警通道（Alertmanager）
  未配置。阈值是起点，按实际流量调整。
