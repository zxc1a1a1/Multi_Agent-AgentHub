# AgentHub 系统 UML 图
## 使用说明：
安装PlantUML插件，
在 VS Code 的 settings.json 中添加：
```
{
  "plantuml.server": "https://www.plantuml.com/plantuml",
  "plantuml.render": "PlantUMLServer"
} 
```
操作步骤：
ctrl + Shift + P → 输入 Preferences: Open User Settings (JSON)


## 一、系统架构图（组件图）

```plantuml
@startuml AgentHub_Architecture
!theme plain
skinparam componentStyle rectangle
skinparam backgroundColor #FEFEFE
skinparam defaultFontSize 12

title AgentHub 多Agent协作平台 - 系统架构图

' ===== 前端层 =====
package "前端层 (React + TypeScript)" as Frontend #E3F2FD {
    component [IM聊天UI\n(对话列表/聊天窗口/消息气泡)] as ChatUI
    component [AG-UI Client SDK\n(SSE解析/事件分发)] as AGUIClient
    component [前端Skills执行器\n(code_preview/web_preview...)] as SkillExecutor
    component [Zustand Store\n(消息状态/对话状态)] as Store
    
    ChatUI --> Store : 读写状态
    ChatUI --> AGUIClient : 发送消息
    AGUIClient --> SkillExecutor : TOOL_CALL事件
    AGUIClient --> Store : 流式更新
}

' ===== 通信协议 =====
interface "AG-UI Protocol\n(SSE Event Stream)" as AGUI_PROTO #FFF9C4

' ===== 网关层 =====
package "Gateway 网关层 (Go + Gin)" as Gateway #E8F5E9 {
    component [AG-UI Server\n(SSE端点/连接管理)] as AGUIServer
    component [REST API\n(会话CRUD/Agent列表)] as RestAPI
    component [鉴权中间件\n(Token验证)] as Auth
    component [会话管理\n(消息持久化)] as SessionMgr
}

' ===== 编排层 =====
package "统一Agent / Orchestrator (Go)" as Orchestrator #FFF3E0 {
    component [意图编排引擎\n(LLM意图分析/路由决策)] as IntentEngine
    component [协议转换器\n(A2A→AG-UI事件映射)] as Converter
    component [A2A Client\n(SSE流式调用子Agent)] as A2AClient
    component [结果聚合器\n(多Agent结果合并)] as Aggregator
    
    IntentEngine --> A2AClient : 执行计划
    A2AClient --> Converter : A2A Events
    Converter --> Aggregator : AG-UI Events
}

' ===== A2A协议 =====
interface "A2A Protocol\n(Agent-to-Agent)" as A2A_PROTO #F3E5F5

' ===== 子Agent层 =====
package "子Agent集群 (ADK Runtime)" as Agents #FCE4EC {
    component [Code Agent\n(代码生成/审查)] as CodeAgent
    component [Web Agent\n(网页/UI生成)] as WebAgent
    component [Doc Agent\n(文档编写)] as DocAgent
    component [Custom Agent\n(用户自建)] as CustomAgent
    
    note right of CodeAgent
        每个Agent包含:
        - AgentCard (能力声明)
        - A2A Server (标准端点)
        - ADK Runtime (生命周期)
        - LLM调用 (流式)
    end note
}

' ===== 数据层 =====
database "MySQL 8" as MySQL #EFEBE9
database "Redis\n(后续扩展)" as Redis #EFEBE9
storage "对象存储\n(文件/产物)" as OSS #EFEBE9

' ===== 外部服务 =====
cloud "LLM APIs" as LLMCloud #F5F5F5 {
    component [Anthropic\n(Claude)] as Anthropic
    component [OpenAI\n(GPT-4)] as OpenAI
}

' ===== 连接关系 =====
Frontend -down-> AGUI_PROTO
AGUI_PROTO -down-> Gateway

Gateway -down-> Orchestrator : 内部调用
Gateway -right-> MySQL : 读写消息
Gateway -left-> Auth

Orchestrator -down-> A2A_PROTO
A2A_PROTO -down-> Agents

Agents -right-> LLMCloud : 调用LLM API

Aggregator -up-> AGUIServer : AG-UI Events

@enduml
```

## 二、系统分层架构图（部署视图）

