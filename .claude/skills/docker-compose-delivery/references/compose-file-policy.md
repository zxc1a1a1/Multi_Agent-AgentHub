# Compose 文件策略

## 1. 目的

本文定义 AgentHub Compose 文件规则。

## 2. 文件命名

MVP 可使用：

```text
docker-compose.yml
```

正式开发推荐：

```text
compose.yaml
compose.override.yaml
```

## 3. MVP 必需 services

```text
frontend
gateway
code-agent
mysql
```

## 4. 端口

```text
frontend: 3000
gateway: 8080
code-agent: 8081
mysql: 3306
```

## 5. build context

推荐：

```text
frontend → ./frontend
gateway → ./server
code-agent → ./agents/code-agent
```

## 6. depends_on

`depends_on` 只能表达启动顺序，不代表 ready。

需要配合：

```text
healthcheck
condition: service_healthy
wait-for-health.sh
```

## 7. 禁止事项

不得：

- 只靠 depends_on 判断服务 ready。
- 只靠 sleep 等数据库。
- 在 Compose 中写真实 API key。
- 硬编码本机绝对路径。
- 将本地 Compose 写成生产部署方案。
