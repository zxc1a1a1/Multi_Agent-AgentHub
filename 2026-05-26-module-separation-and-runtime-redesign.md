# Module Separation & Runtime Redesign — 完整实施方案

**Date:** 2026-05-26
**Status:** Approved


---

## 1. Overview

将当前 Multi_Agent-AgentHub 项目拆分为 5 个独立模块，通过明确的接口契约解耦，并在 ADK 引擎之上构建完整的 Runtime 工程化框架。

**目标架构层次：**

```
业务应用 (Child Agents) — YAML + 少量 Go 代码
         ↓ import
Runtime 框架 (pkg/runtime) — 配置驱动 / 注册表 / AG-UI / Skill / Pruning
         ↓ import
ADK 引擎 (pkg/adk) — Agent / Runner / Session / Plugin / Tool / A2A
```

**核心原则：**
- ADK 是纯引擎，零业务依赖
- Runtime 是工程化框架，实现 ADK 定义的接口
- Gateway 是前端唯一入口，只做协议转换+持久化
- Orchestrator 是编排大脑，独立服务
- Child Agent 完全自治，通过 A2A 协议接入

---

## 2. Target Directory Structure

```
Multi_Agent-AgentHub/
├── go.work                           # Go workspace（本地开发聚合）
│
├── pkg/
│   ├── adk/                          # go.mod: github.com/zxc1a1a1/agenthub/pkg/adk
│   │   ├── agent.go                  # Agent 接口定义
│   │   ├── runner.go                 # 多轮 LLM↔Tool 循环 Runner
│   │   ├── session.go                # Session 接口 + 内存实现
│   │   ├── plugin.go                 # Plugin/Callback 接口
│   │   ├── tool.go                   # Tool 接口
│   │   ├── event.go                  # Event 类型定义（Text/ToolCall/ToolResult/StateDelta）
│   │   ├── llmagent.go              # LLM Agent 实现（带 Instruction/Callback）
│   │   ├── model.go                 # Model 接口（由 Runtime 实现）
│   │   └── a2a/                      # A2A 协议适配层
│   │       ├── server.go             # 暴露 Agent 为 A2A 端点
│   │       ├── client.go             # 调用远程 A2A Agent
│   │       └── types.go              # AgentCard 构建/校验
│   │
│   └── runtime/                      # go.mod: github.com/zxc1a1a1/agenthub/pkg/runtime
│       ├── config/                   # YAML 配置加载 + Setup 顺序
│       │   ├── config.go             # RuntimeConfig + Setup()
│       │   └── expand.go             # ${ENV} 变量展开
│       ├── registry/                 # 全局工厂注册表
│       │   ├── agent.go              # Agent Factory Registry
│       │   ├── tool.go               # Tool Factory Registry (type/set/func)
│       │   └── model.go              # Model Registry
│       ├── model/                    # LLM Provider 实现
│       │   ├── provider.go           # Provider 接口
│       │   ├── anthropic.go          # Anthropic Messages API
│       │   ├── openai.go             # OpenAI ChatCompletions
│       │   └── proxy.go              # 通用 OpenAI 兼容代理
│       ├── session/                  # Session Service
│       │   ├── service.go            # 接口定义
│       │   ├── memory.go             # 内存实现
│       │   └── mysql.go              # MySQL 实现
│       ├── context/                  # 上下文裁剪
│       │   ├── pruner.go             # ChainPruner + 各策略
│       │   └── paired.go             # EnsurePairedFunctionCalls
│       ├── skill/                    # Skill 系统
│       │   ├── repository.go         # Markdown Skill 仓库
│       │   ├── manager.go            # 加载/卸载 状态管理
│       │   └── toolset.go            # skill_load/unload/list 工具集
│       ├── agui/                     # AG-UI 协议层
│       │   ├── translator.go         # 有状态 Event→AG-UI 转换器
│       │   ├── filter.go             # TextStreamFilter 接口 + 缓冲评估
│       │   └── events.go             # AG-UI 事件类型定义
│       └── launcher/                 # 一键启动
│           └── launcher.go           # Gin 路由组装 + 服务注册
│
├── services/
│   ├── gateway/                      # go.mod (imports runtime)
│   │   ├── cmd/main.go              # 入口
│   │   ├── config.go                # Gateway 自身配置
│   │   ├── server.go                # Gin Engine 组装
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   │   ├── agui.go          # POST /agui/runs → SSE 响应
│   │   │   │   ├── conversation.go  # CRUD /api/conversations
│   │   │   │   ├── agent.go         # GET /api/agents
│   │   │   │   └── health.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go          # Token 鉴权
│   │   │   │   ├── cors.go
│   │   │   │   └── requestid.go     # Trace ID 注入
│   │   │   └── store/
│   │   │       ├── interface.go     # Store 接口
│   │   │       ├── mysql.go         # Conversation/Message 持久化
│   │   │       └── mysql_test.go
│   │   ├── proto/
│   │   │   └── orchestrator.proto   # Gateway→Orchestrator gRPC 定义
│   │   └── Dockerfile
│   │
│   ├── orchestrator/                 # go.mod (imports adk + runtime)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── server/
│   │   │   │   └── grpc.go          # gRPC server 实现
│   │   │   ├── planner/
│   │   │   │   ├── planner.go       # 意图识别 + 编排计划生成
│   │   │   │   └── planner_test.go
│   │   │   ├── executor/
│   │   │   │   ├── executor.go      # 执行引擎（串行/并行/条件）
│   │   │   │   ├── single.go        # 单 Agent 执行
│   │   │   │   ├── sequential.go    # 顺序执行多 Agent
│   │   │   │   └── parallel.go      # 并行执行多 Agent
│   │   │   ├── router/
│   │   │   │   ├── router.go        # Agent 路由（能力匹配）
│   │   │   │   └── registry.go      # Agent 注册表
│   │   │   └── converter/
│   │   │       └── a2a_to_agui.go   # A2A Event → AG-UI Event 转换
│   │   ├── config.yaml              # Orchestrator 配置
│   │   ├── proto/
│   │   │   ├── orchestrator.proto
│   │   │   └── orchestrator_grpc.pb.go
│   │   └── Dockerfile
│   │
│   └── agents/
│       ├── code-agent/              # go.mod (imports adk)
│       │   ├── cmd/main.go
│       │   ├── handler.go
│       │   ├── tools.go
│       │   ├── config.yaml
│       │   └── Dockerfile
│       ├── web-agent/
│       ├── review-agent/
│       └── ...
│
├── frontend/                         # 完全独立（package.json）
│   ├── src/
│   ├── package.json
│   └── Dockerfile
│
├── docs/
│   ├── contracts/
│   └── superpowers/specs/
│
├── docker-compose.yml                # 全栈一键启动
└── Makefile                          # build/test/lint/up/down
```

**五个模块的职责边界：**

| 模块 | 职责 | 不负责 |
|------|------|--------|
| `pkg/adk` | Agent 抽象、Runner 循环、Session 接口、Plugin 接口、Tool 接口、Event 定义、A2A 协议 | 不知道配置怎么来、不知道 LLM Provider 具体实现 |
| `pkg/runtime` | YAML 配置、工厂注册表、LLM Provider 实现、Session 实现、Context 裁剪、Skill、AG-UI Translator、Launcher | 不含业务逻辑、不含具体 Agent handler |
| `services/gateway` | HTTP 入口、鉴权、前端 AG-UI SSE 推送、Conversation 持久化、对接 Orchestrator | 不做 Agent 编排决策 |
| `services/orchestrator` | 意图识别、路由、多 Agent 并行/串行编排、调用 Child Agent | 不处理 HTTP 请求、不存消息 |
| `services/agents/*` | 具体业务逻辑（代码生成/Web 搜索等）、使用 ADK 构建、暴露 A2A 端点 | 不关心前端协议 |

---

## 3. ADK Engine Layer (`pkg/adk`) — Full Implementation Design

ADK 是纯接口 + 最小实现的底层引擎，**零外部业务依赖**，只依赖标准库 + `a2a-go/v2`。

### 3.1 Core Interfaces

```go
// ========== agent.go ==========

// Agent 是所有 Agent 的顶层抽象
type Agent interface {
    Name() string
    // Generate 执行一次 LLM 推理（由 Runner 循环调用）
    Generate(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)
}

type GenerateRequest struct {
    Contents    []*Content          // 完整对话历史（经裁剪后）
    Tools       []Tool              // 可用工具列表
    Config      *GenerateConfig     // 温度/MaxTokens 等
}

type GenerateResponse struct {
    Parts       []Part              // 输出内容（Text/ToolCall 混合）
    FinishReason FinishReason       // Stop / ToolUse / MaxTokens
    Usage       *UsageMetadata      // token 用量
}

type FinishReason string
const (
    FinishStop     FinishReason = "stop"
    FinishToolUse  FinishReason = "tool_use"
    FinishMaxToken FinishReason = "max_tokens"
)
```