```plantuml
@startuml AgentHub_Deployment
!theme plain
skinparam nodeStyle rectangle

title AgentHub - 部署架构图 (Docker Compose)

node "Docker Host" {
    
    node "frontend:3000" as FE #E3F2FD {
        artifact "React App\n(Vite Build)" as ReactApp
    }
    
    node "gateway:8080" as GW #E8F5E9 {
        artifact "Go Binary" as GoBin
        component [AG-UI Server] as AGUIS
        component [Orchestrator] as ORC
        component [REST API] as REST
    }
    
    node "code-agent:8081" as CA #FCE4EC {
        artifact "Go Binary" as AgentBin
        component [ADK Runtime] as ADK
        component [A2A Server] as A2AS
    }
    
    node "web-agent:8082" as WA #FCE4EC {
        artifact "Go Binary" as AgentBin2
    }
    
    database "mysql:3306" as DB #EFEBE9 {
        storage "agenthub DB" as ADB
    }
}

cloud "External" {
    component [Anthropic API] as ANTH
    component [OpenAI API] as OAI
}

FE -down-> GW : "HTTP/SSE :8080"
GW -down-> CA : "A2A :8081"
GW -down-> WA : "A2A :8082"
GW -right-> DB : "TCP :3306"
CA -right-> ANTH : "HTTPS"
WA -right-> OAI : "HTTPS"

@enduml
```

## 三、核心时序图 - 单聊完整流程

```plantuml
@startuml AgentHub_Sequence_SingleChat
!theme plain
skinparam sequenceMessageAlign center
skinparam maxMessageSize 200

title AgentHub - 单聊模式完整时序图

actor User as "👤 用户"
participant React as "🖥️ React前端"
participant AGUIClient as "AG-UI Client\n(SSE解析)"
participant Gateway as "🚪 Gateway\n(Go/Gin)"
participant DB as "🗄️ MySQL"
participant Orchestrator as "🧠 Orchestrator"
participant Converter as "🔄 Protocol\nConverter"
participant A2AClient as "📡 A2A Client"
participant CodeAgent as "🔧 Code-Agent\n(ADK Runtime)"
participant LLM as "🤖 Anthropic\nAPI"

== 阶段1: 用户发送消息 ==

User -> React : 输入消息 + 点击发送
activate React
React -> React : 乐观更新UI\n(显示用户消息气泡)
React -> AGUIClient : 构建 RunRequest\n{threadId, runId, messages,\ntools:[code_preview]}
activate AGUIClient

== 阶段2: AG-UI请求到达Gateway ==

AGUIClient -> Gateway : **POST /api/agui/run**\nContent-Type: application/json
activate Gateway

Gateway -> Gateway : Token鉴权验证
Gateway -> DB : INSERT INTO messages\n(保存用户消息)
activate DB
DB --> Gateway : OK
Gateway -> DB : SELECT messages\nWHERE conversation_id=?\n(获取历史上下文)
DB --> Gateway : 历史消息列表
deactivate DB

== 阶段3: 建立SSE + 启动编排 ==

Gateway -> Gateway : 设置响应头:\nContent-Type: text/event-stream\nCache-Control: no-cache
Gateway -> Orchestrator : **异步启动** Process(\nctx, req, history, eventChan)
activate Orchestrator

== 阶段4: 编排决策 + A2A调用 ==

Orchestrator -> Orchestrator : 路由决策:\n直接选择 req.AgentName
Orchestrator -> Converter : 初始化(frontendSkills)
activate Converter

Orchestrator --> Gateway : eventChan ← RUN_STARTED
Gateway --> AGUIClient : **SSE:** {type:"RUN_STARTED"}
AGUIClient --> React : onEvent(RUN_STARTED)
React --> User : 显示Agent头像\n+ "正在思考..."

Orchestrator -> A2AClient : SendSubscribe(agentURL, messages)
activate A2AClient
A2AClient -> CodeAgent : **POST /a2a/tasks/sendSubscribe**\n{id:"task-1", messages:[...]}
activate CodeAgent

== 阶段5: Code-Agent处理 ==

CodeAgent -> CodeAgent : 加载SystemPrompt\n构建LLM消息列表
CodeAgent -> LLM : **POST /v1/messages**\nstream=true
activate LLM

CodeAgent --> A2AClient : A2A SSE:\n{type:"status", status:"working"}
A2AClient --> Orchestrator : StreamEvent(status:working)
Orchestrator -> Converter : Convert(status:working)
Converter --> Orchestrator : [TEXT_MESSAGE_START{msgId}]
Orchestrator --> Gateway : eventChan ← TEXT_MESSAGE_START
Gateway --> AGUIClient : **SSE:** {type:"TEXT_MESSAGE_START",\nmessageId:"msg-1"}
AGUIClient --> React : store.startStreaming()
React --> User : 创建空的Agent消息气泡\n+ 闪烁光标

== 阶段6: 流式文本传输 (核心循环) ==

loop 每个LLM token chunk (约50-100ms/次)
    LLM --> CodeAgent : SSE: content_block_delta\n{text:"package main..."}
    CodeAgent --> A2AClient : A2A SSE:\n{type:"text", content:"package main..."}
    A2AClient --> Orchestrator : StreamEvent(text)
    Orchestrator -> Converter : Convert(text)
    Converter --> Orchestrator : [TEXT_MESSAGE_CONTENT{content}]
    Orchestrator --> Gateway : eventChan ← TEXT_MESSAGE_CONTENT
    Gateway --> AGUIClient : **SSE:** {type:"TEXT_MESSAGE_CONTENT",\ncontent:"package main..."}
    AGUIClient --> React : store.appendStreamContent()
    React --> User : 气泡追加文字\n(逐字出现效果)
end

deactivate LLM

== 阶段7: 产物生成 ==

CodeAgent -> CodeAgent : parseCodeBlocks(fullResponse)\n检测到 ```go 代码块
CodeAgent -> CodeAgent : 生成Artifact:\n{type:"code", title:"main.go",\ncontent:"...", metadata:{language:"go"}}

