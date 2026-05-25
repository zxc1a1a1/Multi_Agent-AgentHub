# Frontend 镜像策略

## 目标

Demo profile 中前端应使用生产构建镜像，而不是开发服务器。

## 允许模式

- Node 构建 + `serve dist`。
- Node 构建 + nginx 静态服务。
- dev profile 中使用 Vite dev server。

## 规则

- Demo 默认路径不得依赖 Vite dev server。
- 前端镜像应使用多阶段构建。
- API base URL 必须文档化。
- 如果 nginx 代理 `/api`，必须考虑流式接口。
- SSE / streaming 路径不得被 proxy buffering 卡住。
- 前端默认宿主机端口建议为 `3000`。

## 禁止

- Demo 使用未构建的源码开发服务器。
- 把真实 API key 注入前端镜像。
- 把后端内网地址硬编码进前端产物。