```go
// ========== event.go ==========

// Event 是 Runner 产出的事件流单元
type Event struct {
    ID        string
    Author    string              // 产出该事件的 Agent 名
    Content   *Content            // 本轮产出内容
    Actions   *EventActions       // 副作用
    Partial   bool                // true = 流式增量 chunk
    Final     bool                // true = 本轮最终事件
    Timestamp time.Time
}

type EventActions struct {
    StateDelta    map[string]any   // State KV 变更
    TransferAgent string           // 切换到另一个 SubAgent
    ArtifactDelta []Artifact       // 产物变更
}

// Content 统一消息体
type Content struct {
    Role  Role
    Parts []Part
}

type Role string
const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
    RoleSystem    Role = "system"
)

// Part 多模态内容片段
type Part interface{ partMarker() }

type TextPart struct {
    Text string
}

type ToolCallPart struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

type ToolResultPart struct {
    CallID  string
    Name    string
    Content string
    IsError bool
}

type ThinkingPart struct {
    Thinking string
}
```

```go
// ========== tool.go ==========

// Tool 是 Agent 可调用的工具
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage        // JSON Schema for parameters
    Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

type ToolResult struct {
    Content string
    IsError bool
}
```

```go
// ========== plugin.go ==========

// Plugin 在 Runner 执行的关键节点注入逻辑
type Plugin interface {
    // BeforeGenerate 在每次 LLM 调用前触发（可修改 request）
    BeforeGenerate(ctx context.Context, state *SessionState, req *GenerateRequest) error
    // AfterGenerate 在 LLM 返回后触发
    AfterGenerate(ctx context.Context, state *SessionState, resp *GenerateResponse) error
    // BeforeTool 在工具执行前触发
    BeforeTool(ctx context.Context, state *SessionState, call *ToolCallPart) error
    // AfterTool 在工具执行后触发
    AfterTool(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error
}

// BasePlugin 空实现，业务可选择性覆盖
type BasePlugin struct{}
func (BasePlugin) BeforeGenerate(...) error { return nil }
func (BasePlugin) AfterGenerate(...) error  { return nil }
func (BasePlugin) BeforeTool(...) error     { return nil }
func (BasePlugin) AfterTool(...) error      { return nil }
```

```go
// ========== session.go ==========

// Session 存储一轮对话的完整执行状态
type Session struct {
    ID        string
    UserID    string
    Events    []Event             // 全部历史事件
    State     *SessionState       // KV 状态
    CreatedAt time.Time
    UpdatedAt time.Time
}

// SessionState 线程安全的 KV store
type SessionState struct {
    mu   sync.RWMutex
    data map[string]any
}

func (s *SessionState) Get(key string) (any, bool) { ... }
func (s *SessionState) Set(key string, val any)    { ... }
func (s *SessionState) All() map[string]any        { ... }

// SessionService 定义 Session 的 CRUD
type SessionService interface {
    Create(ctx context.Context, userID string, initialState map[string]any) (*Session, error)
    Get(ctx context.Context, id string) (*Session, error)
    AppendEvent(ctx context.Context, sessionID string, event Event) error
    UpdateState(ctx context.Context, sessionID string, delta map[string]any) error
    List(ctx context.Context, userID string) ([]*Session, error)
    Delete(ctx context.Context, id string) error
}
```

```go
// ========== model.go ==========

// Model 是 LLM 调用的抽象接口（由 Runtime 实现）
type Model interface {
    GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error]
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}
```

### 3.2 Runner — Multi-Turn Execution Engine

```go
// ========== runner.go ==========

// RunConfig 控制 Runner 行为
type RunConfig struct {
    MaxIterations  int            // 最大 LLM↔Tool 循环次数（防无限循环，默认 25）
    StreamMode     StreamMode     // SSE / Full
    Plugins        []Plugin       // callback 链
}

type StreamMode int
const (
    StreamModeSSE  StreamMode = iota  // 流式：每个 chunk 都 yield
    StreamModeFull                     // 完整：等最终结果再 yield
)

// Runner 驱动 Agent 的多轮执行
type Runner struct {
    agent   Agent
    session SessionService
    tools   []Tool
    plugins []Plugin
    config  RunConfig
}

func NewRunner(agent Agent, session SessionService, opts ...RunOption) *Runner { ... }

// Run 执行一次完整对话，返回事件流迭代器
func (r *Runner) Run(ctx context.Context, sessionID string, userContent *Content) iter.Seq2[Event, error] {
    return func(yield func(Event, error) bool) {
        session, _ := r.session.Get(ctx, sessionID)
        
        // 将用户输入追加到 session
        userEvent := Event{Content: userContent, Author: "user"}
        r.session.AppendEvent(ctx, sessionID, userEvent)
        
        // 构建初始对话历史
        contents := r.buildContents(session)
        
        for iteration := 0; iteration < r.config.MaxIterations; iteration++ {
            // ① BeforeGenerate callback 链
            req := &GenerateRequest{Contents: contents, Tools: r.tools}
            for _, p := range r.plugins {
                p.BeforeGenerate(ctx, session.State, req)
            }
            
            // ② 调用 Agent.Generate
            resp, err := r.agent.Generate(ctx, req)
            if err != nil { yield(Event{}, err); return }
            
            // ③ AfterGenerate callback 链
            for _, p := range r.plugins {
                p.AfterGenerate(ctx, session.State, resp)
            }
            
            // ④ 处理响应
            event := r.buildEvent(session, resp)
            
            if resp.FinishReason == FinishToolUse {
                // 有 ToolCall：执行工具，结果回到对话历史
                yield(event, nil)  // 先 yield ToolCall 事件
                
                toolResults := r.executeTools(ctx, session, resp.Parts)
                resultEvent := r.buildToolResultEvent(toolResults)
                yield(resultEvent, nil)
                
                // 把 ToolCall + ToolResult 追加到 contents，继续循环
                contents = append(contents, event.Content, resultEvent.Content)
                continue
            }
            
            // 纯文本响应：结束循环
            event.Final = true
            yield(event, nil)
            
            // 更新 session
            r.session.AppendEvent(ctx, sessionID, event)
            if event.Actions != nil && event.Actions.StateDelta != nil {
                r.session.UpdateState(ctx, sessionID, event.Actions.StateDelta)
            }
            return
        }
        
        // 超过最大迭代次数
        yield(Event{}, fmt.Errorf("max iterations (%d) exceeded", r.config.MaxIterations))
    }
}

// executeTools 并行执行所有 ToolCall
func (r *Runner) executeTools(ctx context.Context, session *Session, parts []Part) []ToolResultPart {
    var results []ToolResultPart
    for _, part := range parts {
        tc, ok := part.(ToolCallPart)
        if !ok { continue }
        
        // BeforeTool
        for _, p := range r.plugins { p.BeforeTool(ctx, session.State, &tc) }
        
        // 查找并执行 tool
        tool := r.findTool(tc.Name)
        result, err := tool.Execute(ctx, tc.Arguments)
        if err != nil {
            result = &ToolResult{Content: err.Error(), IsError: true}
        }
        
        // AfterTool
        for _, p := range r.plugins { p.AfterTool(ctx, session.State, &tc, result) }
        
        results = append(results, ToolResultPart{
            CallID: tc.ID, Name: tc.Name,
            Content: result.Content, IsError: result.IsError,
        })
    }
    return results
}
```

### 3.3 LLM Agent Implementation

```go
// ========== llmagent.go ==========

// LLMAgent 是基于大模型的 Agent 实现
type LLMAgent struct {
    name        string
    model       Model                // 由 Runtime 的 Model Registry 注入
    instruction string               // system prompt
    subAgents   []Agent              // 子 Agent（用于 transfer）
    tools       []Tool               // Agent 专属工具
}

type LLMAgentConfig struct {
    Name        string
    Model       Model
    Instruction string
    SubAgents   []Agent
    Tools       []Tool
}

func NewLLMAgent(cfg LLMAgentConfig) *LLMAgent { ... }

func (a *LLMAgent) Name() string { return a.name }

func (a *LLMAgent) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    // 注入 system instruction
    if a.instruction != "" {
        req.Contents = prepend(systemContent(a.instruction), req.Contents)
    }
    // 合并 Agent 专属工具
    req.Tools = append(req.Tools, a.tools...)
    // 如有 SubAgents，注入 transfer_to_agent 工具
    if len(a.subAgents) > 0 {
        req.Tools = append(req.Tools, buildTransferTools(a.subAgents)...)
    }
    return a.model.Generate(ctx, req)
}
```

### 3.4 A2A Adapter (`pkg/adk/a2a/`)

