# Makefile 命令策略

## 推荐命令

```text
make docker-up
make docker-down
make docker-build
make docker-reset
make docker-logs
make docker-ps
make smoke-test
make smoke-test-ci
make demo-up
make demo-reset
```

## 语义

- `docker-up`：启动本地栈。
- `docker-down`：停止本地栈，不删除数据。
- `docker-build`：构建镜像。
- `docker-reset`：停止并删除 volume，危险操作。
- `docker-logs`：查看日志。
- `docker-ps`：查看服务状态。
- `smoke-test`：本地 smoke test。
- `smoke-test-ci`：CI 友好 smoke test。
- `demo-up`：Demo profile 启动。
- `demo-reset`：Demo 数据清理。

## 规则

- Makefile 不得隐藏破坏性行为。
- reset / clean 必须明确会删除 volume。
- 默认 up 不应删除数据。
- 命令失败时应返回非零退出码。
- 文档中不得列不存在的命令。