CodeAgent --> A2AClient : A2A SSE:\n{type:"artifact", artifact:{...}}
A2AClient --> Orchestrator : StreamEvent(artifact)
Orchestrator -> Converter : Convert(artifact)
Converter -> Converter : **缓存Artifact**\n(不立即输出,等completed)

CodeAgent --> A2AClient : A2A SSE:\n{type:"status", status:"completed"}
deactivate CodeAgent
A2AClient --> Orchestrator : StreamEvent(status:completed)
deactivate A2AClient

== 阶段8: 协议转换 - Artifact→Skill ==

Orchestrator -> Converter : Convert(status:completed)
activate Converter #FF8A80

Converter --> Orchestrator : [TEXT_MESSAGE_END]
Orchestrator --> Gateway : eventChan ← TEXT_MESSAGE_END
Gateway --> AGUIClient : **SSE:** TEXT_MESSAGE_END
AGUIClient --> React : store.finishStreaming()
React --> User : 消息完成,\n移除闪烁光标

note over Converter
  **Artifact → TOOL_CALL 转换逻辑:**
  1. artifact.type == "code"
  2. 映射: "code" → "code_preview"
  3. 检查前端是否注册了该Skill ✓
  4. 构建args: {code, language, filename}
  5. 生成 TOOL_CALL 三事件序列
end note

Converter --> Orchestrator : [TOOL_CALL_START\n{toolName:"code_preview"}]
Orchestrator --> Gateway : eventChan ← TOOL_CALL_START
Gateway --> AGUIClient : **SSE:** {type:"TOOL_CALL_START",\ntoolCallId:"tc-1",\ntoolName:"code_preview"}

Converter --> Orchestrator : [TOOL_CALL_ARGS\n{content:"{code,lang,file}"}]
Orchestrator --> Gateway : eventChan ← TOOL_CALL_ARGS
Gateway --> AGUIClient : **SSE:** {type:"TOOL_CALL_ARGS",\ncontent:"{...}"}

Converter --> Orchestrator : [TOOL_CALL_END]
Orchestrator --> Gateway : eventChan ← TOOL_CALL_END
Gateway --> AGUIClient : **SSE:** TOOL_CALL_END

deactivate Converter

== 阶段9: 前端执行Skill + 完成 ==

AGUIClient --> React : 收集完整ToolCall\n解析args JSON
React -> React : 执行 code_preview Skill:\n渲染 <CodePreview\n  code="..." language="go"\n  filename="main.go" />
React --> User : 看到代码预览卡片\n(语法高亮+行号+复制按钮)

Orchestrator --> Gateway : eventChan ← RUN_FINISHED
deactivate Orchestrator
Gateway --> AGUIClient : **SSE:** {type:"RUN_FINISHED"}
AGUIClient --> React : 关闭SSE连接
deactivate AGUIClient
React --> User : 隐藏loading,\n显示完成状态
deactivate React