```go
// ========== a2a/server.go ==========

// A2AServer 将 Agent+Runner 包装为标准 A2A JSON-RPC 端点
// 暴露: /health, /.well-known/agent.json, POST / (JSON-RPC)
type A2AServer struct {
    config  *AgentConfig
    runner  *Runner
    mux     *http.ServeMux
}

func NewServer(config *AgentConfig, runner *Runner) *A2AServer { ... }
func (s *A2AServer) Run(addr string) error { ... }
func (s *A2AServer) Handler() http.Handler { ... }
```

```go
// ========== a2a/client.go ==========

// RemoteAgent 把远程 A2A 服务包装为本地 Agent 接口
type RemoteAgent struct {
    name      string
    client    *a2aclient.Client
    agentURL  string
}

func NewRemoteAgent(name, url string) *RemoteAgent { ... }

func (r *RemoteAgent) Name() string { return r.name }
func (r *RemoteAgent) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    // 把 GenerateRequest 转为 A2A Message，发送到远程，收集结果
    ...
}
```

---

## 4. Runtime Framework Layer (`pkg/runtime`) — Full Implementation Design

Runtime 在 ADK 之上，提供所有工程化能力。它 **import ADK**，但 **ADK 不 import Runtime**。

### 4.1 Config — YAML Configuration + Setup Sequence

```go
// ========== config/config.go ==========

// RuntimeConfig 是整个框架的配置入口
type RuntimeConfig struct {
    Models  ModelsConfig  `yaml:"models"`
    Tools   ToolsConfig   `yaml:"tools"`
    Skills  SkillsConfig  `yaml:"skills"`
    Session SessionConfig `yaml:"session"`
    Agents  AgentsConfig  `yaml:"agents"`
}

// LoadConfig 从 YAML 文件加载配置（支持 ${ENV} 展开）
func LoadConfig(path string) (*RuntimeConfig, error) {
    data, _ := os.ReadFile(path)
    expanded := expandEnvVars(data)   // ${VAR} → os.Getenv("VAR")
    var cfg RuntimeConfig
    yaml.Unmarshal(expanded, &cfg)
    return &cfg, nil
}

// Setup 严格按顺序初始化所有子系统
// 依赖关系：Agent 依赖 Model/Tool/Skill，所以 Agent 最后
func (c *RuntimeConfig) Setup(ctx context.Context) error {
    if err := c.Models.Setup(ctx); err != nil { return err }   // ① Model Registry
    if err := c.Tools.Setup(ctx); err != nil { return err }    // ② Tool Registry
    if err := c.Skills.Setup(ctx); err != nil { return err }   // ③ Skill Repository
    if err := c.Session.Setup(ctx); err != nil { return err }  // ④ Session Factory
    if err := c.Agents.Setup(ctx); err != nil { return err }   // ⑤ Agent Factory（依赖上面全部）
    return nil
}
```

**对应 YAML 示例（`conf/agents.yaml`）：**

```yaml
models:
  - name: claude-sonnet
    provider: anthropic
    api_key_env: ANTHROPIC_API_KEY
    model: claude-sonnet-4-20250514
    base_url: https://api.anthropic.com
    max_tokens: 8192

  - name: gpt-4o
    provider: openai
    api_key_env: OPENAI_API_KEY
    model: gpt-4o
    base_url: https://api.openai.com/v1

  - name: deepseek-v3
    provider: proxy
    api_key_env: DEEPSEEK_API_KEY
    model: deepseek-chat
    base_url: https://api.deepseek.com/v1

tools:
  - type: local
    set: code_tools
    config:
      workspace_dir: /tmp/workspace
      max_file_size: 1048576

  - type: local
    set: web_tools
    config:
      timeout: 30s
      user_agent: "AgentHub/1.0"

skills:
  repository: ./skills          # Markdown skill 文件目录
  max_loaded: 5                 # 单 session 最大同时加载数

session:
  backend: memory               # memory | mysql
  mysql:
    dsn_env: SESSION_DATABASE_URL

agents:
  - name: code-agent
    model: claude-sonnet
    instruction: |
      You are a code generation assistant.
      Output code in fenced blocks with language:filename format.
    tools:
      - local/code_tools/generate_code
      - local/code_tools/review_code
    skills:
      - code_generation
    streaming: true
    context_pruning:
      strategy: keep_ends_window
      keep_first: 3
      keep_last: 20
      max_tool_result_len: 4096
    max_iterations: 25

  - name: web-agent
    model: gpt-4o
    instruction: |
      You are a web research assistant.
    tools:
      - local/web_tools/search
      - local/web_tools/fetch
    subagents:
      - code-agent                # 本地 SubAgent
    streaming: true
```

### 4.2 Registry — Factory Registration Tables

三套并行注册表，统一用 `init()` + `Register()` 模式：

```go
// ========== registry/tool.go ==========

// ToolFactory 创建工具实例
type ToolFactory interface {
    // Setup 接收 YAML 配置
    Setup(config map[string]any) error
    // NewTools 按 func 名创建具体工具
    NewTools(ctx context.Context, funcName string) ([]adk.Tool, error)
    // ListFunctions 列出该 set 下所有可用函数名
    ListFunctions() []string
}

// 全局注册表：tools[type][set] = ToolFactory
var (
    toolsMu sync.RWMutex
    tools   = make(map[string]map[string]ToolFactory)
)

// Register 在 init() 中调用
func RegisterTool(typ, set string, factory ToolFactory) {
    toolsMu.Lock()
    defer toolsMu.Unlock()
    if tools[typ] == nil {
        tools[typ] = make(map[string]ToolFactory)
    }
    tools[typ][set] = factory
}

// ResolveTool 按 "type/set/func" 三段解析
func ResolveTool(ctx context.Context, path string) ([]adk.Tool, error) {
    parts := strings.SplitN(path, "/", 3)
    if len(parts) != 3 {
        return nil, fmt.Errorf("invalid tool path %q, expected type/set/func", path)
    }
    typ, set, funcName := parts[0], parts[1], parts[2]
    
    toolsMu.RLock()
    factory, ok := tools[typ][set]
    toolsMu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("tool factory not found: %s/%s", typ, set)
    }
    return factory.NewTools(ctx, funcName)
}
```

```go
// ========== registry/model.go ==========

// ModelProvider 创建 adk.Model 实例
type ModelProvider interface {
    Name() string  // "anthropic" | "openai" | "proxy"
    CreateModel(ctx context.Context, cfg ModelConfig) (adk.Model, error)
}

type ModelConfig struct {
    Name     string `yaml:"name"`
    Provider string `yaml:"provider"`
    APIKeyEnv string `yaml:"api_key_env"`
    Model    string `yaml:"model"`
    BaseURL  string `yaml:"base_url"`
    MaxTokens int   `yaml:"max_tokens"`
}

var (
    modelProviders = make(map[string]ModelProvider)   // provider name → factory
    modelInstances = make(map[string]adk.Model)       // model config name → instance
)

func RegisterModelProvider(p ModelProvider) {
    modelProviders[p.Name()] = p
}

func ResolveModel(ctx context.Context, configName string) (adk.Model, error) {
    if m, ok := modelInstances[configName]; ok {
        return m, nil
    }
    // ... lookup config, create via provider, cache
}
```

```go
// ========== registry/agent.go ==========

// AgentFactory 创建 adk.Agent 实例
type AgentFactory interface {
    Setup(cfg AgentYAMLConfig) error
    NewAgent(ctx context.Context) (adk.Agent, error)
}

var (
    agentFactories = make(map[string]AgentFactory)
)

// RegisterAgent 业务代码在 init() 中注册自定义 Factory
func RegisterAgent(name string, factory AgentFactory) {
    agentFactories[name] = factory
}

// Get 获取 Agent Factory
func GetAgentFactory(name string) (AgentFactory, bool) {
    f, ok := agentFactories[name]
    return f, ok
}
```

### 4.3 Model — LLM Provider Implementations

```go
// ========== model/provider.go ==========

// provider.go 定义统一的内部请求/响应
type streamChunk struct {
    DeltaText    string
    ToolCalls    []adk.ToolCallPart
    FinishReason adk.FinishReason
    Usage        *adk.UsageMetadata
}
```

