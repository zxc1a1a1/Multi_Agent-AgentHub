# Demo Delivery Checklist

Demo 前检查：

- Compose config 通过。
- `.env.example` 完整。
- 本机 `.env` 已配置。
- `docker compose up --build` 可启动。
- 所有基础服务 healthy。
- 当前启用 Agent healthy。
- v1.0 Demo profile 至少 2 个 Child Agent。
- smoke test 通过。
- 前端为生产构建。
- 日志无 secret。
- reset 可清理 Demo 数据。
- README 写明启动步骤。
