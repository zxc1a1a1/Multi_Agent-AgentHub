# Makefile 命令

## 1. 目的

本文定义本地开发和 Demo 的标准命令。

## 2. 必需命令

```text
make docker-up
make docker-down
make docker-reset
make logs
make smoke
make dev
```

## 3. 命令语义

`make docker-up`：

```text
docker compose up -d --build
```

`make docker-down`：

```text
docker compose down
```

`make docker-reset`：

```text
docker compose down -v
```

`make logs`：

```text
docker compose logs -f
```

`make smoke`：

```text
scripts/smoke-test.sh
```

`make dev`：

```text
启动本地开发所需服务或打印开发说明
```

## 4. 可选命令

```text
make compose-config
make ps
make restart
make clean
make seed
```

## 5. 禁止事项

不得：

- Makefile 命令和 README 不一致。
- docker-up 后还需手动启动服务。
- reset 不清理 volume。
- smoke 不返回非零退出码。
