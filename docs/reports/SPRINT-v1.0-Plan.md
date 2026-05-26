# AgentHub v1.0 Sprint Plan（7天冲刺计划）

## 文档信息

| 项目 | 内容 |
|------|------|
| 项目名称 | AgentHub v1.0 Sprint |
| 前置版本 | MVP v0.1（已完成） |
| 冲刺周期 | **7 个工作日** |
| 团队规模 | **2 人**（A: 后端/Agent，B: 前端/集成） |
| 目标 | 多Agent协作 + LLM意图编排 + 群聊模式 + 丰富产物预览 |
| 关联文档 | PDR-AgentHub多Agent协作平台.md |

---

## 一、目标与考察对齐

### 1.1 课题考察要点

| 维度 | 权重 | 对齐策略 |
|------|:----:|----------|
| AI协作能力 | 30% | 维护 Spec/Skill/Contract 文档链路，展示 AI 辅助开发过程 |
| 功能完整度 | 25% | 多Agent调度跑通 + IM核心体验流畅 |
| 生成效果质量 | 20% | 产物预览效果好（代码高亮+网页iframe+Markdown渲染） |
| 代码理解度 | 15% | 清晰解释 A2A↔AG-UI 转换、意图编排逻辑 |
| 创新与产品感 | 10% | 群聊多Agent协作、编排过程可视化 |

### 1.2 MVP → v1.0 功能增量

| 功能 | MVP 状态 | v1.0 目标 |
|------|:--------:|:---------:|
| Agent 数量 | 1 (code-agent) | **2+** (code + web) |
| 意图编排 | 硬编码路由 | **LLM 驱动** |
| 对话模式 | 单聊 | 单聊 + **群聊** |
| 产物类型 | code_preview | code + **web_preview** + **markdown** |
| Agent 发现 | 环境变量硬编码 | **AgentCard 注册表** + 健康检查 |
| 上下文传递 | 纯文本拼接 | **结构化多轮消息** |
| 错误处理 | 基础 | **降级 + 重试提示** |
| 执行策略 | single | single + **parallel** |

### 1.3 交付物清单

- [ ] 可运行 Demo（docker-compose 一键启动）
- [ ] 3 分钟 Demo 视频脚本 + 录制
- [ ] 更新后的技术文档（README + 架构图）
- [ ] AI 协作开发记录（Spec/Skills 文档）

---

## 二、人员分工

| 角色 | 代号 | 职责范围 |
|------|:----:|----------|
| 后端 + Agent | **A** | Agent Registry、LLM意图编排、web-agent实现、并行调度、错误降级、Gateway增强 |
| 前端 + 集成 | **B** | 群聊UI、web_preview/markdown Skill、多Agent消息渲染、SSE修复、Docker集成、Demo |

---

## 三、每日详细任务

---

### Day 1：P0 Bug 修复 + 基础设施升级

> **目标**：消除 MVP 遗留的技术债务，为后续功能开发打好基础。

#### 人员 A（后端）

**任务 1.1：HTTP Client 超时 + LLMClient 单例化**

文件：`agents/adk/llm.go`

```go
// 修改前：使用 http.DefaultClient（无超时）
resp, err := http.DefaultClient.Do(req)

// 修改后：创建带超时的 Client
var llmHTTPClient = &http.Client{
    Timeout: 120 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        10,
        MaxIdleConnsPerHost: 5,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

文件：`agents/code-agent/main.go`

```go
// 修改前：handler 内每次 new
// 修改后：main 中初始化，通过闭包传递
func main() {
    llm := adk.NewLLMClient() // 启动时创建一次
    server := adk.NewA2AServer(config, func(ctx *adk.Context, msgs []a2a.Message) error {
        return handleTask(ctx, msgs, llm) // 注入 LLM client
    })
}
```

**任务 1.2：结构化上下文传递**

文件：`server/internal/orchestrator/orchestrator.go`

```go
// 修改前：buildUserMessage 返回纯文本拼接
// 修改后：返回结构化消息数组

type StructuredMessage struct {
    Role    string `json:"role"`    // "user" | "assistant"
    Content string `json:"content"`
}