```go
// ========== model/anthropic.go ==========

type anthropicProvider struct{}

func init() {
    registry.RegisterModelProvider(&anthropicProvider{})
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) CreateModel(ctx context.Context, cfg registry.ModelConfig) (adk.Model, error) {
    return &anthropicModel{
        apiKey:    os.Getenv(cfg.APIKeyEnv),
        model:     cfg.Model,
        baseURL:   cfg.BaseURL,
        maxTokens: cfg.MaxTokens,
        client:    &http.Client{Timeout: 120 * time.Second},
    }, nil
}

type anthropicModel struct {
    apiKey    string
    model     string
    baseURL   string
    maxTokens int
    client    *http.Client
}

// Generate 实现 adk.Model 接口 — 同步调用
func (m *anthropicModel) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
    // 构建 Anthropic Messages API 请求体
    // POST /v1/messages (stream: false)
    // 解析完整响应，转换为 GenerateResponse
    ...
}

// GenerateStream 实现 adk.Model 接口 — 流式调用
func (m *anthropicModel) GenerateStream(ctx context.Context, req *adk.GenerateRequest) iter.Seq2[*adk.GenerateResponse, error] {
    return func(yield func(*adk.GenerateResponse, error) bool) {
        // 构建请求（stream: true）
        // Header: x-api-key, anthropic-version: 2023-06-01
        // SSE 解析循环:
        //   event.type == "content_block_delta" && delta.type == "text_delta"
        //     → yield partial response with TextPart
        //   event.type == "content_block_delta" && delta.type == "input_json_delta"
        //     → accumulate tool call arguments
        //   event.type == "message_delta" && delta.stop_reason == "tool_use"
        //     → yield final response with ToolCallParts
        ...
    }
}
```

```go
// ========== model/openai.go ==========
// 结构类似 anthropic.go，但使用 OpenAI ChatCompletions 格式
// Header: Authorization: Bearer {key}
// SSE 解析: choices[0].delta.content / choices[0].delta.tool_calls

type openaiProvider struct{}
func init() { registry.RegisterModelProvider(&openaiProvider{}) }
func (p *openaiProvider) Name() string { return "openai" }
```

```go
// ========== model/proxy.go ==========
// 复用 openai 实现，只改 baseURL
// 支持 DeepSeek / Moonshot / Ollama / vLLM 等任何 OpenAI 兼容接口

type proxyProvider struct{}
func init() { registry.RegisterModelProvider(&proxyProvider{}) }
func (p *proxyProvider) Name() string { return "proxy" }
```

### 4.4 Session Service Implementations

```go
// ========== session/memory.go ==========

type memorySessionService struct {
    mu       sync.RWMutex
    sessions map[string]*adk.Session
}

func NewMemorySessionService() adk.SessionService {
    return &memorySessionService{sessions: make(map[string]*adk.Session)}
}

func (s *memorySessionService) Create(ctx context.Context, userID string, initialState map[string]any) (*adk.Session, error) {
    session := &adk.Session{
        ID:        uuid.New().String(),
        UserID:    userID,
        State:     adk.NewSessionState(initialState),
        CreatedAt: time.Now(),
    }
    s.mu.Lock()
    s.sessions[session.ID] = session
    s.mu.Unlock()
    return session, nil
}

func (s *memorySessionService) AppendEvent(ctx context.Context, sessionID string, event adk.Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    sess := s.sessions[sessionID]
    sess.Events = append(sess.Events, event)
    sess.UpdatedAt = time.Now()
    return nil
}
```

```go
// ========== session/mysql.go ==========

type mysqlSessionService struct {
    db *sql.DB
}

func NewMySQLSessionService(dsn string) (adk.SessionService, error) { ... }

// Session 表: id, user_id, state_json, created_at, updated_at
// Event 表:  id, session_id, author, content_json, actions_json, seq, created_at
```

### 4.5 Context Pruning

```go
// ========== context/pruner.go ==========

// Pruner 裁剪对话历史
type Pruner interface {
    Prune(contents []*adk.Content) []*adk.Content
}

// ChainPruner 按顺序应用多个 Pruner
type ChainPruner struct {
    pruners []Pruner
}

func (c *ChainPruner) Prune(contents []*adk.Content) []*adk.Content {
    for _, p := range c.pruners {
        contents = p.Prune(contents)
    }
    return contents
}

// KeepEndsWindowPruner 保留前 N 条 + 后 M 条，中间删除
type KeepEndsWindowPruner struct {
    KeepFirst int
    KeepLast  int
}

func (p *KeepEndsWindowPruner) Prune(contents []*adk.Content) []*adk.Content {
    if len(contents) <= p.KeepFirst+p.KeepLast {
        return contents
    }
    result := make([]*adk.Content, 0, p.KeepFirst+p.KeepLast)
    result = append(result, contents[:p.KeepFirst]...)
    result = append(result, contents[len(contents)-p.KeepLast:]...)
    return result
}

// ToolResultTruncator 截断过长的工具结果
type ToolResultTruncator struct {
    MaxLen int
}

func (t *ToolResultTruncator) Prune(contents []*adk.Content) []*adk.Content {
    // 遍历所有 Content 中的 ToolResultPart
    // 如果 Content JSON 长度 > MaxLen，截断为 MaxLen 并追加 "...[truncated]"
    ...
}
```

```go
// ========== context/paired.go ==========

// EnsurePairedFunctionCalls 确保裁剪后 ToolCall 和 ToolResult 仍配对
// 如果有 ToolCall 没有对应 Result（或反之），删除孤立的那一侧
type EnsurePairedFunctionCalls struct{}

func (e *EnsurePairedFunctionCalls) Prune(contents []*adk.Content) []*adk.Content {
    // 1. 收集所有 ToolCallPart 的 ID → 所在 index
    // 2. 收集所有 ToolResultPart 的 CallID → 所在 index
    // 3. 找出不配对的 ID
    // 4. 删除包含孤立 ToolCall/ToolResult 的 Part
    // 5. 如果某个 Content 的所有 Parts 都被删了，删除整个 Content
    ...
}
```

**集成到 Plugin：**

```go
// PruningPlugin 在 BeforeGenerate 时执行裁剪
type PruningPlugin struct {
    pruner Pruner
}

func NewPruningPlugin(cfg PruningConfig) *PruningPlugin {
    chain := &ChainPruner{pruners: []Pruner{
        &ToolResultTruncator{MaxLen: cfg.MaxToolResultLen},
        &KeepEndsWindowPruner{KeepFirst: cfg.KeepFirst, KeepLast: cfg.KeepLast},
        &EnsurePairedFunctionCalls{},
    }}
    return &PruningPlugin{pruner: chain}
}

func (p *PruningPlugin) BeforeGenerate(ctx context.Context, state *adk.SessionState, req *adk.GenerateRequest) error {
    req.Contents = p.pruner.Prune(req.Contents)
    return nil
}
```

### 4.6 Skill System

```go
// ========== skill/repository.go ==========

// Skill 从 Markdown 文件解析
type Skill struct {
    Name        string            // frontmatter: name
    Description string            // frontmatter: description
    Tags        []string          // frontmatter: tags
    Instruction string            // Markdown body（注入 system prompt）
}

// Repository 管理 Skill 文件
type Repository struct {
    dir    string
    skills map[string]*Skill     // name → Skill
}

func NewRepository(dir string) (*Repository, error) {
    // 扫描 dir 下所有 .md 文件
    // 解析 YAML frontmatter + Markdown body
    ...
}

func (r *Repository) Get(name string) (*Skill, bool) { ... }
func (r *Repository) List() []*Skill { ... }
```

```go
// ========== skill/manager.go ==========

// Manager 管理每个 Session 的 Skill 加载状态
type Manager struct {
    repo      *Repository
    maxLoaded int
    mu        sync.RWMutex
    loaded    map[string]map[string]*Skill   // sessionID → {skillName → Skill}
}

func (m *Manager) Load(sessionID, skillName string) error {
    skill, ok := m.repo.Get(skillName)
    if !ok { return fmt.Errorf("skill %q not found", skillName) }
    
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if m.loaded[sessionID] == nil {
        m.loaded[sessionID] = make(map[string]*Skill)
    }
    
    // LRU eviction if exceeds maxLoaded
    if len(m.loaded[sessionID]) >= m.maxLoaded {
        // 删除最早加载的
        ...
    }
    
    m.loaded[sessionID][skillName] = skill
    return nil
}

func (m *Manager) Unload(sessionID, skillName string) { ... }

// GetInstruction 返回当前 session 所有加载 Skill 的指令合并
func (m *Manager) GetInstruction(sessionID string) string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    var parts []string
    for _, skill := range m.loaded[sessionID] {
        parts = append(parts, "## Skill: "+skill.Name+"\n"+skill.Instruction)
    }
    return strings.Join(parts, "\n\n")
}
```

```go
// ========== skill/toolset.go ==========

// SkillToolset 暴露给 LLM 的 skill 管理工具
type SkillToolset struct {
    manager *Manager
}

// Tools 返回 skill_load / skill_unload / skill_list 三个 adk.Tool
func (s *SkillToolset) Tools(sessionID string) []adk.Tool {
    return []adk.Tool{
        &skillLoadTool{mgr: s.manager, sessionID: sessionID},
        &skillUnloadTool{mgr: s.manager, sessionID: sessionID},
        &skillListTool{mgr: s.manager, sessionID: sessionID},
    }
}
```

### 4.7 AG-UI Translator — Stateful Protocol Conversion

