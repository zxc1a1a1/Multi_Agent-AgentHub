# 后端开发规范：Go 命名、RESTful 接口、错误响应

本文档是本仓库后端开发的统一规范。目标是让团队成员写出来的接口、包名、方法名、错误响应和目录结构保持一致。

## 1. 总原则

- API 先从调用方视角设计，不围绕内部函数名暴露接口。
- URL 表示资源，HTTP Method 表示动作。
- 代码命名短、清楚、稳定；不要为了“显得完整”写很长的名字。
- 对外暴露的 API、JSON 字段和错误结构必须保持向后兼容。
- 协议类接口，例如 A2A、AG-UI、A2UI，可以保留协议命名，但要放在清晰的资源路径下。

## 2. RESTful URL 命名规范

### 2.1 路径必须版本化

统一使用：

```text
/api/v1/{resources}
```

示例：

```http
GET    /api/v1/agents
GET    /api/v1/agents/{agentId}
POST   /api/v1/agent-runs
POST   /api/v1/agent-runs:stream
```

不要这样：

```http
POST /api/chat
POST /api/createRun
GET  /api/getAgent
```

### 2.2 资源名使用复数名词

推荐：

```http
GET /api/v1/agents
GET /api/v1/agents/{agentId}
GET /api/v1/agent-runs
```

不推荐：

```http
GET /api/v1/getAgents
GET /api/v1/agent/{agentId}
```

### 2.3 不在 URL 里写 CRUD 动词

HTTP Method 已经表达动作：

| 操作 | 推荐接口 | 不推荐接口 |
|---|---|---|
| 创建运行 | `POST /api/v1/agent-runs` | `POST /api/v1/create-agent-run` |
| 查询运行 | `GET /api/v1/agent-runs/{runId}` | `GET /api/v1/get-agent-run/{runId}` |
| 更新资源 | `PATCH /api/v1/agents/{agentId}` | `POST /api/v1/update-agent` |
| 删除资源 | `DELETE /api/v1/agents/{agentId}` | `POST /api/v1/delete-agent` |

### 2.4 自定义动作使用冒号形式

当操作不是标准 CRUD，例如流式运行、重新执行、取消运行，可以使用 Google 风格的 custom method：

```http
POST /api/v1/agent-runs:stream
POST /api/v1/agent-runs/{runId}:cancel
POST /api/v1/agent-runs/{runId}:retry
```

不要滥用 custom method。能用标准 CRUD 表达时，优先使用标准 CRUD。

### 2.5 查询参数命名

查询参数统一使用 lowerCamelCase：

```http
GET /api/v1/agent-runs?pageSize=20&pageToken=abc&orderBy=createdAt desc
```

常用参数：

| 参数 | 作用 |
|---|---|
| `pageSize` | 每页数量 |
| `pageToken` | 下一页游标 |
| `filter` | 过滤条件 |
| `orderBy` | 排序条件 |

## 3. HTTP Method 使用规范

| Method | 用途 | 是否可重复调用 |
|---|---|---|
| `GET` | 查询资源，不改变服务端状态 | 是 |
| `POST` | 创建资源，或执行非幂等动作 | 通常不是 |
| `PATCH` | 局部更新资源 | 通常不是 |
| `PUT` | 整体替换资源 | 是 |
| `DELETE` | 删除资源 | 是 |

本项目默认：

- 局部更新使用 `PATCH`。
- 不推荐用 `PUT`，除非明确是“完整替换”。
- 查询接口不要使用请求体，参数放在 path 或 query 中。

## 4. JSON 字段命名

对外 JSON 字段统一使用 lowerCamelCase：

```json
{
  "runId": "run_123",
  "messageId": "msg_123",
  "createdAt": "2026-05-19T10:00:00Z"
}
```

Go 结构体字段使用 PascalCase，并通过 tag 映射：

```go
type AgentRun struct {
    RunID     string    `json:"runId"`
    MessageID string    `json:"messageId"`
    CreatedAt time.Time `json:"createdAt"`
}
```

## 5. 统一错误响应

