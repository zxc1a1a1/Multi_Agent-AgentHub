# 交付 Review Checklist

## Compose

- 是否有主 Compose 文件？
- 是否能通过 `docker compose config`？
- 是否没有真实 secret？
- 是否没有本机绝对路径？
- 容器间是否使用 service name？

## 服务

- frontend 是否可访问？
- gateway 是否可访问？
- mysql 是否 healthy？
- Child Agent 是否都有 `/health`？
- 新增 Agent 是否有 build/image/env/healthcheck？

## 环境

- `.env.example` 是否完整？
- `.env` 是否被 gitignore？
- secret 是否未写入 Dockerfile？
- 新增变量是否有说明？

## Smoke

- 是否有 smoke test？
- 是否检查服务健康？
- 是否检查启用的 Agent？
- 是否不依赖真实 LLM 随机输出？
- 失败是否返回非零退出码？

## Demo

- 是否一键启动？
- 是否一键停止？
- 是否有日志命令？
- 是否有 reset 命令？
- 是否有 Demo checklist？