```go
// ========== agui/translator.go ==========

// Translator 将 ADK Event 流转换为 AG-UI SSE 事件
type Translator struct {
    // 流式状态
    isStreaming   bool
    isThinking    bool
    messageID     string
    
    // 工具调用追踪
    activeToolCalls map[string]bool
    
    // 文本过滤
    filters       []TextStreamFilter
    filterBuffer  string
    filterActive  bool
}

func NewTranslator(filters ...TextStreamFilter) *Translator {
    return &Translator{
        messageID:       "msg-" + uuid.New().String()[:8],
        activeToolCalls: make(map[string]bool),
        filters:         filters,
    }
}

// Translate 将单个 ADK Event 转换为 0~N 个 AG-UI 事件
func (t *Translator) Translate(event adk.Event) []AGUIEvent {
    var result []AGUIEvent
    
    if event.Content == nil {
        return result
    }
    
    for _, part := range event.Content.Parts {
        switch p := part.(type) {
        case adk.TextPart:
            result = append(result, t.handleText(p, event)...)
        case adk.ThinkingPart:
            result = append(result, t.handleThinking(p, event)...)
        case adk.ToolCallPart:
            result = append(result, t.handleToolCall(p)...)
        case adk.ToolResultPart:
            // ToolResult 不直接推给前端
        }
    }
    
    // StateDelta → 自定义事件
    if event.Actions != nil && event.Actions.StateDelta != nil {
        result = append(result, t.handleStateDelta(event.Actions.StateDelta)...)
    }
    
    // Final event → 关闭流
    if event.Final {
        if t.isStreaming {
            result = append(result, AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: t.messageID})
            t.isStreaming = false
        }
    }
    
    return result
}

func (t *Translator) handleText(p adk.TextPart, event adk.Event) []AGUIEvent {
    var result []AGUIEvent
    text := p.Text
    
    // 应用 TextStreamFilter
    if len(t.filters) > 0 {
        text, result = t.applyFilters(text, event.Author)
        if text == "" {
            return result
        }
    }
    
    // 开始文本流
    if !t.isStreaming {
        t.isStreaming = true
        result = append(result, AGUIEvent{Type: "TEXT_MESSAGE_START", MessageID: t.messageID, Role: "assistant"})
    }
    
    result = append(result, AGUIEvent{Type: "TEXT_MESSAGE_CONTENT", MessageID: t.messageID, Content: text})
    return result
}

func (t *Translator) handleToolCall(p adk.ToolCallPart) []AGUIEvent {
    // 如果当前在文本流中，先关闭
    var result []AGUIEvent
    if t.isStreaming {
        result = append(result, AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: t.messageID})
        t.isStreaming = false
    }
    
    toolCallID := p.ID
    t.activeToolCalls[toolCallID] = true
    
    result = append(result,
        AGUIEvent{Type: "TOOL_CALL_START", ToolCallID: toolCallID, ToolName: p.Name, MessageID: t.messageID},
        AGUIEvent{Type: "TOOL_CALL_ARGS", ToolCallID: toolCallID, Content: string(p.Arguments)},
        AGUIEvent{Type: "TOOL_CALL_END", ToolCallID: toolCallID},
    )
    return result
}
```

```go
// ========== agui/filter.go ==========

type FilterDecision int
const (
    FilterPass    FilterDecision = iota  // 放行文本
    FilterDrop                            // 丢弃文本
    FilterPending                         // 继续缓冲，还无法判断
)

// TextStreamFilter 在文本推送前进行缓冲评估
type TextStreamFilter interface {
    // Match 判断该 filter 是否对当前消息生效
    Match(author string, isThought bool) bool
    // Evaluate 评估缓冲内容
    Evaluate(buffered string) FilterDecision
    // MaxBuffer 缓冲上限（字节），超过强制 Pass
    MaxBuffer() int
}

func (t *Translator) applyFilters(text string, author string) (string, []AGUIEvent) {
    // 找到匹配的 filter
    var activeFilter TextStreamFilter
    for _, f := range t.filters {
        if f.Match(author, t.isThinking) {
            activeFilter = f
            break
        }
    }
    if activeFilter == nil {
        return text, nil  // 无 filter，直接放行
    }
    
    // 缓冲模式
    t.filterBuffer += text
    
    // 超过缓冲上限，强制放行
    if len(t.filterBuffer) > activeFilter.MaxBuffer() {
        result := t.filterBuffer
        t.filterBuffer = ""
        return result, nil
    }
    
    switch activeFilter.Evaluate(t.filterBuffer) {
    case FilterPass:
        result := t.filterBuffer
        t.filterBuffer = ""
        return result, nil
    case FilterDrop:
        t.filterBuffer = ""
        return "", nil
    case FilterPending:
        return "", nil  // 继续缓冲，不输出
    }
    return text, nil
}
```

**TextStreamFilter 工作原理示意：**

```
LLM 输出 chunk: "{"
    → Match=true, 进入缓冲模式（不立即推 SSE）
    → Evaluate("{") → Pending（还不完整）

LLM 输出 chunk: "\"agui_activity\":"
    → filterBuffer = "{\"agui_activity\":"
    → Evaluate → Drop（匹配内部协议）
    → 清空缓冲区

LLM 输出 chunk: "你好，请问"
    → Evaluate("你好，请问") → Pass（不是 JSON）
    → 补发 TEXT_MESSAGE_START + TEXT_MESSAGE_CONTENT
    → 后续 chunk 正常实时输出
```

```go
// ========== agui/events.go ==========

type AGUIEvent struct {
    Type       string `json:"type"`
    RunID      string `json:"runId,omitempty"`
    MessageID  string `json:"messageId,omitempty"`
    Role       string `json:"role,omitempty"`
    Content    string `json:"content,omitempty"`
    ToolCallID string `json:"toolCallId,omitempty"`
    ToolName   string `json:"toolName,omitempty"`
    Error      string `json:"error,omitempty"`
    AgentName  string `json:"agentName,omitempty"`
    Metadata   map[string]any `json:"metadata,omitempty"`
}
```

### 4.8 Launcher — One-Line Service Assembly

```go
// ========== launcher/launcher.go ==========

type Launcher struct {
    engine     *gin.Engine
    config     *RuntimeConfig
    session    adk.SessionService
    
    // 可选组件
    filters     []agui.TextStreamFilter
}

type Option func(*Launcher)

func WithTextStreamFilter(f ...agui.TextStreamFilter) Option { ... }
func WithSessionService(s adk.SessionService) Option { ... }
func WithMiddleware(m ...gin.HandlerFunc) Option { ... }

func New(cfg *RuntimeConfig, opts ...Option) (*Launcher, error) {
    l := &Launcher{
        engine: gin.New(),
        config: cfg,
    }
    for _, opt := range opts { opt(l) }
    
    // 初始化 Session Service
    if l.session == nil {
        switch cfg.Session.Backend {
        case "mysql":
            l.session, _ = session.NewMySQLSessionService(os.Getenv(cfg.Session.MySQL.DSNEnv))
        default:
            l.session = session.NewMemorySessionService()
        }
    }
    
    return l, nil
}

// Start 按注册的服务类型启动路由
func (l *Launcher) Start(ctx context.Context, addr string, services ...string) error {
    for _, svc := range services {
        switch svc {
        case "agui":
            l.registerAGUIRoutes()    // POST /agui/runs (SSE)
        case "a2a":
            l.registerA2ARoutes()     // /.well-known/agent.json + POST /
        case "api":
            l.registerAPIRoutes()     // /api/sessions, /api/agents
        case "health":
            l.registerHealthRoutes()  // /health
        }
    }
    return l.engine.Run(addr)
}
```

### 4.9 configuredAgentFactory — YAML to Running Agent

