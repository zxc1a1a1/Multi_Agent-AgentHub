# Secrets 策略

## 目的

防止 API key、token、数据库密码和其他敏感信息进入仓库、镜像或日志。

## 允许

- 本机 `.env`，且必须被 gitignore。
- shell 环境变量。
- Compose secrets。
- 外部 secret manager。

## 禁止

- 把真实 API key 写入 `.env.example`。
- 把真实 API key 写入 Dockerfile。
- 把真实 API key 写入 Compose 文件。
- 把 secrets 文件提交到仓库。
- 在 smoke test 输出 secret。
- 在容器日志输出 secret。

## Compose secrets 规则

如果使用 Compose secrets：

- 顶层声明 secret。
- 服务只挂载自己需要的 secret。
- 应用从 `/run/secrets/<name>` 读取。
- secrets 源文件不得被 git 跟踪。

## Review

每次修改 Compose、Dockerfile、README、脚本时，都应检查是否误写 secret。