func (o *Orchestrator) buildStructuredMessages(req model.AGUIRunRequest, history []model.Message) []StructuredMessage {
    var msgs []StructuredMessage
    for _, h := range history {
        role := "user"
        if h.SenderType == "agent" {
            role = "assistant"
        }
        msgs = append(msgs, StructuredMessage{Role: role, Content: h.Content})
    }
    // 加入当前消息
    for _, m := range req.Messages {
        msgs = append(msgs, StructuredMessage{Role: m.Role, Content: m.Content})
    }
    return msgs
}
```

文件：`server/internal/a2a/client.go`  
修改 `SendStreamingMessage` 接受结构化消息，拼接为带角色标记的文本发送给 A2A Agent。

**任务 1.3：Graceful Shutdown**

文件：`server/cmd/server/main.go`

```go
import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // ... 现有初始化代码 ...

    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: r,
    }

    go func() {
        log.Printf("server starting on :%s", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("shutting down gracefully...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

**任务 1.4：A2A Client 连接复用**

文件：`server/internal/a2a/client.go`

```go
type Client struct {
    clients map[string]*a2aclient.Client // agentURL → cached client
    mu      sync.RWMutex
}

func (c *Client) getOrCreateClient(ctx context.Context, agentURL string) (*a2aclient.Client, error) {
    c.mu.RLock()
    if client, ok := c.clients[agentURL]; ok {
        c.mu.RUnlock()
        return client, nil
    }
    c.mu.RUnlock()

    c.mu.Lock()
    defer c.mu.Unlock()
    // double-check
    if client, ok := c.clients[agentURL]; ok {
        return client, nil
    }

    endpoints := []*a2a.AgentInterface{
        a2a.NewAgentInterface(agentURL, a2a.TransportProtocolJSONRPC),
    }
    client, err := a2aclient.NewFromEndpoints(ctx, endpoints)
    if err != nil {
        return nil, err
    }
    c.clients[agentURL] = client
    return client, nil
}
```

#### 人员 B（前端）

**任务 1.5：修复 SSE 流解析 Bug**

文件：`frontend/src/agui/client.ts`

```typescript
// 修改前：按 \n split（不可靠）
// 修改后：按 \n\n 分隔 event block

while (true) {
  const { done, value } = await reader.read()
  if (done) break

  buffer += decoder.decode(value, { stream: true })

  // SSE events are separated by \n\n
  const blocks = buffer.split('\n\n')
  buffer = blocks.pop() || '' // keep last incomplete block

  for (const block of blocks) {
    const lines = block.split('\n')
    for (const line of lines) {
      if (line.startsWith('data:')) {
        const data = line.slice(5).trim()
        if (!data) continue
        try {
          const event: AGUIEvent = JSON.parse(data)
          onEvent(event)
        } catch {
          // skip malformed JSON
        }
      }
    }
  }
}
```

**任务 1.6：前端 Streaming 状态改为 per-conversation**

文件：`frontend/src/stores/messageStore.ts`

```typescript
interface MessageState {
  messages: Record<string, Message[]>
  streaming: Record<string, boolean>        // 改：per-conversation
  abortControllers: Record<string, AbortController> // 改：per-conversation
  // ...
}
```

**任务 1.7：MessageBubble 支持 Agent 名称显示**

文件：`frontend/src/types/index.ts`

```typescript
export interface Message {
  id: string
  conversationId: string
  senderType: 'user' | 'agent'
  senderName?: string           // 新增：Agent 名称
  content: string
  status: 'sending' | 'streaming' | 'sent' | 'failed'
  codeBlocks?: CodeBlock[]
  webPreview?: WebPreviewData   // 新增
  createdAt: string
}

export interface WebPreviewData {
  html: string
  css?: string
  js?: string
  title: string
}
```

**任务 1.8：添加 React Error Boundary**

文件：`frontend/src/components/ErrorBoundary.tsx`（新建）

```tsx
import { Component, type ReactNode } from 'react'

interface Props { children: ReactNode }
interface State { hasError: boolean; error?: Error }

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex items-center justify-center h-screen">
          <div className="text-center p-8">
            <p className="text-lg text-red-600">Something went wrong</p>
            <p className="text-sm text-gray-500 mt-2">{this.state.error?.message}</p>
            <button onClick={() => window.location.reload()} className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded">
              Reload
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
```

#### Day 1 交付标准

- [ ] `make build-check` 通过
- [ ] 现有单元测试全部通过
- [ ] SSE 流式回复正常工作（手动 curl 验证）
- [ ] 服务可以 graceful shutdown（`docker stop` 不报错）

---

### Day 2：web-agent 实现 + Agent Registry

> **目标**：实现第2个Agent（web-agent），并建立Agent注册发现机制。

#### 人员 A（后端）

**任务 2.1：web-agent 核心实现**

目录结构：
```
agents/
  web-agent/
    main.go
    handler.go
    config.yaml
    Dockerfile
```

文件：`agents/web-agent/config.yaml`

```yaml
name: web-agent
description: Generates web pages with HTML, CSS, and JavaScript. Creates responsive, modern UI components.
version: "0.1.0"
url: "http://localhost:8082"
skills:
  - web_generation
  - ui_design
  - responsive_layout
inputModes:
  - text
outputModes:
  - text
  - webpage
  - code
streaming: true
```

文件：`agents/web-agent/main.go`

```go
package main

import (
    "log"
    "os"
    "github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func main() {
    config, err := adk.LoadConfig("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    port := os.Getenv("WEB_AGENT_PORT")
    if port == "" {
        port = "8082"
    }

    llm := adk.NewLLMClient()
    server := adk.NewA2AServer(config, func(ctx *adk.Context, msgs []a2a.Message) error {
        return handleTask(ctx, msgs, llm)
    })

    log.Printf("Web-Agent [%s] starting on port %s", config.Name, port)
    if err := server.Run(":" + port); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

文件：`agents/web-agent/handler.go`

```go
package main

import (
    "fmt"
    "regexp"
    "strings"

    "github.com/a2aproject/a2a-go/v2/a2a"
    "github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const systemPrompt = `You are a web page generation assistant. When the user asks you to create a web page or UI component:

1. Provide a brief description of what you're building.
2. Output a complete, self-contained HTML file with embedded CSS and JavaScript.
3. Use this format for the output:

` + "```html:index.html" + `
<!DOCTYPE html>
<html>
<head>
  <style>/* CSS here */</style>
</head>
<body>
  <!-- HTML here -->
  <script>/* JS here */</script>
</body>
</html>
` + "```" + `

Design principles:
- Use modern CSS (flexbox, grid, custom properties)
- Make it responsive and visually appealing
- Use clean, readable code
- Include subtle animations where appropriate
- Use a consistent color palette`

func handleTask(ctx *adk.Context, messages []a2a.Message, llm *adk.LLMClient) error {
    llmMsgs := make([]adk.LLMMessage, 0, len(messages))
    for _, msg := range messages {
        role := string(msg.Role)
        content := extractTextFromParts(msg.Parts)
        if content != "" {
            llmMsgs = append(llmMsgs, adk.LLMMessage{Role: role, Content: content})
        }
    }

    if len(llmMsgs) == 0 {
        return fmt.Errorf("no messages to process")
    }

    var fullResponse strings.Builder

    err := llm.StreamCompletion(ctx.Context(), systemPrompt, llmMsgs, func(chunk string) {
        fullResponse.WriteString(chunk)
        ctx.StreamText(chunk)
    })
    if err != nil {
        return fmt.Errorf("web generation failed: %w", err)
    }

    // 解析 HTML 代码块 → webpage artifact
    blocks := parseHTMLBlocks(fullResponse.String())
    for _, block := range blocks {
        ctx.AddArtifact(adk.Artifact{
            Type:    "webpage",
            Title:   block.filename,
            Content: block.html,
            Metadata: map[string]string{
                "language": "html",
                "css":      block.css,
                "js":       block.js,
            },
        })
    }

    // 其他代码块作为 code artifact
    codeBlocks := parseNonHTMLCodeBlocks(fullResponse.String())
    for _, block := range codeBlocks {
        ctx.AddArtifact(adk.Artifact{
            Type:    "code",
            Title:   block.filename,
            Content: block.code,
            Metadata: map[string]string{
                "language": block.language,
            },
        })
    }

    return nil
}

type htmlBlock struct {
    filename string
    html     string
    css      string
    js       string
}

// parseHTMLBlocks 提取 HTML 代码块并拆分 CSS/JS
func parseHTMLBlocks(text string) []htmlBlock {
    re := regexp.MustCompile("(?s)```html(?::([^\\n]+))?\\n(.*?)```")
    matches := re.FindAllStringSubmatch(text, -1)

    var blocks []htmlBlock
    for _, m := range matches {
        filename := strings.TrimSpace(m[1])
        if filename == "" {
            filename = "index.html"
        }
        content := strings.TrimSpace(m[2])

        block := htmlBlock{filename: filename, html: content}

        // 提取 <style> 内容
        styleRe := regexp.MustCompile("(?s)<style[^>]*>(.*?)</style>")
        if sm := styleRe.FindStringSubmatch(content); len(sm) > 1 {
            block.css = strings.TrimSpace(sm[1])
        }

        // 提取 <script> 内容
        scriptRe := regexp.MustCompile("(?s)<script[^>]*>(.*?)</script>")
        if sm := scriptRe.FindStringSubmatch(content); len(sm) > 1 {
            block.js = strings.TrimSpace(sm[1])
        }

        blocks = append(blocks, block)
    }
    return blocks
}

func extractTextFromParts(parts a2a.ContentParts) string {
    var texts []string
    for _, part := range parts {
        if text, ok := part.Content.(a2a.Text); ok {
            texts = append(texts, string(text))
        }
    }
    return strings.Join(texts, "\n")
}
```

文件：`agents/web-agent/Dockerfile`

```dockerfile
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download || true
COPY adk/ ./adk/
COPY web-agent/ ./web-agent/
WORKDIR /app/web-agent
RUN go build -o /web-agent .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /web-agent /web-agent
COPY --from=build /app/web-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8082
CMD ["/web-agent"]
```

**任务 2.2：ADK 支持 webpage artifact 类型**

文件：`agents/adk/types.go`  
在 `ToA2APart()` 中增加 webpage 类型处理：

```go
func (art Artifact) ToA2APart() *a2a.Part {
    part := a2a.NewTextPart(art.Content)
    switch art.Type {
    case "webpage":
        part.MediaType = "text/html"
    case "code":
        part.MediaType = "text/x-" + art.Metadata["language"]
    }
    part.Filename = art.Title
    part.Metadata = map[string]any{
        "type":     art.Type,
        "language": art.Metadata["language"],
        "css":      art.Metadata["css"],
        "js":       art.Metadata["js"],
    }
    return part
}
```

**任务 2.3：Agent Registry 实现**

文件：`server/internal/registry/registry.go`（新建）

```go
package registry

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"
)

// AgentInfo 包含 Agent 的注册信息
type AgentInfo struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    URL         string   `json:"url"`
    Version     string   `json:"version"`
    Skills      []Skill  `json:"skills"`
    OutputModes []string `json:"outputModes"`
    Streaming   bool     `json:"streaming"`
    Healthy     bool     `json:"healthy"`
    LastCheck   time.Time `json:"lastCheck"`
}

type Skill struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

// Registry 管理所有已注册的 Agent
type Registry struct {
    agents map[string]*AgentInfo
    mu     sync.RWMutex
    client *http.Client
}

// New 创建 Registry 实例
func New() *Registry {
    return &Registry{
        agents: make(map[string]*AgentInfo),
        client: &http.Client{Timeout: 5 * time.Second},
    }
}

// DiscoverAgents 从配置的 URL 列表中发现 Agent
func (r *Registry) DiscoverAgents(ctx context.Context, agentURLs map[string]string) {
    for name, url := range agentURLs {
        card, err := r.fetchAgentCard(ctx, url)
        if err != nil {
            log.Printf("registry: failed to discover %s at %s: %v", name, url, err)
            r.mu.Lock()
            r.agents[name] = &AgentInfo{
                Name: name, URL: url, Healthy: false, LastCheck: time.Now(),
            }
            r.mu.Unlock()
            continue
        }
        r.mu.Lock()
        r.agents[name] = card
        r.agents[name].URL = url
        r.agents[name].Healthy = true
        r.agents[name].LastCheck = time.Now()
        r.mu.Unlock()
        log.Printf("registry: discovered agent %s at %s (skills: %d)", name, url, len(card.Skills))
    }
}

// StartHealthCheck 启动定期健康检查（每30秒）
func (r *Registry) StartHealthCheck(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for {
            select {
            case <-ctx.Done():
                ticker.Stop()
                return
            case <-ticker.C:
                r.checkAll(ctx)
            }
        }
    }()
}

func (r *Registry) checkAll(ctx context.Context) {
    r.mu.RLock()
    urls := make(map[string]string)
    for name, info := range r.agents {
        urls[name] = info.URL
    }
    r.mu.RUnlock()

    for name, url := range urls {
        healthy := r.pingAgent(ctx, url)
        r.mu.Lock()
        if agent, ok := r.agents[name]; ok {
            agent.Healthy = healthy
            agent.LastCheck = time.Now()
        }
        r.mu.Unlock()
    }
}

func (r *Registry) pingAgent(ctx context.Context, url string) bool {
    req, _ := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
    resp, err := r.client.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

func (r *Registry) fetchAgentCard(ctx context.Context, url string) (*AgentInfo, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url+"/.well-known/agent.json", nil)
    resp, err := r.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("agent card returned %d", resp.StatusCode)
    }
    var info AgentInfo
    if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
        return nil, err
    }
    return &info, nil
}

// GetHealthyAgents 返回所有健康的 Agent 列表
func (r *Registry) GetHealthyAgents() []*AgentInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var result []*AgentInfo
    for _, a := range r.agents {
        if a.Healthy {
            result = append(result, a)
        }
    }
    return result
}

// GetAgent 获取指定 Agent 信息
func (r *Registry) GetAgent(name string) (*AgentInfo, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    info, ok := r.agents[name]
    return info, ok
}

// GetAllAgents 返回所有 Agent（含不健康的）
func (r *Registry) GetAllAgents() []*AgentInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var result []*AgentInfo
    for _, a := range r.agents {
        result = append(result, a)
    }
    return result
}
```

#### 人员 B（前端）

**任务 2.4：Markdown 渲染支持**

文件：`frontend/src/components/StreamingText.tsx`（改造）

```tsx
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Props {
  content: string
  isStreaming: boolean
}

export default function StreamingText({ content, isStreaming }: Props) {
  return (
    <div className="text-sm prose prose-sm max-w-none prose-p:my-1 prose-headings:my-2">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>
        {content}
      </ReactMarkdown>
      {isStreaming && <span className="inline-block w-2 h-4 bg-gray-400 animate-pulse ml-0.5" />}
    </div>
  )
}
```

**任务 2.5：WebPreview 组件实现**

文件：`frontend/src/components/WebPreview.tsx`（新建）

```tsx
import { useState } from 'react'
import { Globe, Maximize2, Minimize2, ExternalLink } from 'lucide-react'
import type { WebPreviewData } from '../types'

interface Props {
  data: WebPreviewData
}

export default function WebPreview({ data }: Props) {
  const [expanded, setExpanded] = useState(false)

  // 构建完整 HTML 文档（用于 srcdoc）
  const fullHTML = data.html.includes('<!DOCTYPE') || data.html.includes('<html')
    ? data.html
    : `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><style>${data.css || ''}</style></head>
<body>${data.html}<script>${data.js || ''}</script></body>
</html>`

  const handleOpenInNew = () => {
    const blob = new Blob([fullHTML], { type: 'text/html' })
    const url = URL.createObjectURL(blob)
    window.open(url, '_blank')
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }

  return (
    <div className={`rounded-lg border border-gray-300 overflow-hidden mt-3 shadow-sm ${expanded ? 'fixed inset-4 z-50 bg-white' : ''}`}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 bg-gray-100 text-gray-700 text-xs border-b">
        <div className="flex items-center gap-2">
          <Globe className="w-3.5 h-3.5 text-blue-500" />
          <span className="font-medium">{data.title || 'Web Preview'}</span>
        </div>
        <div className="flex items-center gap-1">
          <button onClick={handleOpenInNew} className="p-1 rounded hover:bg-gray-200" title="Open in new tab">
            <ExternalLink className="w-3.5 h-3.5" />
          </button>
          <button onClick={() => setExpanded(!expanded)} className="p-1 rounded hover:bg-gray-200" title={expanded ? 'Exit fullscreen' : 'Fullscreen'}>
            {expanded ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
          </button>
        </div>
      </div>
      {/* iframe */}
      <iframe
        srcDoc={fullHTML}
        sandbox="allow-scripts"
        className="w-full border-0 bg-white"
        style={{ height: expanded ? 'calc(100% - 36px)' : '400px' }}
        title={data.title || 'Web Preview'}
      />
      {/* Overlay for fullscreen background */}
      {expanded && <div className="fixed inset-0 bg-black/50 -z-10" onClick={() => setExpanded(false)} />}
    </div>
  )
}
```

**任务 2.6：更新 MessageBubble 支持 WebPreview**

文件：`frontend/src/components/MessageBubble.tsx`

在 CodePreview 之后添加 WebPreview 渲染逻辑：

```tsx
{/* Web previews */}
{message.webPreviews && message.webPreviews.length > 0 && (
  <div className="w-full mt-1">
    {message.webPreviews.map((preview, i) => (
      <WebPreview key={`${message.id}-web-${i}`} data={preview} />
    ))}
  </div>
)}
```

#### Day 2 交付标准

- [ ] web-agent 可独立运行，`curl POST http://localhost:8082/` 返回 SSE 流
- [ ] web-agent 的 AgentCard：`curl http://localhost:8082/.well-known/agent.json` 返回正确 JSON
- [ ] Agent Registry 可从 URL 列表发现 Agent
- [ ] WebPreview 组件渲染 HTML/CSS/JS，iframe 可交互
- [ ] 文本回复支持 Markdown 渲染

---

### Day 3：LLM 意图编排器

> **目标**：实现 LLM 驱动的智能路由，根据用户意图自动选择合适的 Agent。

#### 人员 A（后端）

**任务 3.1：Planner 意图编排器**

文件：`server/internal/orchestrator/planner.go`（新建）

```go
package orchestrator

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "strings"

    "github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/registry"
)

// ExecutionPlan 编排器输出的执行计划
type ExecutionPlan struct {
    Intent   string     `json:"intent"`   // 意图摘要
    Strategy string     `json:"strategy"` // single | parallel | sequential
    Tasks    []TaskPlan `json:"tasks"`    // 任务列表
}

// TaskPlan 单个子任务
type TaskPlan struct {
    AgentName   string `json:"agent_name"`
    TaskContent string `json:"task_content"`
    DependsOn   string `json:"depends_on,omitempty"`
}

// Planner LLM驱动的意图编排器
type Planner struct {
    llmClient *PlannerLLM
}

// NewPlanner 创建编排器
func NewPlanner(apiKey, model, baseURL, provider string) *Planner {
    return &Planner{
        llmClient: NewPlannerLLM(apiKey, model, baseURL, provider),
    }
}

// Plan 分析用户意图并生成执行计划
func (p *Planner) Plan(ctx context.Context, userMsg string, history []StructuredMessage, agents []*registry.AgentInfo) (*ExecutionPlan, error) {
    prompt := p.buildPrompt(userMsg, history, agents)

    result, err := p.llmClient.Generate(ctx, prompt)
    if err != nil {
        log.Printf("planner LLM error: %v, falling back to default routing", err)
        return p.fallbackPlan(userMsg, agents), nil
    }

    var plan ExecutionPlan
    if err := json.Unmarshal([]byte(result), &plan); err != nil {
        log.Printf("planner JSON parse error: %v, result: %s", err, result)
        return p.fallbackPlan(userMsg, agents), nil
    }

    // 验证 plan 中的 agent 都存在
    for i, task := range plan.Tasks {
        found := false
        for _, agent := range agents {
            if agent.Name == task.AgentName {
                found = true
                break
            }
        }
        if !found {
            // 降级到第一个可用 Agent
            if len(agents) > 0 {
                plan.Tasks[i].AgentName = agents[0].Name
            }
        }
    }

    return &plan, nil
}

func (p *Planner) buildPrompt(userMsg string, history []StructuredMessage, agents []*registry.AgentInfo) string {
    var agentDescs []string
    for _, a := range agents {
        skills := make([]string, 0, len(a.Skills))
        for _, s := range a.Skills {
            skills = append(skills, s.ID)
        }
        agentDescs = append(agentDescs, fmt.Sprintf(
            "- %s: %s (skills: %s, output: %s)",
            a.Name, a.Description, strings.Join(skills, ","), strings.Join(a.OutputModes, ","),
        ))
    }

    var historyStr string
    if len(history) > 0 {
        var parts []string
        for _, h := range history {
            parts = append(parts, fmt.Sprintf("[%s]: %s", h.Role, truncate(h.Content, 200)))
        }
        historyStr = strings.Join(parts, "\n")
    }

    return fmt.Sprintf(`You are a task orchestrator. Analyze the user's request and decide how to route it to available agents.

## Available Agents:
%s

## Conversation History:
%s

## Current User Message:
%s

## Instructions:
Return a JSON object with this exact format (no markdown, no explanation, just JSON):
{
  "intent": "brief summary of user intent",
  "strategy": "single" or "parallel" or "sequential",
  "tasks": [
    {"agent_name": "agent-name-here", "task_content": "what to ask this agent"}
  ]
}

Rules:
1. If the task only needs ONE agent, use strategy "single" with one task.
2. If the task has INDEPENDENT parts for different agents, use "parallel".
3. If tasks depend on each other's output, use "sequential".
4. For code-related requests → code-agent
5. For web page/UI/frontend visual requests → web-agent
6. If the request involves BOTH backend code AND frontend UI → use "parallel" with both agents.
7. Always use the exact agent names from the available list.`,
        strings.Join(agentDescs, "\n"),
        historyStr,
        userMsg,
    )
}

// fallbackPlan 当 LLM 失败时的降级策略
func (p *Planner) fallbackPlan(userMsg string, agents []*registry.AgentInfo) *ExecutionPlan {
    // 简单关键词匹配
    targetAgent := "code-agent" // 默认
    lowerMsg := strings.ToLower(userMsg)

    for _, agent := range agents {
        if agent.Name == "web-agent" {
            if strings.Contains(lowerMsg, "网页") || strings.Contains(lowerMsg, "页面") ||
                strings.Contains(lowerMsg, "html") || strings.Contains(lowerMsg, "web page") ||
                strings.Contains(lowerMsg, "ui") || strings.Contains(lowerMsg, "前端") ||
                strings.Contains(lowerMsg, "登录") || strings.Contains(lowerMsg, "界面") {
                targetAgent = "web-agent"
                break
            }
        }
    }

    return &ExecutionPlan{
        Intent:   "user request",
        Strategy: "single",
        Tasks: []TaskPlan{
            {AgentName: targetAgent, TaskContent: userMsg},
        },
    }
}

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
```

**任务 3.2：Planner 用的轻量 LLM Client**

文件：`server/internal/orchestrator/planner_llm.go`（新建）

```go
package orchestrator

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

// PlannerLLM 编排器专用的 LLM 客户端（非流式，获取结构化输出）
type PlannerLLM struct {
    provider string
    apiKey   string
    model    string
    baseURL  string
    client   *http.Client
}

func NewPlannerLLM(apiKey, model, baseURL, provider string) *PlannerLLM {
    return &PlannerLLM{
        provider: provider,
        apiKey:   apiKey,
        model:    model,
        baseURL:  baseURL,
        client:   &http.Client{Timeout: 30_000_000_000}, // 30s
    }
}

// Generate 非流式调用 LLM，返回纯文本结果
func (l *PlannerLLM) Generate(ctx context.Context, prompt string) (string, error) {
    switch l.provider {
    case "openai":
        return l.generateOpenAI(ctx, prompt)
    default:
        return l.generateAnthropic(ctx, prompt)
    }
}

func (l *PlannerLLM) generateAnthropic(ctx context.Context, prompt string) (string, error) {
    body, _ := json.Marshal(map[string]any{
        "model":      l.model,
        "max_tokens": 1024,
        "messages":   []map[string]string{{"role": "user", "content": prompt}},
    })

    req, _ := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/v1/messages", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", l.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    resp, err := l.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("planner LLM unreachable")
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("planner LLM error: %d", resp.StatusCode)
    }

    var result struct {
        Content []struct {
            Text string `json:"text"`
        } `json:"content"`
    }
    data, _ := io.ReadAll(resp.Body)
    if err := json.Unmarshal(data, &result); err != nil {
        return "", err
    }
    if len(result.Content) == 0 {
        return "", fmt.Errorf("empty planner response")
    }

    // 提取 JSON（LLM 可能返回带 markdown 包裹的 JSON）
    text := result.Content[0].Text
    text = strings.TrimSpace(text)
    if strings.HasPrefix(text, "```") {
        lines := strings.Split(text, "\n")
        if len(lines) > 2 {
            text = strings.Join(lines[1:len(lines)-1], "\n")
        }
    }
    return text, nil
}

func (l *PlannerLLM) generateOpenAI(ctx context.Context, prompt string) (string, error) {
    body, _ := json.Marshal(map[string]any{
        "model": l.model,
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
        "response_format": map[string]string{"type": "json_object"},
    })

    req, _ := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/chat/completions", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+l.apiKey)

    resp, err := l.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("planner LLM unreachable")
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("planner LLM error: %d", resp.StatusCode)
    }

    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    data, _ := io.ReadAll(resp.Body)
    if err := json.Unmarshal(data, &result); err != nil {
        return "", err
    }
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("empty planner response")
    }
    return result.Choices[0].Message.Content, nil
}
```

**任务 3.3：改造 Orchestrator 使用 Planner + Registry**

文件：`server/internal/orchestrator/orchestrator.go`（大改）

改造 `Process` 方法，集成 Planner 和并行调度能力。核心流程：
1. 从 Registry 获取可用 Agents
2. 调用 Planner 生成 ExecutionPlan
3. 根据 strategy 执行（single/parallel）
4. A2A 事件 → AG-UI 事件转换

#### 人员 B（前端）

**任务 3.4：更新 skills 注册列表**

文件：`frontend/src/agui/skills.ts`

```typescript
export const frontendSkills: string[] = [
  'code_preview',
  'web_preview',
  'markdown_render',
]
```

**任务 3.5：MessageStore 处理 web_preview tool call**

文件：`frontend/src/stores/messageStore.ts`

在 `TOOL_CALL_END` 处理中增加 `web_preview` 分支：

```typescript
case 'TOOL_CALL_END':
  if (event.toolCallId) {
    const toolName = /* ... */
    const argsStr = toolCallArgs[event.toolCallId] || '{}'
    try {
      const args = JSON.parse(argsStr)
      if (toolName === 'code_preview') {
        codeBlocks.push({ code: args.code, language: args.language, filename: args.filename })
      } else if (toolName === 'web_preview') {
        webPreviews.push({ html: args.html || args.code, css: args.css, js: args.js, title: args.title || args.filename })
      }
      // 更新消息
      set(...)
    } catch { /* ... */ }
  }
  break