所有 JSON API 的错误响应统一格式：

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "message is required",
    "details": {
      "field": "message"
    }
  }
}
```

常用错误码：

| HTTP 状态码 | `error.code` | 场景 |
|---|---|---|
| 400 | `invalid_argument` | 请求参数错误 |
| 401 | `unauthenticated` | 未登录或 token 无效 |
| 403 | `permission_denied` | 没有权限 |
| 404 | `not_found` | 资源不存在 |
| 409 | `conflict` | 状态冲突 |
| 429 | `rate_limited` | 请求过多 |
| 500 | `internal` | 服务端异常 |

## 6. Go 包名规范

包名应该：

- 小写。
- 单个短单词优先。
- 不用下划线。
- 不用 mixedCaps。
- 不重复包名上下文。

推荐：

```text
internal/config
internal/httpapi
internal/service
internal/agent
pkg/protocol/a2a
pkg/protocol/agui
pkg/protocol/a2ui
```

不推荐：

```text
internal/http_api
internal/AgentService
internal/agent_manager_package
```

## 7. Go 类型、方法、接口命名

### 7.1 类型名

- 对外类型用 PascalCase：`AgentService`、`RootAgent`、`AgentRun`。
- 内部类型用 lowerCamelCase：`createAgentRunRequest`。
- 避免名字重复包名，例如在 `agent` 包中不要写 `AgentRootAgent`。

### 7.2 方法名

服务层方法使用业务动作：

```go
func (s *AgentService) Run(ctx context.Context, message string) AgentResult
func (s *AgentService) Cancel(ctx context.Context, runID string) error
```

HTTP handler 使用 `handle + 资源动作`：

```go
func (r *Router) handleCreateAgentRun(w http.ResponseWriter, req *http.Request)
func (r *Router) handleStreamAgentRun(w http.ResponseWriter, req *http.Request)
func (r *Router) handleGetAgentCard(w http.ResponseWriter, req *http.Request)
```

### 7.3 接口名

- 单方法接口优先使用 `-er` 后缀：`Runner`、`Reader`、`Writer`。
- 多方法接口按职责命名：`SubAgent`、`AgentStore`。
- 不使用 `I` 前缀：不要写 `IAgentService`。
- 不使用 `Interface` 后缀：不要写 `AgentServiceInterface`。

### 7.4 构造函数

单一核心类型的包，构造函数用 `New`：

```go
func New() *Client
```

当前包中有多个核心类型时，用 `NewTypeName`：

```go
func NewRouter(cfg config.Config) http.Handler
func NewAgentService() *AgentService
func NewRootAgent() *RootAgent
```

## 8. 文件命名规范

Go 文件名统一小写，多个单词用下划线：

```text
agent_service.go
error_response.go
router.go
handlers.go
```

测试文件：

```text
agent_service_test.go
router_test.go
```

## 9. 本项目当前接口规范

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/healthz` | 存活检查 |
| `GET` | `/.well-known/agent-card.json` | A2A Agent Card 发现入口 |
| `POST` | `/api/v1/agent-runs` | 创建一次 Agent 运行，返回完整 JSON |
| `POST` | `/api/v1/agent-runs:stream` | 创建一次 Agent 运行，并通过 SSE 返回 AG-UI 事件流 |

## 10. Pull Request 检查项

提交后端代码前，必须确认：

- [ ] 新接口遵循 `/api/v1` 版本前缀。
- [ ] 资源名是复数名词。
- [ ] 没有在 URL 中使用 `get/create/update/delete`。
- [ ] JSON 字段是 lowerCamelCase。
- [ ] 错误响应使用统一结构。
- [ ] Go 包名小写、短、无下划线。
- [ ] 导出类型和函数有注释。
- [ ] 已执行 `gofmt`。
- [ ] 已执行 `go test ./...`。

## 参考资料

- Microsoft Azure Architecture Center: Web API design best practices
- Google Cloud API Design Guide and AIP standard methods
- Effective Go
- Go Blog: Package names
- Uber Go Style Guide
