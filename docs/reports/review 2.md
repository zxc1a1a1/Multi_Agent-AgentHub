
  🔍 AgentHub MVP Code Review 报告

  总体评价

  这是一个 质量不错的 MVP 实现，架构清晰、分层合理、协议设计前瞻性强。代码整体可
  读性好，契约文档丰富，测试覆盖了关键路径。以下是按优先级分类的详细问题和建议。

  ---
  ✅ 做得好的地方

  ┌──────────────┬──────────────────────────────────────────────────────────┐
  │     方面     │                           说明                           │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ 协议分层     │ A2A ↔ AG-UI 转换清晰，前端不直接暴露 A2A 协议细节        │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ ADK 抽象     │ ctx.StreamText() / ctx.AddArtifact() API                 │
  │              │ 设计简洁，handler 不感知传输层                           │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ 安全意识     │ Token auth 中间件、错误信息脱敏（不暴露 stack trace/API  │
  │              │ key）、XSS 测试                                          │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ LLM Provider │ 支持 Anthropic/OpenAI/兼容服务切换，归一化设计好         │
  │  抽象        │                                                          │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ 测试         │ converter_test.go / mysql_test.go / CodePreview.test.tsx │
  │              │  / E2E mock 覆盖全                                       │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ Docker 交付  │ docker-compose + smoke-test.sh + healthcheck 完整        │
  ├──────────────┼──────────────────────────────────────────────────────────┤
  │ 代码风格     │ 每个文件职责单一，注释引用 contract 章节便于追溯         │
  └──────────────┴──────────────────────────────────────────────────────────┘

  ---
  🔴 严重问题 (P0 — 必须修复)

  1. SSE 客户端流解析 Bug — 可能丢失事件

  文件: frontend/src/agui/client.ts (L43-53)

  const lines = buffer.split('\n')
  buffer = lines.pop() || ''

  问题: SSE 事件以 \n\n 分隔，但这里按单 \n split。当一个 chunk 包含完整的 data:
   {...}\n\n 时，split 后会产生 ["data: {...}", "", 
  ...]，而空行被直接跳过。但如果 chunk 在 data: 行中间断开，buffer
  只保留了最后一个不完整片段 — 紧邻的前一行如果也不完整会被错误处理。

  建议: 用 \n\n 分隔 event block，或使用更健壮的 SSE 解析：
  const events = buffer.split('\n\n')
  buffer = events.pop() || '' // keep last incomplete event
  for (const block of events) {
    const dataLine = block.split('\n').find(l => l.startsWith('data:'))
    // ...
  }

  2. http.DefaultClient 无超时 — 潜在资源泄露

  文件: agents/adk/llm.go (L131, L219)

  resp, err := http.DefaultClient.Do(req)

  问题: http.DefaultClient 没有设置 Timeout，如果 LLM Provider 无响应，goroutine
   会永远阻塞。在高并发下会耗尽文件描述符。

  建议:
  var llmHTTPClient = &http.Client{
      Timeout: 120 * time.Second,
  }

  3. A2A Client 每次请求都创建新连接 — 性能问题

  文件: server/internal/a2a/client.go (L33-36)

  client, err := a2aclient.NewFromEndpoints(ctx, endpoints)

  问题: 每次 SendStreamingMessage 都 NewFromEndpoints，意味着每个用户请求都新建
  TCP 连接。在 MVP 阶段可接受，但一旦有并发用户会成为瓶颈。

  建议: 为每个 agent URL 维护一个连接池或复用 client instance。

  4. Conversation ID 未验证 — SQL 注入风险低但存在越权

  文件: server/internal/handler/agui.go (L28)

  h.db.SaveMessage(req.ThreadID, "user", "", lastMsg.Content, nil)

  问题: ThreadID 来自用户输入，虽然用了参数化查询不存在 SQL 注入，但没有验证该
  conversation 是否存在。用户可以向任意 threadId 写入消息。

  建议: 在 SaveMessage 前验证 conversation 存在性，或使用 FK 约束让 INSERT
  自然失败时返回 4xx。

  ---
  🟠 重要问题 (P1 — 应该修复)

  5. LLMClient 每次调用都重新读取环境变量

  文件: agents/code-agent/handler.go (L39)

  llm := adk.NewLLMClient()  // 每次请求都 new

  问题: 每次请求都 os.Getenv() 读取配置并创建新
  client。应在启动时创建一次并注入。

  建议: 在 main.go 中初始化 LLMClient，通过闭包或 struct 传递给 handler。

  6. History 拼接为纯文本 — 丢失结构化上下文

  文件: server/internal/orchestrator/orchestrator.go (L101-119)

  func (o *Orchestrator) buildUserMessage(...) string {
      parts = append(parts, role+": "+h.Content)
      ...
      return strings.Join(parts, "\n\n")
  }

  问题: 多轮对话的 history 被拼成一个字符串发给 LLM，丢失了 role 信息。LLM
  难以区分哪些是 user 说的、哪些是 assistant 说的。

  建议: 将 history 转为结构化的 []LLMMessage 传递，让 LLM API 正确理解多轮对话。

  7. 前端 Docker 用 dev 模式运行 — 生产环境不安全

  文件: frontend/Dockerfile

  CMD ["npm", "run", "dev"]

  问题: Docker 生产镜像应使用 npm run build + serve 或 nginx 部署静态文件。vite 
  dev 暴露 HMR websocket，且性能差。

  建议:
  FROM node:22-alpine AS build
  WORKDIR /app
  COPY . .
  RUN npm ci && npm run build

  FROM nginx:alpine
  COPY --from=build /app/dist /usr/share/nginx/html
  COPY nginx.conf /etc/nginx/conf.d/default.conf

  8. Gateway Dockerfile 无多阶段构建

  文件: server/Dockerfile

  FROM golang:1.26-alpine
  ...
  RUN go build -o server ./cmd/server
  CMD ["./server"]

  问题: 最终镜像包含完整 Go 工具链（~400MB），应分阶段构建仅保留二进制文件。

  建议:
  FROM golang:1.26-alpine AS build
  WORKDIR /app
  COPY . .
  RUN go build -o server ./cmd/server

  FROM alpine:3.20
  RUN apk --no-cache add ca-certificates
  COPY --from=build /app/server /server
  CMD ["/server"]

  9. 缺少 Graceful Shutdown

  文件: server/cmd/server/main.go (L56)

  r.Run(":" + cfg.Port)  // 直接 ListenAndServe，无 graceful shutdown

  问题: 收到 SIGTERM 时（Docker 停止），正在进行的 SSE 流会被强制断开。

  建议:
  srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
  go func() { srv.ListenAndServe() }()
  quit := make(chan os.Signal, 1)
  signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
  <-quit
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()
  srv.Shutdown(ctx)

  10. CORS 使用 cors.Default() — 允许所有来源

  文件: server/cmd/server/main.go (L33)

  r.Use(cors.Default())

  问题: 在生产环境允许任意 origin 请求 API，存在安全风险。

  建议: 配置白名单：
  r.Use(cors.New(cors.Config{
      AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
      AllowMethods: []string{"GET", "POST"},
      AllowHeaders: []string{"Content-Type", "Authorization"},
  }))

  ---
  🟡 改进建议 (P2 — 建议优化)

  11. extractTextFromEvent 函数未被使用

  文件: server/internal/orchestrator/orchestrator.go (L134-152)

  Dead code，应删除。

  12. 前端 Message Store 无并发防护

  文件: frontend/src/stores/messageStore.ts

  如果用户快速连续发送消息，streaming 状态只有一个
  boolean，第二条消息会覆盖第一条的 abort controller。

  建议: 将 streaming 状态改为 per-conversation：
  streaming: Record<string, boolean>
  abortControllers: Record<string, AbortController>

  13. parseCodeBlocks 正则可能误匹配

  文件: agents/code-agent/handler.go (L107)

  var codeBlockRegex =
  regexp.MustCompile("(?s)```(\\w+)(?::([^\\n]+))?\\n(.*?)```")

  问题: 如果 code 内容包含 ``` 字符串（比如 Markdown 教程），会被截断。

  建议: 可接受 MVP 限制，但应在 system prompt 中告知 LLM
  避免在输出中使用三反引号。

  14. DB 连接池配置硬编码

  文件: server/internal/store/mysql.go (L24-25)

  db.SetMaxOpenConns(20)
  db.SetMaxIdleConns(5)

  建议: 通过环境变量或配置文件控制，不同部署环境需要不同值。

  15. conversation_id 参数未做 UUID 格式校验

  文件: server/internal/handler/conversation.go (L56)

  convID := c.Param("id")

  建议: 加入简单的 UUID 格式校验，防止恶意路径遍历。

  16. 前端没有 error boundary

  如果任何组件 throw，整个应用白屏。建议在 App.tsx 外层包装一个 React Error
  Boundary。

  17. Docker Compose 缺少 restart policy

  文件: docker-compose.yml

  建议为 gateway/code-agent/frontend 加上 restart: unless-stopped。

  18. Smoke Test 未验证 SSE 流式功能

  smoke-test.sh 只检查了 healthcheck 和 REST API，核心的 SSE
  流式功能未验证。建议加一个 curl SSE 端点的简单验证（即使无有效 API Key
  也能验证到 RUN_ERROR 返回格式正确）。

  ---
  📊 测试覆盖评估

  ┌────────────────────────┬───────────────────┬────────────────────────────┐
  │          模块          │     覆盖情况      │            评价            │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ orchestrator/converter │ ✅ 全覆盖         │ 测试质量高，覆盖 edge case │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ store/mysql            │ ✅ 全覆盖         │ 使用 sqlmock，覆盖错误路径 │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ CodePreview 组件       │ ✅ 全覆盖         │ XSS 测试、fallback         │
  │                        │                   │ 测试完善                   │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ E2E                    │ ✅ Happy path +   │ Mock                       │
  │                        │ Error path        │ 方式好，不依赖真实后端     │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ handler/agui.go        │ ❌ 未覆盖         │ 核心流式 handler 无测试    │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ adk/context.go         │ ❌ 未覆盖         │ ExecuteHandler             │
  │                        │                   │ 逻辑关键但未测试           │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ adk/llm.go             │ ❌ 未覆盖         │ SSE 解析逻辑应有单元测试   │
  ├────────────────────────┼───────────────────┼────────────────────────────┤
  │ code-agent/handler.go  │ ❌ 未覆盖         │ parseCodeBlocks 应有测试   │
  └────────────────────────┴───────────────────┴────────────────────────────┘

  ---
  🏗️ 架构建议（面向后续迭代）

  1. 多 Agent 路由: 当前 orchestrator 硬编码 "code-agent"，后续应从 AgentCard
  动态发现和 skill-matching
  2. 消息去重: 当前无幂等性保障，网络重试可能导致重复消息
  3. Token 计数/限流: LLM 调用无 rate limiting，一个用户可以打满 Anthropic quota
  4. 可观测性: 缺少 traceId 传播和结构化日志，调试困难
  5. 前端路由: 单页应用无 URL routing，刷新后丢失当前对话选中状态

  ---
  总结

  ┌──────────┬──────────────┬───────────────────────────────────┐
  │   维度   │ 评分 (5分制) │               说明                │
  ├──────────┼──────────────┼───────────────────────────────────┤
  │ 架构设计 │  ⭐⭐⭐⭐⭐  │ 协议分层、契约驱动优秀            │
  ├──────────┼──────────────┼───────────────────────────────────┤
  │ 代码质量 │   ⭐⭐⭐⭐   │ 清晰可读，少量 dead code          │
  ├──────────┼──────────────┼───────────────────────────────────┤
  │ 安全性   │   ⭐⭐⭐⭐   │ 基础安全到位，CORS/auth 需收紧    │
  ├──────────┼──────────────┼───────────────────────────────────┤
  │ 测试     │   ⭐⭐⭐⭐   │ 关键模块覆盖好，handler 层缺测试  │
  ├──────────┼──────────────┼───────────────────────────────────┤
  │ 生产就绪 │    ⭐⭐⭐    │ Dockerfile 和 shutdown 需改进     │
  ├──────────┼──────────────┼───────────────────────────────────┤
  │ 文档     │  ⭐⭐⭐⭐⭐  │ README + contract + curl 示例完整 │
  └──────────┴──────────────┴───────────────────────────────────┘

  总体: 作为 MVP v0.1 这是一个 高质量的起点。P0 问题建议尽快修复（尤其 #1 SSE
  解析和 #2 HTTP 超时），P1 在进入 v0.2 前解决即可。

✻ Cogitated for 8m 27s

※ recap: I reviewed the AgentHub MVP codebase and delivered a detailed code review covering architecture, security, performance, and testing. The review is complete — no pending action unless you want me to fix specific issues identified. (disable recaps in /config)