```

**任务 3.6：更新 API 获取 Agent 列表（从 Registry）**

文件：`frontend/src/services/api.ts`

```typescript
export async function listAgents(): Promise<Agent[]> {
  const res = await fetch(`${API_BASE}/agents`)
  if (!res.ok) throw new Error('Failed to load agents')
  return res.json()
}
```

文件：`server/internal/handler/conversation.go`  
改造 `ListAgents` 从 Registry 读取：

```go
func (h *Handler) ListAgents(c *gin.Context) {
    agents := h.registry.GetAllAgents()
    c.JSON(http.StatusOK, agents)
}
```

#### Day 3 交付标准

- [ ] 发送 "写一个Go HTTP服务器" → Planner 选择 code-agent
- [ ] 发送 "写一个登录页面" → Planner 选择 web-agent
- [ ] 发送 "做一个待办清单应用，前端+后端" → Planner 返回 parallel 策略
- [ ] Planner LLM 超时/失败时 fallback 到关键词匹配正常工作
- [ ] `/api/agents` 返回 Registry 中所有 Agent 信息

---

### Day 4：并行调度 + 协议转换增强

> **目标**：Orchestrator 支持 parallel 策略调用多个 Agent，前端正确渲染多 Agent 响应。

#### 人员 A（后端）

**任务 4.1：Orchestrator 并行执行引擎**

文件：`server/internal/orchestrator/orchestrator.go`

改造 `Process` 方法的核心逻辑：

```go
func (o *Orchestrator) Process(ctx context.Context, req model.AGUIRunRequest, history []model.Message, events chan<- model.AGUIEvent) {
    runID := req.RunID
    if runID == "" {
        runID = "run-" + uuid.New().String()[:8]
    }
    events <- model.AGUIEvent{Type: "RUN_STARTED", RunID: runID}

    // 1. 构建结构化消息
    structuredMsgs := o.buildStructuredMessages(req, history)
    userMsg := req.Messages[len(req.Messages)-1].Content

    // 2. 获取健康 Agent 列表
    healthyAgents := o.registry.GetHealthyAgents()
    if len(healthyAgents) == 0 {
        events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "no healthy agents available"}
        return
    }

    // 3. LLM 意图编排
    plan, err := o.planner.Plan(ctx, userMsg, structuredMsgs, healthyAgents)
    if err != nil {
        events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "orchestration failed"}
        return
    }

    // 4. 发送编排状态给前端
    events <- model.AGUIEvent{
        Type:    "STATE_UPDATE",
        Content: fmt.Sprintf(`{"intent":"%s","strategy":"%s","agents":%d}`, plan.Intent, plan.Strategy, len(plan.Tasks)),
    }

    // 5. 根据策略执行
    skills := o.extractSkills(req.Tools)
    switch plan.Strategy {
    case "parallel":
        o.executeParallel(ctx, plan, skills, events)
    case "sequential":
        o.executeSequential(ctx, plan, skills, events)
    default: // "single"
        o.executeSingle(ctx, plan, skills, events)
    }

    events <- model.AGUIEvent{Type: "RUN_FINISHED"}
}

