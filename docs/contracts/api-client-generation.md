# AgentHub REST API Client 生成规范

## 1. 文档目的

本规范用于约束后续 Frontend REST API 类型与客户端生成方式。  
`docs/contracts/openapi.yaml` 是 REST API 唯一事实源。  
Frontend 不能手写 API response 类型，必须基于 OpenAPI 生成。

## 2. 适用阶段

当前阶段只定义“生成规范”，不生成实际 client。  
实际生成动作应在 `frontend` 项目初始化之后进行。

## 3. 推荐输出目录

建议后续生成到：

```text
frontend/src/types/generated/
frontend/src/services/generated/
```

如果实际生成工具只支持单目录，也可以统一放到：

```text
frontend/src/generated/api/
```

## 4. 推荐生成方式

可选工具（本阶段仅记录，不安装、不执行）：

- `openapi-typescript`
- `openapi-fetch`
- `orval`
- `openapi-generator-cli`

具体采用哪个工具，由前端初始化后结合团队习惯统一决定。

## 5. 生成原则

- 类型必须来自 `docs/contracts/openapi.yaml`。  
- 不允许手写 API response 类型。  
- 不允许前端调用未进入 OpenAPI 的接口。  
- 不允许前端绕过 Gateway。  
- 不允许把 AG-UI 事件当作 REST API 类型。  
- 不允许把 A2A Task 当作前端 API 类型。  
- 生成物可以提交，也可以由构建生成，但团队必须统一。  
- OpenAPI 变更后必须重新生成类型。  

## 6. 命令占位（前端项目初始化后再执行）

```bash
# 示例：仅在 frontend 初始化后执行
npx openapi-typescript docs/contracts/openapi.yaml -o frontend/src/types/generated/platform-api.d.ts
```

```bash
# 示例：仅在 frontend 初始化后执行
npx orval --config frontend/orval.config.ts
```

```bash
# 示例：仅在 frontend 初始化后执行
npx openapi-generator-cli generate -i docs/contracts/openapi.yaml -g typescript-fetch -o frontend/src/generated/api
```

## 7. Review Checklist

- [ ] `docs/contracts/openapi.yaml` 是否已更新。  
- [ ] 生成类型是否已同步。  
- [ ] 是否没有手写 response 类型。  
- [ ] 是否没有调用未定义接口。  
- [ ] 是否没有把 AG-UI / A2A / internal contract 混入 REST client。  
- [ ] 是否保留统一响应 `code / data / message`。  
- [ ] 是否保留分页 `list / total / page / pageSize`。  

## MVP v0.1 生成范围

MVP v0.1 阶段可以只在前端使用以下接口对应的生成类型：

```text
GET  /api/conversations
POST /api/conversations
GET  /api/conversations/{id}/messages
GET  /api/agents
POST /api/agui/run
```

但类型生成仍然必须来自 `docs/contracts/openapi.yaml`。

前端不得因为 MVP 接口少而手写 response 类型。

如果 `openapi.yaml` 中保留了 Post-MVP planned 接口，前端可以暂不使用这些生成类型。