Gateway -> DB : INSERT INTO messages\n(保存Agent回复+artifacts)
deactivate Gateway

@enduml
```

## 四、协议转换时序图

```plantuml
@startuml AgentHub_ProtocolConversion
!theme plain
skinparam sequenceMessageAlign center

title A2A ↔ AG-UI 协议转换详细时序图

participant "A2A Stream\n(子Agent SSE)" as A2A #FCE4EC
participant "ProtocolConverter\n(转换器)" as CVT #FFF3E0
participant "ArtifactBuffer\n(产物缓存)" as BUF #E8F5E9
participant "AG-UI EventChan\n(输出到前端)" as AGUI #E3F2FD

== 文本流式阶段 ==

A2A -> CVT : {type:"status", status:"working"}
activate CVT
CVT -> CVT : messageId = uuid()
CVT -> AGUI : **{type:"TEXT_MESSAGE_START", messageId}**
deactivate CVT

loop 每个text chunk
    A2A -> CVT : {type:"text", content:"..."}
    activate CVT
    CVT -> AGUI : **{type:"TEXT_MESSAGE_CONTENT",\n messageId, content:"..."}**
    deactivate CVT
end

== 产物接收阶段 (缓存,不立即输出) ==

A2A -> CVT : {type:"artifact",\n artifact:{type:"code",\n title:"main.go", content:"..."}}
activate CVT
CVT -> BUF : **缓存** artifact[0]
deactivate CVT

A2A -> CVT : {type:"artifact",\n artifact:{type:"code",\n title:"utils.go", content:"..."}}
activate CVT
CVT -> BUF : **缓存** artifact[1]
deactivate CVT

== 完成信号 → 批量转换产物 ==

A2A -> CVT : {type:"status", status:"completed"}
activate CVT #FF8A80

CVT -> AGUI : **{type:"TEXT_MESSAGE_END", messageId}**

CVT -> BUF : 取出所有缓存artifacts
BUF --> CVT : [artifact[0], artifact[1]]

note over CVT
  **映射规则:**
  artifact.type → Skill名称
  ─────────────────────────
  "code"     → "code_preview"
  "webpage"  → "web_preview"
  "file"     → "file_download"
  "image"    → "image_preview"
  "document" → "markdown_render"
end note

loop 每个artifact (i=0,1)
    CVT -> CVT : toolCallId = uuid()\ntoolName = mapToSkill(artifact.type)\nargs = buildArgs(artifact)
    CVT -> AGUI : **{type:"TOOL_CALL_START",\n toolCallId, toolName}**
    CVT -> AGUI : **{type:"TOOL_CALL_ARGS",\n toolCallId, content: argsJSON}**
    CVT -> AGUI : **{type:"TOOL_CALL_END",\n toolCallId}**
end

CVT -> AGUI : **{type:"RUN_FINISHED"}**
deactivate CVT

@enduml
```

## 五、前端流式渲染时序图

```plantuml
@startuml AgentHub_StreamRendering
!theme plain
skinparam sequenceMessageAlign center

title 前端流式渲染时序图

participant "SSE\nEventSource" as SSE #E3F2FD
participant "useSendMessage\nHook" as Hook #E8F5E9
participant "Zustand\nMessageStore" as Store #FFF3E0
participant "React\n组件树" as UI #FCE4EC
participant "DOM\n(用户可见)" as DOM #EFEBE9

== 消息开始 ==

SSE -> Hook : TEXT_MESSAGE_START\n{messageId:"msg-1"}
activate Hook
Hook -> Store : startStreaming("conv-1", "msg-1")
activate Store
Store -> Store : streamingMessage = {\n  id:"msg-1",\n  content:"",\n  status:"streaming"\n}
Store --> UI : 状态变更通知
deactivate Store
UI -> DOM : 渲染空Agent气泡\n+ 闪烁光标动画
deactivate Hook

== 流式文本 (高频: 每50-100ms) ==

SSE -> Hook : TEXT_MESSAGE_CONTENT\n{content:"好的，"}
activate Hook
Hook -> Store : appendStreamContent("msg-1", "好的，")
activate Store
Store -> Store : streamingMessage.content\n+= "好的，"
Store --> UI : 状态变更
deactivate Store
UI -> DOM : 文本: "好的，|"
deactivate Hook

