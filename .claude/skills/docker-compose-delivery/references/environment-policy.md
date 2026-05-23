# 环境变量策略

## 1. 目的

本文定义 `.env.example` 和环境变量规则。

## 2. 必须提交

```text
.env.example
```

## 3. 不得提交

```text
.env
.env.local
*.secret.env
```

## 4. MVP 必需变量

```text
DATABASE_URL
ANTHROPIC_API_KEY
GATEWAY_PORT
CODE_AGENT_PORT
AGENT_CODE_URL
```

## 5. 示例值

`.env.example` 中只能使用占位值：

```text
ANTHROPIC_API_KEY=replace-me
DATABASE_URL=mysql://agenthub:agenthub@mysql:3306/agenthub
AGENT_CODE_URL=http://code-agent:8081
```

## 6. Secret 规则

真实 secret 只能来自本地 `.env`、Docker secrets、secret manager 或 runtime injection。

不得写入 Compose 文件、Dockerfile、README、日志或 image。

## 7. 禁止事项

不得：

- 提交真实 API key。
- 在 Dockerfile ENV 写 secret。
- 在 compose environment 中写真实 key。
- 在日志中打印 env 全量内容。
- 缺失必填环境变量时静默失败。