func (o *Orchestrator) executeSingle(ctx context.Context, plan *ExecutionPlan, skills []string, events chan<- model.AGUIEvent) {
    if len(plan.Tasks) == 0 {
        return
    }
    task := plan.Tasks[0]
    o.callAgent(ctx, task, skills, events)
}

func (o *Orchestrator) executeParallel(ctx context.Context, plan *ExecutionPlan, skills []string, events chan<- model.AGUIEvent) {
    // 串行执行但前端感知为多Agent（MVP并行简化版）
    // 真正的 goroutine 并行会导致事件交错，前端难以处理
    // 简化方案：按顺序调用，每个 Agent 有独立的消息流
    for _, task := range plan.Tasks {
        // 通知前端切换 Agent
        events <- model.AGUIEvent{
            Type:    "STATE_UPDATE",
            Content: fmt.Sprintf(`{"activeAgent":"%s"}`, task.AgentName),
        }
        o.callAgent(ctx, task, skills, events)
    }
}

func (o *Orchestrator) executeSequential(ctx context.Context, plan *ExecutionPlan, skills []string, events chan<- model.AGUIEvent) {
    // sequential 和 parallel 简化版行为一致（依次执行）
    o.executeParallel(ctx, plan, skills, events)
}

func (o *Orchestrator) callAgent(ctx context.Context, task TaskPlan, skills []string, events chan<- model.AGUIEvent) {
    agentInfo, ok := o.registry.GetAgent(task.AgentName)
    if !ok || !agentInfo.Healthy {
        events <- model.AGUIEvent{Type: "RUN_ERROR", Error: fmt.Sprintf("agent %s unavailable", task.AgentName)}
        return
    }

    converter := NewConverter(skills)

    eventIter, err := o.a2aClient.SendStreamingMessage(ctx, agentInfo.URL, task.TaskContent)
    if err != nil {
        events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "agent communication error"}
        return
    }

    for event, err := range eventIter {
        if err != nil {
            return
        }
        aguiEvents := converter.ConvertA2AEvent(event)
        for _, e := range aguiEvents {
            if e.Type == "RUN_FINISHED" || e.Type == "RUN_ERROR" {
                // 不在这里发 RUN_FINISHED，由 Process 统一发
                continue
            }
            select {
            case <-ctx.Done():
                return
            case events <- e:
            }
        }
    }
}
```

**任务 4.2：Converter 支持 webpage artifact**

文件：`server/internal/orchestrator/converter.go`

在 `flushArtifacts()` 中增加 web_preview 映射：

```go
func (c *ProtocolConverter) flushArtifacts() []model.AGUIEvent {
    var events []model.AGUIEvent

    for _, art := range c.artifacts {
        var toolName string
        var args map[string]string

        switch {
        case art.language == "html" || c.isWebArtifact(art):
            if !c.hasSkill("web_preview") {
                continue
            }
            toolName = "web_preview"
            args = map[string]string{
                "html":  art.content,
                "css":   art.css,
                "js":    art.js,
                "title": art.filename,
            }
        default:
            if !c.hasSkill("code_preview") {
                continue
            }
            toolName = "code_preview"
            args = map[string]string{
                "code":     art.content,
                "language": art.language,
                "filename": art.filename,
            }
        }

        toolCallID := "tc-" + uuid.New().String()[:8]
        argsJSON, _ := json.Marshal(args)

        events = append(events,
            model.AGUIEvent{Type: "TOOL_CALL_START", ToolCallID: toolCallID, ToolName: toolName, MessageID: c.messageID},
            model.AGUIEvent{Type: "TOOL_CALL_ARGS", ToolCallID: toolCallID, Content: string(argsJSON)},
            model.AGUIEvent{Type: "TOOL_CALL_END", ToolCallID: toolCallID},
        )
    }
    c.artifacts = nil
    return events
}
```

**任务 4.3：AGUIEvent 增加 STATE_UPDATE 支持**

文件：`server/internal/model/agui.go`

```go
type AGUIEvent struct {
    Type       string `json:"type"`
    MessageID  string `json:"messageId,omitempty"`
    RunID      string `json:"runId,omitempty"`
    Content    string `json:"content,omitempty"`
    ToolCallID string `json:"toolCallId,omitempty"`
    ToolName   string `json:"toolName,omitempty"`
    Error      string `json:"error,omitempty"`
    State      string `json:"state,omitempty"`  // 新增：STATE_UPDATE 时的状态 JSON
}
```

#### 人员 B（前端）

**任务 4.4：前端处理 STATE_UPDATE 事件**

文件：`frontend/src/stores/messageStore.ts`

在 event handler 中新增：

```typescript
case 'STATE_UPDATE':
  // 解析状态信息，可用于显示编排过程
  try {
    const state = JSON.parse(event.content || event.state || '{}')
    if (state.activeAgent) {
      // 记录当前活跃 Agent，下一条消息使用此 Agent 名称
      currentAgentName = state.activeAgent
    }
  } catch {}
  break