```go
// ========== registry/agent_factory.go ==========

// AgentYAMLConfig 对应 YAML 中一个 agent 节点
type AgentYAMLConfig struct {
    Name           string         `yaml:"name"`
    Model          string         `yaml:"model"`          // 引用 models 中的名字
    Instruction    string         `yaml:"instruction"`
    Tools          []string       `yaml:"tools"`          // "type/set/func" 路径
    SubAgents      []string       `yaml:"subagents"`      // Agent 名字（本地或 a2a/远程）
    Skills         []string       `yaml:"skills"`         // 初始加载的 Skill 名
    Streaming      bool           `yaml:"streaming"`
    MaxIterations  int            `yaml:"max_iterations"`
    ContextPruning *PruningConfig `yaml:"context_pruning"`
}

type PruningConfig struct {
    Strategy        string `yaml:"strategy"`
    KeepFirst       int    `yaml:"keep_first"`
    KeepLast        int    `yaml:"keep_last"`
    MaxToolResultLen int   `yaml:"max_tool_result_len"`
}

// configuredAgentFactory 从 YAML 配置自动构建 Agent
type configuredAgentFactory struct {
    cfg AgentYAMLConfig
}

func (f *configuredAgentFactory) NewAgent(ctx context.Context) (adk.Agent, error) {
    // 1. 解析 Model
    model, err := ResolveModel(ctx, f.cfg.Model)
    if err != nil { return nil, fmt.Errorf("model %q: %w", f.cfg.Model, err) }
    
    // 2. 解析 Tools
    var tools []adk.Tool
    for _, toolPath := range f.cfg.Tools {
        t, err := ResolveTool(ctx, toolPath)
        if err != nil { return nil, fmt.Errorf("tool %q: %w", toolPath, err) }
        tools = append(tools, t...)
    }
    
    // 3. 解析 SubAgents
    var subAgents []adk.Agent
    for _, saName := range f.cfg.SubAgents {
        sa, err := resolveSubAgent(ctx, saName)
        if err != nil { return nil, fmt.Errorf("subagent %q: %w", saName, err) }
        subAgents = append(subAgents, sa)
    }
    
    // 4. 构建 LLM Agent
    agent := adk.NewLLMAgent(adk.LLMAgentConfig{
        Name:        f.cfg.Name,
        Model:       model,
        Instruction: f.cfg.Instruction,
        Tools:       tools,
        SubAgents:   subAgents,
    })
    
    return agent, nil
}

// resolveSubAgent 支持本地和远程 Agent
func resolveSubAgent(ctx context.Context, name string) (adk.Agent, error) {
    // 本地 Agent：从注册表递归构建
    if factory, ok := GetAgentFactory(name); ok {
        return factory.NewAgent(ctx)
    }
    // 远程 Agent（a2a:// 前缀）：构建 RemoteAgent
    if strings.HasPrefix(name, "a2a://") {
        url := strings.TrimPrefix(name, "a2a://")
        return a2a.NewRemoteAgent(name, url), nil
    }
    return nil, fmt.Errorf("agent %q not found in registry", name)
}
```

---

## 5. Services Layer — Gateway, Orchestrator, Child Agents

### 5.1 Communication Topology

```
                    AG-UI/SSE                     gRPC (internal)
┌──────────┐  ←─────────────────→  ┌──────────┐  ←─────────────→  ┌──────────────┐
│ Frontend │                       │ Gateway  │                    │ Orchestrator │
└──────────┘                       └──────────┘                    └──────┬───────┘
                                                                          │
                                                         A2A/JSON-RPC     │  A2A/JSON-RPC
                                                    ┌─────────────────────┼─────────────────────┐
                                                    │                     │                     │
                                              ┌─────▼─────┐        ┌─────▼─────┐        ┌─────▼─────┐
                                              │code-agent │        │ web-agent │        │review-agnt│
                                              └───────────┘        └───────────┘        └───────────┘
```

**通信协议选型：**

| 链路 | 协议 | 原因 |
|------|------|------|
| Frontend ↔ Gateway | AG-UI SSE (HTTP) | 前端友好，实时流式 |
| Gateway ↔ Orchestrator | gRPC (双向 streaming) | 内部服务，强类型，流式传输高效 |
| Orchestrator ↔ Child Agent | A2A JSON-RPC (HTTP streaming) | 标准 Agent 互操作协议，Child Agent 可独立部署 |

### 5.2 Gateway Service

Gateway 是前端唯一入口，**只做协议转换和持久化，不做编排决策**。

```go
// ========== services/gateway/cmd/main.go ==========

func main() {
    cfg := gateway.LoadConfig()   // 环境变量 + 本地 YAML
    
    db := store.NewMySQL(cfg.DatabaseURL)
    orchClient := orchestrator.NewGRPCClient(cfg.OrchestratorAddr)
    
    srv := gateway.NewServer(gateway.Deps{
        Store:        db,
        Orchestrator: orchClient,
        Config:       cfg,
    })
    
    srv.Run(":8080")
}
```

**核心 Handler — AG-UI Run：**

```go
// ========== internal/handler/agui.go ==========

func (h *Handler) HandleAGUIRun(c *gin.Context) {
    var req AGUIRunRequest
    c.ShouldBindJSON(&req)
    
    // 1. 验证 conversation 存在
    if !h.store.ConversationExists(req.ThreadID) {
        c.JSON(404, gin.H{"error": "conversation not found"})
        return
    }
    
    // 2. 保存用户消息
    h.store.SaveMessage(req.ThreadID, "user", lastMessage(req))
    
    // 3. 加载历史消息（给 Orchestrator 做上下文）
    history := h.store.GetMessages(req.ThreadID, 50)
    
    // 4. SSE 头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    
    // 5. 调用 Orchestrator (gRPC streaming)
    stream, _ := h.orchestrator.Execute(c.Request.Context(), &pb.ExecuteRequest{
        ThreadID:  req.ThreadID,
        Messages:  convertMessages(req.Messages),
        History:   convertHistory(history),
        Tools:     convertTools(req.Tools),   // 前端声明的 Skill/能力
        RunID:     req.RunID,
    })
    
    // 6. gRPC stream → SSE
    var agentText string
    var artifacts []store.ArtifactData
    
    c.Stream(func(w io.Writer) bool {
        event, err := stream.Recv()
        if err == io.EOF { return false }
        if err != nil {
            writeSSE(w, AGUIEvent{Type: "RUN_ERROR", Error: "orchestration error"})
            return false
        }
        
        // 透传 AG-UI 事件
        aguiEvent := convertProtoToAGUI(event)
        
        // 累积 agent 回复用于持久化
        if aguiEvent.Type == "TEXT_MESSAGE_CONTENT" {
            agentText += aguiEvent.Content
        }
        if aguiEvent.Type == "TOOL_CALL_ARGS" {
            artifacts = append(artifacts, parseArtifact(aguiEvent))
        }
        
        writeSSE(w, aguiEvent)
        c.Writer.Flush()
        return true
    })
    
    // 7. 流结束后持久化 agent 回复
    if agentText != "" || len(artifacts) > 0 {
        h.store.SaveMessage(req.ThreadID, "agent", agentText, artifacts)
    }
}
```

**Gateway API Surface：**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/agui/runs` | POST | AG-UI run (SSE response) |
| `/api/conversations` | GET | List conversations |
| `/api/conversations` | POST | Create conversation |
| `/api/conversations/:id` | DELETE | Delete conversation |
| `/api/conversations/:id/messages` | GET | Get message history |
| `/api/agents` | GET | List available agents (proxied from Orchestrator) |
| `/health` | GET | Health check |

### 5.3 Orchestrator Service

Orchestrator 是编排大脑，接收 Gateway 的 gRPC 请求，决定调哪些 Agent、用什么策略。

**gRPC Proto 定义：**

```protobuf
// ========== proto/orchestrator.proto ==========

syntax = "proto3";
package orchestrator;

service OrchestratorService {
    // Execute 接收 Gateway 请求，流式返回 AG-UI 事件
    rpc Execute(ExecuteRequest) returns (stream AGUIEvent);
    
    // ListAgents 返回可用 Agent 列表
    rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
}

message ExecuteRequest {
    string thread_id = 1;
    string run_id = 2;
    repeated Message messages = 3;      // 当前轮消息
    repeated Message history = 4;       // 历史上下文
    repeated Tool tools = 5;            // 前端声明的能力
    map<string, string> metadata = 6;   // 扩展字段（user_id 等）
}

message Message {
    string role = 1;
    string content = 2;
    string sender_name = 3;
}

message Tool {
    string name = 1;
    string description = 2;
    string schema_json = 3;
}

message AGUIEvent {
    string type = 1;
    string run_id = 2;
    string message_id = 3;
    string content = 4;
    string tool_call_id = 5;
    string tool_name = 6;
    string role = 7;
    string error = 8;
    string agent_name = 9;              // 多 Agent 时标识来源
    map<string, string> metadata = 10;
}

message ListAgentsRequest {}
message ListAgentsResponse {
    repeated AgentInfo agents = 1;
}
message AgentInfo {
    string name = 1;
    string description = 2;
    repeated string skills = 3;
    string url = 4;
    bool streaming = 5;
}
```

**Planner — Intent Recognition + Orchestration Plan：**

```go
// ========== internal/planner/planner.go ==========

// PlanningMode 编排模式
type PlanningMode string
const (
    ModeSingle     PlanningMode = "single"       // 单 Agent
    ModeSequential PlanningMode = "sequential"   // 顺序多 Agent
    ModeParallel   PlanningMode = "parallel"     // 并行多 Agent
)

// OrchestrationPlan 编排计划
type OrchestrationPlan struct {
    Mode   PlanningMode
    Steps  []PlanStep
}

type PlanStep struct {
    AgentName string
    Input     string                // 给该 Agent 的输入（可从上一步结果派生）
    DependsOn []string              // 依赖的前置 step（并行时为空）
}

