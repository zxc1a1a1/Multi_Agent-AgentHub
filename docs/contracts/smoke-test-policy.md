# Smoke Test Policy

## 目标

Smoke test 验证本地交付是否可运行。

## 必须检查

- `docker compose config`
- mysql health
- gateway health
- frontend reachable
- enabled child agents health
- agent list / registry API
- minimal run 或 mock run
- Demo profile 的关键路径

## 禁止

- 依赖真实 LLM 随机输出。
- 输出 secret。
- 失败后仍返回 0。
