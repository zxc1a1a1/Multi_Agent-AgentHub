# Demo Checklist

Demo 前必须检查：

- `docker compose config` 通过。
- `.env.example` 完整。
- 本机 `.env` 已配置必要变量。
- `docker compose up --build` 可启动。
- mysql healthy。
- gateway healthy。
- frontend 可访问。
- 当前启用的 Child Agent healthy。
- v1.0 Demo profile 至少启用 2 个 Child Agent。
- smoke test 通过。
- 前端使用生产构建或 demo build。
- 日志中没有 secret。
- 停止重启后核心数据符合预期。
- reset 命令可清理演示数据。
- README 或交付文档说明启动步骤。