// Planner 根据用户消息决定编排策略
type Planner struct {
    agents   []router.AgentInfo    // 可用 Agent 列表
    llm      adk.Model              // 用 LLM 做意图识别（可选）
}

// Plan 生成编排计划
func (p *Planner) Plan(ctx context.Context, messages []Message, history []Message) (*OrchestrationPlan, error) {
    // MVP 策略：基于关键词 + Agent Skill 匹配
    // 后续演进：用 LLM 做 structured output 生成计划
    
    matchedAgents := p.matchAgents(messages)
    
    switch len(matchedAgents) {
    case 0:
        return nil, fmt.Errorf("no agent can handle this request")
    case 1:
        return &OrchestrationPlan{
            Mode:  ModeSingle,
            Steps: []PlanStep{{AgentName: matchedAgents[0]}},
        }, nil
    default:
        // 多 Agent：判断是否可并行
        if p.canParallel(matchedAgents, messages) {
            return &OrchestrationPlan{Mode: ModeParallel, Steps: ...}, nil
        }
        return &OrchestrationPlan{Mode: ModeSequential, Steps: ...}, nil
    }
}
```

**Executor — Plan Execution Engine：**

```go
// ========== internal/executor/executor.go ==========

type Executor struct {
    a2aClient   *a2a.Client        // A2A 客户端（调 Child Agent）
    agentRouter *router.Router
    converter   *converter.A2AToAGUI
}

// Execute 执行编排计划，流式返回 AG-UI 事件
func (e *Executor) Execute(ctx context.Context, plan *planner.OrchestrationPlan, req *pb.ExecuteRequest, eventCh chan<- *pb.AGUIEvent) error {
    switch plan.Mode {
    case planner.ModeSingle:
        return e.executeSingle(ctx, plan.Steps[0], req, eventCh)
    case planner.ModeSequential:
        return e.executeSequential(ctx, plan.Steps, req, eventCh)
    case planner.ModeParallel:
        return e.executeParallel(ctx, plan.Steps, req, eventCh)
    }
    return nil
}

func (e *Executor) executeSingle(ctx context.Context, step planner.PlanStep, req *pb.ExecuteRequest, eventCh chan<- *pb.AGUIEvent) error {
    agentURL := e.agentRouter.GetURL(step.AgentName)
    
    // 构建结构化消息
    messages := buildA2AMessages(req)
    
    // 调用 Child Agent via A2A
    eventIter, err := e.a2aClient.SendStreamingMessages(ctx, agentURL, messages)
    if err != nil { return err }
    
    // 转换 A2A → AG-UI 并推送
    for a2aEvent, err := range eventIter {
        if err != nil { return err }
        
        aguiEvents := e.converter.Convert(a2aEvent, step.AgentName)
        for _, ev := range aguiEvents {
            eventCh <- ev
        }
    }
    return nil
}

func (e *Executor) executeSequential(ctx context.Context, steps []planner.PlanStep, req *pb.ExecuteRequest, eventCh chan<- *pb.AGUIEvent) error {
    var lastOutput string
    for _, step := range steps {
        // 后续 step 使用前一步的输出作为输入
        if lastOutput != "" {
            step.Input = lastOutput
        }
        // ... executeSingle + capture output
    }
    return nil
}

func (e *Executor) executeParallel(ctx context.Context, steps []planner.PlanStep, req *pb.ExecuteRequest, eventCh chan<- *pb.AGUIEvent) error {
    var wg sync.WaitGroup
    
    for _, step := range steps {
        wg.Add(1)
        go func(s planner.PlanStep) {
            defer wg.Done()
            e.executeSingle(ctx, s, req, eventCh)  // eventCh 是线程安全的
        }(step)
    }
    
    wg.Wait()
    return nil
}
```

### 5.4 Child Agent Service

每个 Child Agent 是完全独立的服务，只依赖 `pkg/adk`，通过 A2A 协议对外暴露。

**Mode A — Pure ADK (lightweight, for simple agents)：**

```go
// ========== services/agents/code-agent/cmd/main.go ==========
func main() {
    // 1. 加载身份配置
    cfg := adk.LoadAgentConfig("config.yaml")
    
    // 2. 创建 LLM Model（直接用 runtime/model 或自建）
    model := createModel()
    
    // 3. 构建 LLM Agent
    agent := adk.NewLLMAgent(adk.LLMAgentConfig{
        Name:        cfg.Name,
        Model:       model,
        Instruction: systemPrompt,
        Tools:       []adk.Tool{&generateCodeTool{}, &reviewCodeTool{}},
    })
    
    // 4. 创建 Runner + Session
    session := adk.NewMemorySessionService()  // ADK 内置内存实现
    runner := adk.NewRunner(agent, session, adk.WithMaxIterations(25))
    
    // 5. 暴露为 A2A Server
    a2aServer := a2a.NewServer(cfg, runner)
    a2aServer.Run(":8081")
}
```

**Mode B — Full Runtime (for complex agents with skills, pruning, etc.)：**

```go
// ========== cmd/main.go (Mode B) ==========
func main() {
    // 使用 Runtime 的完整能力（YAML 配置、Skill、Context Pruning）
    cfg, _ := runtime.LoadConfig("config.yaml")
    cfg.Setup(context.Background())
    
    launcher, _ := launcher.New(cfg,
        launcher.WithTextStreamFilter(&internalJSONFilter{}),
    )
    
    launcher.Start(context.Background(), ":8081", "a2a", "health")
}
```

**Agent-specific tool example：**

```go
// ========== tools.go ==========

type generateCodeTool struct{}

func (t *generateCodeTool) Name() string        { return "generate_code" }
func (t *generateCodeTool) Description() string { return "Generate code based on requirements" }
func (t *generateCodeTool) Schema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "language": {"type": "string", "description": "Programming language"},
            "requirements": {"type": "string", "description": "What the code should do"}
        },
        "required": ["language", "requirements"]
    }`)
}

func (t *generateCodeTool) Execute(ctx context.Context, args json.RawMessage) (*adk.ToolResult, error) {
    var input struct {
        Language     string `json:"language"`
        Requirements string `json:"requirements"`
    }
    json.Unmarshal(args, &input)
    
    // 实际的代码生成逻辑
    code := generateCode(input.Language, input.Requirements)
    
    return &adk.ToolResult{Content: code}, nil
}
```

### 5.5 Service Dependency Graph

```
pkg/adk           ← 零外部依赖（只有标准库 + a2a-go）
    ↑
pkg/runtime       ← import adk
    ↑
services/gateway       ← import runtime（用 AG-UI Translator）
services/orchestrator  ← import adk + runtime（用 A2A Client + Model）
services/agents/*      ← import adk（轻量模式）或 adk + runtime（完整模式）
```

**go.work（本地开发）：**

```
go 1.22

use (
    ./pkg/adk
    ./pkg/runtime
    ./services/gateway
    ./services/orchestrator
    ./services/agents/code-agent
    ./services/agents/web-agent
)
```

---

## 6. Frontend

### 6.1 Key Changes

Frontend 已经是独立 npm 项目，重构后只需调整 AG-UI 事件处理来适配多 Agent 场景：

- Add `agentName` field handling in event processing
- Render different avatars/labels per agent
- No structural changes to SSE client logic

**事件类型更新：**

```typescript
interface AGUIEvent {
    type: AGUIEventType;
    runId?: string;
    messageId?: string;
    role?: string;
    content?: string;
    toolCallId?: string;
    toolName?: string;
    error?: string;
    agentName?: string;        // 新增：标识产出该事件的 Agent
    metadata?: Record<string, any>;
}
```

### 6.2 Interface Contract

Frontend communicates ONLY with Gateway:

| Endpoint | Direction | Description |
|----------|-----------|-------------|
| `POST /agui/runs` | FE → Gateway | Start conversation (SSE response) |
| `GET /api/conversations` | FE → Gateway | List conversations |
| `POST /api/conversations` | FE → Gateway | Create conversation |
| `GET /api/conversations/:id/messages` | FE → Gateway | Message history |
| `GET /api/agents` | FE → Gateway | Available agents list |

---

## 7. End-to-End Data Flow (Complete Request)

