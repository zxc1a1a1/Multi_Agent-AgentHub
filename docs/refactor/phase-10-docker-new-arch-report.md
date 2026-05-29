# Phase 10.1~10.4 Docker 新架构 Demo 基线报告（更新）

## 1. 范围说明
本报告记录 Phase 10.1~10.4 的基线成果，包含：
- `services/gateway/Dockerfile`
- `docker-compose.new-arch.yml`
- `smoke-new-arch.ps1` / `smoke-new-arch.sh`
- `Makefile` 新架构 targets
- `frontend/nginx.conf` `/api` 反代基础接入

说明：Phase 10.5~10.7 稳定化成果已单独记录在：
- `docs/refactor/phase-10-demo-stabilization-report.md`

## 2. 已完成基线
- 新增 Gateway 多阶段镜像构建。
- 新增 `docker-compose.new-arch.yml`，不影响旧 `docker-compose.yml`。
- 新增新架构 smoke 脚本（PowerShell + sh）。
- `Makefile` 增加新架构 Docker/smoke 入口。
- 前端 nginx 加入 `/api` 代理链路（用于容器化 demo）。

## 3. 技术边界
- 本链路是 `frontend -> gateway -> child agents` 的最小 demo。
- 不实现 Orchestrator / Planner / Executor。
- 不引入真实 LLM、不连接真实数据库。
- `/api/agents` 为静态注册表输出。

## 4. 安全边界
- 不读取或提交 `.env`。
- 不写入真实 API key、token、DATABASE_URL 或私钥。
- 前端不直连子 Agent，不直连 A2A endpoint。
- unknown-agent 错误需脱敏。

## 5. 后续工作
Phase 10.5~10.7 已继续推进 Docker Demo 稳定化，详见：
- `docs/refactor/phase-10-demo-stabilization-report.md`
- `docs/refactor/new-arch-demo-acceptance-checklist.md`
- `docs/refactor/new-arch-demo-troubleshooting.md`