SSE -> Hook : TEXT_MESSAGE_CONTENT\n{content:"我来帮你写"}
activate Hook
Hook -> Store : appendStreamContent(...)
activate Store
Store -> Store : content += "我来帮你写"
Store --> UI : 状态变更
deactivate Store
UI -> DOM : 文本: "好的，我来帮你写|"
deactivate Hook

note over UI, DOM
  **渲染策略:**
  - streaming阶段: 纯文本 + 闪烁光标
  - 避免每次Markdown解析(性能)
  - 使用 React.memo 减少重渲染
end note

== 消息结束 ==

SSE -> Hook : TEXT_MESSAGE_END\n{messageId:"msg-1"}
activate Hook
Hook -> Store : finishStreaming("msg-1")
activate Store
Store -> Store : 1. msg.status = "sent"\n2. messages[conv].push(msg)\n3. streamingMessage = null
Store --> UI : 状态变更
deactivate Store
UI -> UI : 对完整content执行\nMarkdown解析
UI -> DOM : 渲染格式化消息\n(标题/列表/代码块样式)
deactivate Hook

== 产物Skill调用 ==

SSE -> Hook : TOOL_CALL_START\n{toolCallId:"tc-1",\n toolName:"code_preview"}
activate Hook
Hook -> Hook : currentToolCall = {\n  id:"tc-1",\n  name:"code_preview",\n  args:""\n}

SSE -> Hook : TOOL_CALL_ARGS\n{content:"{\"code\":\"...\",\n\"language\":\"go\",\n\"filename\":\"main.go\"}"}
Hook -> Hook : currentToolCall.args\n+= content

SSE -> Hook : TOOL_CALL_END\n{toolCallId:"tc-1"}
Hook -> Hook : params = JSON.parse(\ncurrentToolCall.args)
Hook -> Store : addCodeBlock(params)
activate Store
Store -> Store : 最后一条agent消息\n.codeBlocks.push(params)
Store --> UI : 状态变更
deactivate Store
UI -> DOM : 渲染 <CodePreview />\n┌─────────────────┐\n│ main.go      [复制]│\n├─────────────────┤\n│ 1│package main   │\n│ 2│import "net/http"│\n│ 3│...             │\n└─────────────────┘
deactivate Hook

== 运行结束 ==

SSE -> Hook : RUN_FINISHED
activate Hook
Hook -> Hook : 清理状态\n关闭SSE连接
UI -> DOM : 隐藏loading指示器
deactivate Hook

@enduml
```

## 六、群聊多Agent协作时序图

```plantuml
@startuml AgentHub_MultiAgent
!theme plain
skinparam sequenceMessageAlign center

title AgentHub - 群聊模式多Agent协作时序图

actor User as "👤 用户"
participant Frontend as "🖥️ 前端"
participant Gateway as "🚪 Gateway"
participant Orchestrator as "🧠 Orchestrator"
participant LLM as "🤖 LLM\n(意图分析)"
participant Converter as "🔄 Converter"
participant CodeAgent as "🔧 Code-Agent"
participant WebAgent as "🎨 Web-Agent"

== 用户发送复杂任务 ==

User -> Frontend : "帮我写一个登录页面,\n前端React + 后端Go接口"
Frontend -> Gateway : AG-UI RunRequest
Gateway -> Orchestrator : Process(req)

== 意图编排 ==

Orchestrator -> LLM : 意图分析Prompt\n+ AgentCards\n+ 用户消息
activate LLM
LLM --> Orchestrator : ExecutionPlan {\n  strategy: "parallel",\n  tasks: [\n    {agent:"web-agent",\n     task:"React登录组件"},\n    {agent:"code-agent",\n     task:"Go登录接口"}\n  ]\n}
deactivate LLM

Orchestrator --> Gateway : STATE_UPDATE\n{phase:"orchestrating",\nplan:"拆解为2个子任务"}
Gateway --> Frontend : SSE → 显示编排状态

== 并行调用子Agent ==

par 并行执行
    Orchestrator -> WebAgent : A2A sendSubscribe\n{task:"React登录组件"}
    activate WebAgent
else
    Orchestrator -> CodeAgent : A2A sendSubscribe\n{task:"Go登录接口"}
    activate CodeAgent
end

== Web-Agent 响应 ==

