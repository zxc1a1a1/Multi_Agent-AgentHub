# R0-08 OTel/Prometheus/Alertmanager 运行认证

- commit: ee833cd3c59a40c1e98a51bb4358534317862928
- started: 2026-11-03T03:56:06Z / ended: 2026-11-03T03:57:57Z
- result: **PASS**（12 项断言，0 项失败）
- compose project: agenthub-r008 / profile: observability
- provider: fake LLM (http://127.0.0.1:18081)；无外部告警通道

## 断言结果

| 场景 | 结果 | 证据 |
|---|---|---|
| build orchestrator/gateway/code-agent | PASS |  |
| jaeger/collector/prometheus/alertmanager/webhook healthy | PASS |  |
| host processes up (code-agent, orchestrator, gateway) | PASS |  |
| R0-08 trace run completed | PASS | run=run_6c8725407c0ec059 |
| R0-08 phase1 trace gateway->orchestrator->code-agent | PASS | run=run_6c8725407c0ec059 |
| Prometheus gateway target UP | PASS |  |
| Prometheus orchestrator target UP | PASS |  |
| test alert delivered to local webhook | PASS |  |
| services stay healthy with collector down | PASS |  |
| R0-08 outage run completed | PASS | run=run_8b75c628c5381096 |
| R0-08 recovery run completed | PASS | run=run_4fb9d7ec44c1b002 |
| R0-08 recovery trace gateway->orchestrator->code-agent | PASS | run=run_4fb9d7ec44c1b002 |

## 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: API 级冒烟 |
| API 请求/响应 | events.jsonl | 2 | yes（confirm） |
| SSE/AG-UI | logs/*-stream.txt | 3 | yes（三个 run 的流文件） |
| HITL | timeline.jsonl | 3 | yes |
| 工具/记忆/Artifact | - | 0 | not_applicable: 假 LLM 计划不触发写工具，session_only |
| 多实例 | - | 0 | not_applicable: 单 Orchestrator 进程；多副本见 R0-06 |
| 观测/告警 | logs/phase1-jaeger.json、prom-*.json、webhook-alerts.jsonl | 4 | yes |
| 故障注入 | timeline.jsonl | 2 | yes（collector stop/start） |

## 修复项（本任务内）

1. **OTEL service.name 丢失**：`observability.Setup` 原用
   `resource.Merge(resource.Default(), resource.NewWithAttributes(ServiceName))`，实际导出的服务名仍是
   `unknown_service:*`，导致 Jaeger 无法按服务查询。改为 `resource.New(... WithAttributes(ServiceName))`；
   修复后 Jaeger 服务名为 `gateway` / `orchestrator` / `code-agent`，trace 可按 run 串联。
2. **Alertmanager 单节点 gossip**：首次通知在 gossip settle 前被丢弃（`prometheus_notifications_dropped_total=1`）。
   单节点配置 `--cluster.listen-address=`，并让 Prometheus `depends_on` Alertmanager healthy；
   冒烟在健康后再 restart Prometheus，规则首次评估即送达。
3. **观测端口参数化**：`PROMETHEUS_PORT` / `ALERTMANAGER_PORT` / `JAEGER_PORT` / `ALERT_WEBHOOK_PORT`
   可覆盖，避免与本机 9090 冲突；默认值仍为标准端口。
4. 新增本地 webhook receiver（只落盘 `/data/alerts.jsonl`，不转发外部）与 smoke-only 规则
   `AgentHubObservabilitySmoke`（不进入生产 rule_files）。

## 校验

- `promtool check config`：SUCCESS（2 个 rule files，7 + 1 rules）
- `amtool check-config`：SUCCESS
- 证据：`logs/validation.txt`

## 已知边界

- 三个业务进程运行在宿主机、观测栈运行在 compose project `agenthub-r008`；Prometheus 通过
  `host.docker.internal` 抓取。compose 内默认 target（`gateway-new` / `orchestrator-new`）未在此场景使用。
- 告警通道是本地 webhook，未接入真实 pager；provider 为 fake LLM。
- 采样率默认 1.0（`OTEL_TRACES_SAMPLE_RATIO`）；生产建议按比例采样后再验收。

## 证据文件
- commands.txt / events.jsonl / timeline.jsonl / logs/（含 validation.txt、Jaeger JSON、Prometheus 查询、webhook payload）
