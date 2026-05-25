# 鉴权 Header 策略

公开 Platform API 使用：

```text
Authorization: Bearer <user-token>
```

规则：token 不得放入 query string；所有用户资源必须做对象级授权；用户 token 不得复用为 Gateway ↔ Orchestrator 的 service token。
