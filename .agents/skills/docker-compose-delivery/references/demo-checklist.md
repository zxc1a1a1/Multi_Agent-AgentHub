# Demo Checklist

## 1. 目的

本文定义 AgentHub Demo 前检查清单。

## 2. Demo 前检查

必须确认：

```text
.env exists
.env.example up to date
API key present or mock mode enabled
docker compose config passes
docker compose up works
mysql healthy
gateway /health ok
code-agent /health ok
frontend opens
smoke-test passes
```

## 3. 功能检查

必须确认：

```text
open frontend
new conversation
select code-agent
send prompt
streaming reply visible
code preview visible
copy code works
refresh keeps history
docker compose down works
```

## 4. 日志检查

必须确认：

```text
no panic
no fatal
no obvious 500 loop
no API key in logs
no token in logs
```

## 5. 禁止事项

不得：

- Demo 前临时手动启动隐藏服务。
- Demo 使用未提交的本地配置说明。
- Demo 日志输出真实 secret。
- Demo 完成后没有 down / cleanup 说明。