Orchestrator --> Gateway : STATE_UPDATE\n{activeAgent:"web-agent"}
Gateway --> Frontend : 切换显示Web-Agent头像

WebAgent --> Orchestrator : A2A: status=working
Orchestrator -> Converter : Convert → TEXT_MESSAGE_START
Converter --> Gateway : AG-UI Events
Gateway --> Frontend : SSE → 创建Web-Agent气泡

loop Web-Agent流式输出
    WebAgent --> Orchestrator : A2A: text chunks
    Orchestrator -> Converter : Convert
    Converter --> Gateway : TEXT_MESSAGE_CONTENT
    Gateway --> Frontend : SSE → 逐字渲染
end

WebAgent --> Orchestrator : A2A: artifact\n{type:"webpage", html:"..."}
WebAgent --> Orchestrator : A2A: status=completed
deactivate WebAgent

Orchestrator -> Converter : Convert(completed)
Converter --> Gateway : TEXT_MESSAGE_END\n+ TOOL_CALL(web_preview)
Gateway --> Frontend : SSE → 渲染网页预览iframe

== Code-Agent 响应 ==

Orchestrator --> Gateway : STATE_UPDATE\n{activeAgent:"code-agent"}
Gateway --> Frontend : 切换显示Code-Agent头像

CodeAgent --> Orchestrator : A2A: status=working
Orchestrator -> Converter : Convert → TEXT_MESSAGE_START
Converter --> Gateway : AG-UI Events
Gateway --> Frontend : SSE → 创建Code-Agent气泡

loop Code-Agent流式输出
    CodeAgent --> Orchestrator : A2A: text chunks
    Orchestrator -> Converter : Convert
    Converter --> Gateway : TEXT_MESSAGE_CONTENT
    Gateway --> Frontend : SSE → 逐字渲染
end

CodeAgent --> Orchestrator : A2A: artifact\n{type:"code", title:"handler.go"}
CodeAgent --> Orchestrator : A2A: status=completed
deactivate CodeAgent

Orchestrator -> Converter : Convert(completed)
Converter --> Gateway : TEXT_MESSAGE_END\n+ TOOL_CALL(code_preview)
Gateway --> Frontend : SSE → 渲染代码预览卡片

== 完成 ==

Orchestrator --> Gateway : RUN_FINISHED
Gateway --> Frontend : SSE → 完成状态
Frontend --> User : 看到两个Agent的回复:\n1. 网页预览(iframe)\n2. Go代码卡片

@enduml
```

## 七、类图 - 核心数据模型

```plantuml
@startuml AgentHub_ClassDiagram
!theme plain
skinparam classAttributeIconSize 0

title AgentHub - 核心数据模型类图

package "Gateway层" {
    class AGUIRunRequest {
        +threadId: string
        +runId: string
        +messages: []AGUIMessage
        +tools: []AGUITool
    }
    
    class AGUIEvent {
        +type: string
        +runId: string
        +messageId: string
        +content: string
        +toolCallId: string
        +toolName: string
        +error: string
    }
    
    class Conversation {
        +id: UUID
        +title: string
        +agentName: string
        +createdAt: DateTime
        +updatedAt: DateTime
    }
    
    class Message {
        +id: UUID
        +conversationId: UUID
        +senderType: "user"|"agent"
        +senderName: string
        +content: string
        +artifacts: JSON
        +aguiRunId: string
        +createdAt: DateTime
    }
}

package "Orchestrator层" {
    class Orchestrator {
        -a2aClient: A2AClient
        -agents: map[string]AgentConfig
        +Process(ctx, req, history, events)
    }
    
    class ProtocolConverter {
        -messageID: string
        -artifactBuffer: []A2AArtifact
        -frontendSkills: []string
        +Convert(event): []AGUIEvent
        -flushArtifacts(): []AGUIEvent
        -mapArtifactToSkill(type): string
        -buildToolArgs(artifact): map
    }
    
    class ExecutionPlan {
        +intent: string
        +strategy: "single"|"parallel"|"sequential"
        +tasks: []TaskPlan
    }
    
    class TaskPlan {
        +agentName: string
        +taskContent: string
        +dependsOn: []string
        +priority: int
    }
}