```

**任务 4.5：多 Agent 头像区分**

文件：`frontend/src/components/AgentAvatar.tsx`（改造）

```tsx
import { Bot, Globe, Code } from 'lucide-react'

interface Props {
  agentName?: string
  isStreaming?: boolean
}

const agentIcons: Record<string, { icon: typeof Bot; color: string }> = {
  'code-agent': { icon: Code, color: 'bg-green-100 text-green-600' },
  'web-agent': { icon: Globe, color: 'bg-blue-100 text-blue-600' },
}

export default function AgentAvatar({ agentName, isStreaming }: Props) {
  const config = agentIcons[agentName || ''] || { icon: Bot, color: 'bg-indigo-100 text-indigo-600' }
  const Icon = config.icon

  return (
    <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${config.color} ${isStreaming ? 'animate-pulse' : ''}`}>
      <Icon className="w-5 h-5" />
    </div>
  )
}
```

**任务 4.6：ConversationList 增加 Agent 选择**

文件：`frontend/src/components/ConversationList.tsx`

新建对话时支持选择 Agent（弹出简单选择菜单）。

#### Day 4 交付标准

- [ ] 发送需要多 Agent 的请求 → 看到两个 Agent 依次回复
- [ ] 每个 Agent 回复有独立的消息气泡和头像
- [ ] web-agent 的 HTML 产物正确触发 web_preview tool call
- [ ] WebPreview iframe 可正常渲染

---

### Day 5：群聊模式 + 完善 UI

> **目标**：实现群聊模式（一个对话中包含多 Agent），完善交互体验。

#### 人员 A（后端）

**任务 5.1：对话支持群聊标识**

文件：`init.sql` — 增加 `conversation_type` 字段

```sql
ALTER TABLE conversations ADD COLUMN conversation_type VARCHAR(20) DEFAULT 'single';
-- single: 单Agent对话
-- group: 多Agent群聊
```

或直接改 init.sql 重建：

```sql
CREATE TABLE conversations (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    title VARCHAR(255) NOT NULL DEFAULT 'New Conversation',
    agent_name VARCHAR(100) NOT NULL,
    conversation_type VARCHAR(20) NOT NULL DEFAULT 'single',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;
```

**任务 5.2：messages 表增加 sender_name 使用**

Agent 回复时保存 agent name 到 `sender_name` 字段，前端通过此字段显示头像。

文件：`server/internal/handler/agui.go`

在 persist 阶段，根据 STATE_UPDATE 中的 activeAgent 记录 sender_name。

**任务 5.3：群聊路由逻辑**

当 `conversation_type = "group"` 时：
- Planner 始终分析意图（不跳过）
- 支持 `@agent-name` 语法直接路由

文件：`server/internal/orchestrator/orchestrator.go`

```go
// 检查是否有 @ 指定
func extractMentionedAgent(msg string) string {
    if idx := strings.Index(msg, "@"); idx >= 0 {
        rest := msg[idx+1:]
        parts := strings.Fields(rest)
        if len(parts) > 0 {
            return parts[0]
        }
    }
    return ""
}
```

**任务 5.4：Handler 暴露 Agent 信息给前端**

文件：`server/internal/handler/conversation.go`

```go
// ListAgents 从 Registry 获取真实列表
func (h *Handler) ListAgents(c *gin.Context) {
    agents := h.registry.GetAllAgents()
    result := make([]gin.H, 0, len(agents))
    for _, a := range agents {
        result = append(result, gin.H{
            "name":        a.Name,
            "description": a.Description,
            "healthy":     a.Healthy,
            "skills":      a.Skills,
        })
    }
    c.JSON(http.StatusOK, result)
}
```

#### 人员 B（前端）

**任务 5.5：新建对话 — 支持群聊模式选择**

文件：`frontend/src/components/NewConversationModal.tsx`（新建）

```tsx
// 弹窗或下拉菜单
// - 单聊：选择一个 Agent
// - 群聊：选择多个 Agent（或默认全选）
// 创建请求：POST /api/conversations { title, agentName, conversationType: "group" }
```

**任务 5.6：群聊 UI — Agent 标识 + 消息分组**

文件：`frontend/src/components/MessageBubble.tsx`

```tsx
// Agent 消息头部显示名称
{!isUser && (
  <span className="text-xs text-gray-500 mb-1 ml-1">
    {message.senderName || 'Agent'}
  </span>
)}
```

**任务 5.7：输入框支持 @Agent 提及**

文件：`frontend/src/components/MessageInput.tsx`

简化版：输入 `@` 时显示 Agent 列表浮层。

```tsx
// 当检测到输入 "@" 字符时
// 显示 Agent 选择浮层
// 选择后在输入框插入 "@agent-name "
```

**任务 5.8：对话 Store 支持 conversationType**

文件：`frontend/src/types/index.ts`

```typescript
export interface Conversation {
  id: string
  title: string
  agentName: string
  conversationType: 'single' | 'group' // 新增
  createdAt: string
  updatedAt: string
}
```

#### Day 5 交付标准

- [ ] 可创建群聊对话
- [ ] 群聊中发送消息 → 多 Agent 依次回复，UI 正确标识
- [ ] `@web-agent 写个按钮` → 直接路由到 web-agent
- [ ] 不指定 Agent 时 → Planner 自动编排

---

### Day 6：错误降级 + Docker 集成 + 测试

> **目标**：完善错误处理，确保 Docker 一键启动可用，补充关键测试。

#### 人员 A（后端）

**任务 6.1：Agent 调用失败降级**

文件：`server/internal/orchestrator/orchestrator.go`

```go
func (o *Orchestrator) callAgentWithFallback(ctx context.Context, task TaskPlan, skills []string, events chan<- model.AGUIEvent) {
    err := o.callAgent(ctx, task, skills, events)
    if err == nil {
        return
    }

    // 尝试降级到其他 Agent
    fallbackAgents := o.registry.GetHealthyAgents()
    for _, agent := range fallbackAgents {
        if agent.Name == task.AgentName {
            continue
        }
        events <- model.AGUIEvent{
            Type:    "STATE_UPDATE",
            Content: fmt.Sprintf(`{"phase":"retrying","failedAgent":"%s","fallbackAgent":"%s"}`, task.AgentName, agent.Name),
        }
        fallbackTask := TaskPlan{AgentName: agent.Name, TaskContent: task.TaskContent}
        if err := o.callAgent(ctx, fallbackTask, skills, events); err == nil {
            return
        }
    }

    events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "all agents failed"}
}
```

**任务 6.2：更新 docker-compose.yml**

文件：`docker-compose.yml`

```yaml
services:
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - gateway
    environment:
      - VITE_API_URL=http://gateway:8080

  gateway:
    build: ./server
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=${DB_HOST:-mysql}
      - DB_PORT=${DB_PORT:-3306}
      - DB_USER=${DB_USER:-root}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME:-agenthub}
      - AGENTHUB_API_TOKEN=${AGENTHUB_API_TOKEN}
      - AGENT_CODE_URL=http://code-agent:8081
      - AGENT_WEB_URL=http://web-agent:8082
      - GATEWAY_PORT=8080
      - LLM_PROVIDER=${LLM_PROVIDER:-anthropic}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - ANTHROPIC_MODEL=${ANTHROPIC_MODEL:-claude-sonnet-4-20250514}
      - ANTHROPIC_BASE_URL=${ANTHROPIC_BASE_URL:-https://api.anthropic.com}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_MODEL=${OPENAI_MODEL:-gpt-4o}
      - OPENAI_BASE_URL=${OPENAI_BASE_URL:-https://api.openai.com/v1}
    depends_on:
      mysql:
        condition: service_healthy
    restart: unless-stopped

  code-agent:
    build:
      context: ./agents
      dockerfile: code-agent/Dockerfile
    ports:
      - "8081:8081"
    environment:
      - CODE_AGENT_PORT=8081
      - LLM_PROVIDER=${LLM_PROVIDER:-anthropic}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - ANTHROPIC_MODEL=${ANTHROPIC_MODEL:-claude-sonnet-4-20250514}
      - ANTHROPIC_BASE_URL=${ANTHROPIC_BASE_URL:-https://api.anthropic.com}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_MODEL=${OPENAI_MODEL:-gpt-4o}
      - OPENAI_BASE_URL=${OPENAI_BASE_URL:-https://api.openai.com/v1}
    restart: unless-stopped

  web-agent:
    build:
      context: ./agents
      dockerfile: web-agent/Dockerfile
    ports:
      - "8082:8082"
    environment:
      - WEB_AGENT_PORT=8082
      - LLM_PROVIDER=${LLM_PROVIDER:-anthropic}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - ANTHROPIC_MODEL=${ANTHROPIC_MODEL:-claude-sonnet-4-20250514}
      - ANTHROPIC_BASE_URL=${ANTHROPIC_BASE_URL:-https://api.anthropic.com}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_MODEL=${OPENAI_MODEL:-gpt-4o}
      - OPENAI_BASE_URL=${OPENAI_BASE_URL:-https://api.openai.com/v1}
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: ${MYSQL_DATABASE:-agenthub}
    ports:
      - "3306:3306"
    volumes:
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
      - mysqldata:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 10
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    restart: unless-stopped

volumes:
  mysqldata:
```

**任务 6.3：Gateway config 支持多 Agent URL**

文件：`server/internal/config/config.go`

```go
type Config struct {
    Port        string
    DatabaseURL string
    APIToken    string
    AgentURLs   map[string]string // name → URL
    // LLM 配置（供 Planner 使用）
    LLMProvider string
    LLMAPIKey   string
    LLMModel    string
    LLMBaseURL  string
}

func Load() (*Config, error) {
    // ...
    agentURLs := map[string]string{
        "code-agent": getEnv("AGENT_CODE_URL", "http://localhost:8081"),
        "web-agent":  getEnv("AGENT_WEB_URL", "http://localhost:8082"),
    }
    // ...
}
```

**任务 6.4：Planner 测试**

文件：`server/internal/orchestrator/planner_test.go`

```go
func TestPlannerFallbackOnLLMError(t *testing.T) { ... }
func TestPlannerValidatesAgentNames(t *testing.T) { ... }
func TestExtractMentionedAgent(t *testing.T) { ... }
```

#### 人员 B（前端）

**任务 6.5：前端 Dockerfile 改为生产构建**

文件：`frontend/Dockerfile`

```dockerfile
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
RUN npm install -g serve
COPY --from=build /app/dist ./dist
EXPOSE 3000
CMD ["serve", "-s", "dist", "-l", "3000"]
```

或使用 nginx：

```dockerfile
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
```

文件：`frontend/nginx.conf`（新建）

```nginx
server {
    listen 3000;
    root /usr/share/nginx/html;
    index index.html;

    location /api {
        proxy_pass http://gateway:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

**任务 6.6：错误降级 UI 提示**

前端处理 STATE_UPDATE 中的 retrying 状态，显示提示：

```tsx
// 当收到 retrying 状态时，在消息流中插入系统提示
if (state.phase === 'retrying') {
  // 显示 "Agent-A 失败，正在切换到 Agent-B..."
}
```

**任务 6.7：补充 E2E 测试场景**

更新 E2E mock，增加多 Agent 场景的测试用例。

#### Day 6 交付标准

- [ ] `docker compose up --build` 一键启动 5 个服务（mysql + gateway + code-agent + web-agent + frontend）
- [ ] 所有服务 health check 通过
- [ ] Agent 失败时有降级提示
- [ ] 前端 Docker 镜像为生产构建（非 dev 模式）
- [ ] smoke-test.sh 通过（含新增 web-agent 检查）

---

### Day 7：Demo 打磨 + 文档整理 + 录制

> **目标**：最终打磨、录制 Demo 视频、整理交付文档。

#### 人员 A + B 协作

**任务 7.1：端到端 Demo 验证**

按以下场景逐一验证，确保流畅：

```
Demo 脚本（3分钟）：

[0:00-0:30] 场景1: 平台总览
  - 打开 AgentHub 首页
  - 展示对话列表（空状态）
  - 展示可用 Agent 列表（code-agent、web-agent）

[0:30-1:15] 场景2: 单聊 code-agent
  - 新建对话 → 选择 code-agent
  - 发送："用 Go 写一个带中间件的 HTTP 服务器"
  - 展示：流式回复 + 代码预览卡片（语法高亮+复制）
  - 追问："加上日志中间件"（验证上下文连续）

[1:15-2:00] 场景3: 单聊 web-agent  
  - 新建对话 → 选择 web-agent
  - 发送："写一个好看的登录页面，带动画效果"
  - 展示：流式回复 + WebPreview iframe 实时预览
  - 点击全屏查看

[2:00-2:45] 场景4: 群聊多Agent协作（核心亮点）
  - 新建群聊对话
  - 发送："做一个计数器应用，要一个好看的前端页面和Go后端API"
  - 展示：
    - Orchestrator 拆解为2个子任务
    - web-agent 先回复（网页预览）
    - code-agent 再回复（Go代码预览）
    - 两个 Agent 有不同头像标识

[2:45-3:00] 场景5: 收尾
  - 刷新页面 → 历史消息保留（持久化验证）
  - 展示 Docker 一键部署能力
```

**任务 7.2：UI 最终打磨**

| 打磨项 | 说明 |
|--------|------|
| 加载动画 | 对话列表和消息加载时有 skeleton |
| 空状态优化 | 新对话时显示引导提示 |
| 响应式 | 移动端基本可用（侧边栏折叠） |
| 编排可视化 | 群聊时显示 "正在编排..." → "已分派给 X 和 Y" |
| 时间戳 | 消息显示相对时间 |

**任务 7.3：更新 README.md**

更新架构图、环境变量说明、新增 web-agent 相关内容。

**任务 7.4：整理 AI 协作记录**

确保 `.agents/skills/` 目录中的 contract 文档完整且最新。这对应考察维度中权重最高的 **AI协作能力（30%）**。

需要整理的文档：
- [ ] 意图编排协议契约
- [ ] A2A Agent 通信契约
- [ ] AG-UI 事件协议契约
- [ ] ADK Runtime 运行时契约
- [ ] 前端 Skills 注册契约

**任务 7.5：录制 3 分钟 Demo 视频**

工具推荐：OBS / macOS 自带录屏 / Loom

**任务 7.6：smoke-test 更新**

文件：`smoke-test.sh` — 增加 web-agent 和编排能力验证

```bash
# 9. Web-agent /health
echo "9. Web-agent /health"
check_url "http://localhost:8082/health" "web-agent"

# 10. Web-agent AgentCard
echo "10. Web-agent AgentCard"
CARD=$(curl -sf http://localhost:8082/.well-known/agent.json 2>&1)
if echo "$CARD" | grep -q "web-agent"; then
  green "  OK  web-agent AgentCard valid"
else
  red "  FAIL web-agent AgentCard"
  FAILED=1
fi
```

#### Day 7 交付标准

- [ ] 3 分钟 Demo 场景全部流畅跑通
- [ ] `make docker-up` → 全栈启动无报错
- [ ] `make smoke-test-ci` 通过
- [ ] README 更新完毕
- [ ] AI 协作文档（.agents/skills/）完整
- [ ] Demo 视频录制完成

---

## 四、风险与应对

| 风险 | 概率 | 影响 | 应对措施 |
|------|:----:|:----:|----------|
| LLM 编排不稳定（返回非法 JSON） | 高 | 中 | fallback 关键词匹配兜底 |
| web-agent 生成的 HTML 质量不佳 | 中 | 中 | 优化 system prompt，增加 few-shot 示例 |
| 并行调度事件交错 | 中 | 高 | 简化为顺序调度（保证事件有序） |
| Docker 构建时间过长 | 低 | 低 | 利用多阶段构建 + 缓存 |
| Anthropic API 限流 | 中 | 中 | 支持 OpenAI Provider 切换 |
| 2人7天时间不够 | 中 | 高 | 优先保证 Day1-5 核心功能，Day6-7 可压缩 |

---

## 五、关键文件变更清单

### 新增文件

```
agents/web-agent/main.go
agents/web-agent/handler.go
agents/web-agent/config.yaml
agents/web-agent/Dockerfile
server/internal/registry/registry.go
server/internal/orchestrator/planner.go
server/internal/orchestrator/planner_llm.go
server/internal/orchestrator/planner_test.go
frontend/src/components/WebPreview.tsx
frontend/src/components/ErrorBoundary.tsx
frontend/src/components/NewConversationModal.tsx
frontend/nginx.conf
```

### 修改文件

```
agents/adk/llm.go                          (HTTP 超时)
agents/adk/types.go                        (webpage artifact)
agents/code-agent/main.go                  (LLM 单例)
agents/code-agent/handler.go               (注入 LLM)
server/cmd/server/main.go                  (graceful shutdown + registry 初始化)
server/internal/config/config.go           (多Agent URL + LLM config)
server/internal/a2a/client.go              (连接复用)
server/internal/orchestrator/orchestrator.go (Planner + 并行调度)
server/internal/orchestrator/converter.go  (web_preview 映射)
server/internal/model/agui.go             (STATE_UPDATE)
server/internal/handler/conversation.go    (ListAgents from Registry)
server/internal/handler/agui.go           (agent name 持久化)
frontend/src/agui/client.ts               (SSE 解析修复)
frontend/src/agui/skills.ts               (新增 skills)
frontend/src/stores/messageStore.ts       (多Agent + web_preview)
frontend/src/stores/conversationStore.ts  (conversationType)
frontend/src/types/index.ts               (WebPreviewData + senderName)
frontend/src/components/MessageBubble.tsx  (Agent 名称 + WebPreview)
frontend/src/components/StreamingText.tsx  (Markdown 渲染)
frontend/src/components/AgentAvatar.tsx    (多 Agent 头像)
frontend/src/components/ConversationList.tsx (群聊创建)
frontend/src/components/MessageInput.tsx   (@提及)
frontend/src/services/api.ts              (createConversation 参数)
frontend/Dockerfile                       (生产构建)
docker-compose.yml                        (web-agent 服务)
init.sql                                  (conversation_type 字段)
smoke-test.sh                             (web-agent 检查)
.env.example                              (新增环境变量)
Makefile                                  (dev-web-agent 命令)
```

---

## 六、每日站会 Checklist

每天开始前快速同步（5分钟）：

- [ ] 昨天完成了什么？
- [ ] 今天计划做什么？
- [ ] 有什么阻塞？
- [ ] 需要对方配合什么？

---

## 七、Definition of Done

v1.0 Sprint 完成标准：

1. **功能完整**：单聊 + 群聊 + 两个Agent + 意图编排全部可 Demo
2. **一键启动**：`docker compose up --build` 5个服务全部健康
3. **产物丰富**：代码预览 + 网页 iframe 预览 + Markdown 渲染
4. **错误可控**：Agent 失败有降级提示，不白屏
5. **文档齐全**：README + AI协作文档 + Demo 视频
6. **测试覆盖**：核心模块单测通过 + smoke test 通过

---

*最后更新：2026-05-23*