```
用户输入: "帮我写一个 Go HTTP server"
    │
    ▼ Frontend
    POST /agui/runs {threadId: "abc", messages: [{role: "user", content: "帮我写一个 Go HTTP server"}]}
    │
    ▼ Gateway (handler/agui.go)
    │  ├─ store.SaveMessage("abc", "user", "帮我写一个 Go HTTP server")
    │  ├─ history = store.GetMessages("abc", 50)
    │  └─ orchestrator.Execute(gRPC stream) ──────────────────────────────┐
    │                                                                      │
    │  ▼ Orchestrator (server/grpc.go)                                    │
    │  │  ├─ planner.Plan(messages, history)                              │
    │  │  │    → OrchestrationPlan{Mode: Single, Steps: [{code-agent}]}   │
    │  │  │                                                                │
    │  │  └─ executor.ExecuteSingle("code-agent", ...)                    │
    │  │       │                                                           │
    │  │       ▼ A2A Client → code-agent:8081                             │
    │  │                                                                   │
    │  │  ▼ Code-Agent (ADK Runner)                                       │
    │  │  │  iteration 1:                                                  │
    │  │  │    ├─ BeforeGenerate: PruningPlugin 裁剪历史                    │
    │  │  │    ├─ agent.Generate() → LLM 返回 ToolCall[generate_code]     │
    │  │  │    ├─ AfterGenerate: 无                                        │
    │  │  │    ├─ BeforeTool: 记录 span                                   │
    │  │  │    ├─ tool.Execute(generate_code, {lang:"go",...})            │
    │  │  │    ├─ AfterTool: 记录结果                                      │
    │  │  │    └─ yield Event{ToolCall} + Event{ToolResult}               │
    │  │  │                                                                │
    │  │  │  iteration 2:                                                  │
    │  │  │    ├─ agent.Generate() → LLM 返回 Text（代码解释 + 代码块）    │
    │  │  │    └─ yield Event{Text, Final=true}                           │
    │  │  │                                                                │
    │  │  │  A2A 事件流:                                                   │
    │  │  │    Task(Submitted) → Status(Working) → Artifact(text chunks)  │
    │  │  │    → Artifact(code) → Status(Completed)                       │
    │  │  │                                                                │
    │  │  ▼ Orchestrator converter                                        │
    │  │     A2A Events → AG-UI Events:                                   │
    │  │       Working        → TEXT_MESSAGE_START                         │
    │  │       Artifact(text) → TEXT_MESSAGE_CONTENT × N                  │
    │  │       Artifact(code) → TOOL_CALL_START/ARGS/END (code_preview)   │
    │  │       Completed      → TEXT_MESSAGE_END + RUN_FINISHED           │
    │  │                                                                   │
    │  │  gRPC stream.Send(aguiEvent) ←────────────────────────────────────┘
    │  │
    ▼ Gateway
    │  SSE write: "data: {\"type\":\"TEXT_MESSAGE_START\",...}\n\n"
    │  SSE write: "data: {\"type\":\"TEXT_MESSAGE_CONTENT\",\"content\":\"我来帮你...\"}\n\n"
    │  ...
    │  SSE write: "data: {\"type\":\"TOOL_CALL_START\",\"toolName\":\"code_preview\"}\n\n"
    │  SSE write: "data: {\"type\":\"TOOL_CALL_ARGS\",\"content\":\"{...code...}\"}\n\n"
    │  SSE write: "data: {\"type\":\"RUN_FINISHED\"}\n\n"
    │  │
    │  └─ store.SaveMessage("abc", "agent", agentText, artifacts)
    │
    ▼ Frontend
    ├─ TEXT_MESSAGE_START → 创建消息气泡，开始流式渲染
    ├─ TEXT_MESSAGE_CONTENT × N → 逐字追加显示
    ├─ TOOL_CALL (code_preview) → 渲染 CodePreview 组件
    └─ RUN_FINISHED → 结束 loading 状态
```

---

## 8. Error Handling & Fallback

```
任意层出错 → 向上传播为 AG-UI RUN_ERROR 事件

Child Agent 错误:
  handler 返回 error → ADK ExecuteHandler → Status(Failed)
  → Orchestrator converter → AGUIEvent{type: "RUN_ERROR", agent_name: "code-agent"}
  → Gateway → SSE: {type: "RUN_ERROR", error: "agent task failed"}
  → Frontend → 显示错误提示

Orchestrator 连接 Agent 失败:
  A2A Client error → Executor fallback:
    1. 重试 1 次（可配置）
    2. 仍失败 → 返回 RUN_ERROR 到 Gateway

Gateway 连接 Orchestrator 失败:
  gRPC dial error → 返回 HTTP 503 + {error: "orchestration service unavailable"}

错误信息脱敏规则（所有层统一）:
  - 不暴露内部 URL（localhost:8081）
  - 不暴露 API Key / 堆栈信息
  - 只返回用户可理解的描述
```

---

## 9. New Agent Onboarding Process

一个新的 Child Agent 接入只需：

```bash
# 1. 创建目录
mkdir -p services/agents/my-agent

# 2. 写 config.yaml
cat > services/agents/my-agent/config.yaml << 'EOF'
name: my-agent
description: 我的自定义 Agent
version: "0.1.0"
url: "http://localhost:8083"
skills:
  - custom_task
inputModes:
  - text
outputModes:
  - text
streaming: true
EOF

# 3. 写 main.go + handler.go（~50行）
# 4. 在 Orchestrator config 中注册
# 5. docker-compose 加一个 service
```

无需改 Gateway、Frontend、ADK、Runtime 的任何代码。

---

## 10. Docker Compose

```yaml
services:
  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    depends_on: [gateway]
    environment:
      VITE_API_URL: http://localhost:8080

  gateway:
    build: ./services/gateway
    ports: ["8080:8080"]
    depends_on:
      mysql: { condition: service_healthy }
      orchestrator: { condition: service_started }
    environment:
      DATABASE_URL: root:${MYSQL_PASSWORD}@tcp(mysql:3306)/agenthub?parseTime=true
      ORCHESTRATOR_ADDR: orchestrator:9090
      AGENTHUB_API_TOKEN: ${AGENTHUB_API_TOKEN}

  orchestrator:
    build: ./services/orchestrator
    ports: ["9090:9090"]
    environment:
      AGENT_CODE_URL: http://code-agent:8081
      AGENT_WEB_URL: http://web-agent:8082
      GRPC_PORT: 9090

  code-agent:
    build:
      context: .
      dockerfile: ./services/agents/code-agent/Dockerfile
    ports: ["8081:8081"]
    environment:
      LLM_PROVIDER: anthropic
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      ANTHROPIC_MODEL: claude-sonnet-4-20250514

  web-agent:
    build:
      context: .
      dockerfile: ./services/agents/web-agent/Dockerfile
    ports: ["8082:8082"]
    environment:
      LLM_PROVIDER: openai
      OPENAI_API_KEY: ${OPENAI_API_KEY}

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_PASSWORD}
      MYSQL_DATABASE: agenthub
    volumes:
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
      - mysqldata:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      retries: 10

volumes:
  mysqldata:
```

---

## 11. Implementation Phases

| Phase | Scope | Deliverable |
|-------|-------|-------------|
| **Phase 1** | `pkg/adk` core | Agent, Model, Tool, Plugin interfaces + Runner + Event types + Session (memory) + unit tests |
| **Phase 2** | `pkg/adk/a2a` | A2A Server + Client + RemoteAgent + AgentCard + integration tests |
| **Phase 3** | `pkg/runtime` registry + model | Config loader + Model Registry + Anthropic/OpenAI/Proxy providers + Tool Registry |
| **Phase 4** | `pkg/runtime` session + context | MySQL Session Service + ChainPruner + PruningPlugin |
| **Phase 5** | `pkg/runtime` skill + agui + launcher | Skill system + AG-UI Translator + TextStreamFilter + Launcher |
| **Phase 6** | `services/orchestrator` | gRPC server + Planner + Executor (single/sequential/parallel) + A2A→AG-UI converter |
| **Phase 7** | `services/gateway` | Gin HTTP + AG-UI SSE handler + gRPC client + Conversation store |
| **Phase 8** | `services/agents/code-agent` | First Child Agent with full ADK+Runner integration |
| **Phase 9** | `frontend` updates | Multi-agent support (agentName) + updated event handling |
| **Phase 10** | Integration + Docker | docker-compose.yml + Makefile + end-to-end smoke test |

---

## 12. Key Design Decisions Summary

| Decision | Rationale |
|----------|-----------|
| Monorepo multi-module (go.work) | Single repo for convenience, separate go.mod for true isolation |
| Clean rewrite over incremental refactoring | Current coupling too deep, cleaner to start fresh |
| ADK as pure interface layer | Enables swapping Runtime implementations without touching engine |
| Runner with iter.Seq2 | Go 1.22+ range-over-func for natural streaming |
| gRPC between Gateway and Orchestrator | Efficient streaming, strong types, internal service communication |
| A2A for Child Agents | Standard protocol, agents can be any language/framework |
| TextStreamFilter with buffered eval | LLM outputs character-by-character, need buffering to detect patterns |
| EnsurePairedFunctionCalls post-pruning | Pruning can break ToolCall/ToolResult pairs, causing LLM errors |
| init() registration pattern | Zero-config: import a package = register its capabilities |
| Setup() strict ordering | Agent construction depends on Models+Tools being ready first |
| Session = Event stream + State KV | Replay-friendly, supports cross-turn state passing |
| Two Child Agent modes (pure ADK / full Runtime) | Simple agents don't need full framework overhead |