package "A2A层" {
    class A2AClient {
        +SendSubscribe(ctx, url, msgs): <-chan StreamEvent
    }
    
    class A2AStreamEvent {
        +type: "status"|"text"|"artifact"
        +status: string
        +content: string
        +artifact: A2AArtifact
        +error: string
    }
    
    class A2AArtifact {
        +type: "code"|"webpage"|"file"|"image"
        +title: string
        +content: string
        +metadata: map[string]string
    }
    
    class AgentCard {
        +name: string
        +description: string
        +url: string
        +version: string
        +capabilities: Capabilities
        +skills: []Skill
        +inputModes: []string
        +outputModes: []string
    }
}

package "ADK Runtime" {
    class Agent {
        -config: AgentConfig
        -handler: HandlerFunc
        -tools: map[string]ToolFunc
        +SetHandler(fn)
        +RegisterTool(name, fn)
        +Serve(addr)
    }
    
    class Context {
        -writer: ResponseWriter
        -flusher: Flusher
        -artifacts: []Artifact
        +StreamText(text)
        +AddArtifact(artifact)
        +LLM(): LLMClient
    }
    
    class Task {
        +id: string
        +messages: []TaskMessage
        +status: TaskStatus
    }
}

' 关系
Orchestrator --> ProtocolConverter : 使用
Orchestrator --> A2AClient : 调用
Orchestrator --> ExecutionPlan : 生成
ExecutionPlan *-- TaskPlan
A2AClient --> A2AStreamEvent : 接收
A2AStreamEvent --> A2AArtifact : 包含
ProtocolConverter --> AGUIEvent : 输出
Agent --> Context : 创建
Agent --> Task : 处理
Agent --> AgentCard : 暴露

@enduml
```

## 八、状态图 - 消息生命周期

```plantuml
@startuml AgentHub_MessageState
!theme plain

title 消息状态机

[*] --> Sending : 用户点击发送

state "Sending" as Sending #E3F2FD {
    Sending : 前端乐观更新
    Sending : 显示用户消息气泡
}

Sending --> Sent : Gateway保存成功
Sending --> Failed : 网络错误

state "Agent处理中" as AgentProcessing #FFF3E0 {
    state "Streaming" as Streaming {
        Streaming : 逐字接收Agent回复
        Streaming : 实时渲染到气泡
    }
}

Sent --> AgentProcessing : AG-UI RUN_STARTED

AgentProcessing --> AgentSent : TEXT_MESSAGE_END
AgentProcessing --> Failed : RUN_ERROR

state "AgentSent" as AgentSent #E8F5E9 {
    AgentSent : Agent回复完成
    AgentSent : Markdown格式化
    AgentSent : 产物卡片渲染
}

state "Failed" as Failed #FFCDD2 {
    Failed : 显示错误提示
    Failed : 可重试
}

AgentSent --> [*]
Failed --> Sending : 用户重试

@enduml
```

## 九、活动图 - AG-UI Run 处理流程

```plantuml
@startuml AgentHub_Activity
!theme plain

title AG-UI Run 请求处理流程 (Gateway + Orchestrator)

start

:接收 POST /api/agui/run;

:Token鉴权验证;
if (鉴权通过?) then (是)
else (否)
    :返回 401 Unauthorized;
    stop
endif

:解析 RunRequest\n(threadId, messages, tools);

:保存用户消息到MySQL;

:查询会话历史消息\n(最近20条);

:设置SSE响应头\nContent-Type: text/event-stream;

:创建 eventChan;

fork
    :异步启动 Orchestrator.Process();
    
    :初始化 ProtocolConverter\n(注入前端Skills列表);
    
    :发送 RUN_STARTED;
    
    if (用户@了指定Agent?) then (是)
        :直接路由到指定Agent;
    else (否)
        :LLM意图分析;
        :生成ExecutionPlan;
    endif
    
    :通过A2A调用子Agent\n(sendSubscribe);
    
    repeat
        :接收A2A StreamEvent;
        :ProtocolConverter.Convert();
        :输出AG-UI Events到eventChan;
    repeat while (Agent未完成?) is (是)
    
    :flushArtifacts()\n产物→TOOL_CALL转换;
    
    :发送 RUN_FINISHED;
    
    :保存Agent回复到MySQL;
    
fork again
    repeat
        :从eventChan读取事件;
        :SSE写入响应流\ndata: {event JSON};
        :Flush;
    repeat while (eventChan未关闭?) is (是)
end fork

:关闭SSE连接;

stop

@enduml
```
