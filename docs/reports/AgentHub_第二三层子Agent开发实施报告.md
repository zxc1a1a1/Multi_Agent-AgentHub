# AgentHub 第二层与第三层子 Agent 开发实施报告

> 版本：v1.0  
> 适用范围：暂不展开第一层编排控制层，优先完成第二层“产品可见 Agent”和第三层“内部能力 Agent”的代码骨架、运行方式、职责边界、Artifact 输出和测试策略。  
> 目标周期：2 人协作，1–2 天完成“可编译、可启动、可被 A2A 调用、可产出基础 Artifact”的第一版；生产级工具权限、真实文件读写、真实部署、完整 Orchestrator 编排后续再做。  
> 重要边界：本报告基于 Codex 对当前 `agents/adk` 与 `agents/code-agent` 的源码事实分析，不再假设 ADK 中存在没有实现的 API。

---

## 1. 当前结论

当前阶段不优先做第一层编排控制层：

```text
第一层：编排控制层
- Orchestrator / 统一 Agent
- Planner / Intent Engine
- Registry / AgentCard / HealthCheck
```

当前优先做第二层和第三层：

```text
第二层：产品可见 Agent
- Code Agent
- Web Agent
- Document / Markdown Agent
- Custom Agent

第三层：内部工作流 Agent / 能力 Agent
- Context Agent
- Codebase Explorer Agent
- Diff Agent
- Review Agent
- Test Agent
- Artifact Agent
- Security Agent
- Deploy Agent
- Version Agent
- Agent Builder Agent
- File Agent
- Web Research Agent
- QA Acceptance Agent
- PPT Agent
- Release Agent
```

但是要注意一个现实判断：

```text
1–2 天内可以完成：
- 统一模板
- ADK 最小通用化补丁
- 所有 Agent 的目录、config、main、handler、handler_test、Dockerfile
- 第二层核心 Agent 的基础可用版本
- 第三层内部 Agent 的“LLM + 结构化 Artifact”第一版
- go test / go build 基线验证

1–2 天内不应该承诺完成：
- 真实 Shell 工具执行
- 真实文件系统读写
- 真实部署
- 真实网络抓取
- 完整 Registry / Orchestrator / Intent Engine
- 完整多 Agent 自动编排闭环
```

所以本次开发的正确目标是：

> 先把所有子 Agent 按当前 ADK 运行时标准化出来，保证每个 Agent 都能作为独立 A2A 服务运行，并能输出稳定 Artifact。后续再接 Registry / Orchestrator / Intent Engine 进行调度。

---

## 2. 文档与外部资料依据

### 2.1 项目内 5 个文档的使用方式

| 文档 | 在本报告中的作用 |
|---|---|
| `agenthub-skills-usage-report-v3.md` | 确定新增 Child Agent 应使用 `adk-runtime-contract`、`a2a-agent-contract`，并受 Artifact、安全、测试、观测、代码风格等契约约束。 |
| `SPRINT-v1.0-Plan.md` | 确定 v1.0 的近期目标：2+ Agent、web_preview、markdown、群聊、结构化消息、Docker Demo、降级重试。 |
| `UML-AgentHub系统图.md` | 确定系统链路：Frontend → AG-UI → Gateway → Orchestrator → A2A → Child Agent → Artifact → Frontend Runtime Skill。 |
| `AgentHub- 多Agent协作平台设计.pdf` | 对齐赛题目标：IM 聊天式交互、多 Agent 群聊协作、上下文连续、产物内联、AI 协作记录、可运行 Demo。 |
| `多Agent聊天式工作台_子Agent设计与开发建议报告.md` | 明确子 Agent 应围绕任务执行链路划分，不应把 UI 功能拟人化成 Agent。 |

### 2.2 Codex ADK 源码事实报告的关键结论

Codex 已读取并分析：

```text
agents/adk/agent.go
agents/adk/context.go
agents/adk/llm.go
agents/adk/server.go
agents/adk/types.go
agents/code-agent/main.go
agents/code-agent/handler.go
agents/code-agent/handler_test.go
agents/code-agent/config.yaml
agents/code-agent/Dockerfile
```

关键事实如下：

1. `agents/adk/agent.go` 当前主要提供 `LoadConfig(path string) (*AgentConfig, error)`，通过 YAML 加载 `AgentConfig`。
2. `agents/adk/context.go` 中 `Context` 已提供：
   - `StreamText(chunk string)`
   - `AddArtifact(artifact Artifact)`
   - `Context() context.Context`
   - `Done() <-chan struct{}`
   - `Cancel()`
3. `Context.StreamText` 会通过 A2A artifact event 输出文本片段。
4. `Context.AddArtifact` 先把产物追加到内存，后续由 `ExecuteHandler` 转成 A2A artifact event。
5. `agents/adk/llm.go` 已有 `LLMClient`，支持 Anthropic、OpenAI、OpenAI-compatible baseURL，且已有 120s timeout 和复用 `http.Client`。
6. `StreamCompletion` 真实签名为：

```go
func (l *LLMClient) StreamCompletion(
    ctx context.Context,
    systemPrompt string,
    messages []LLMMessage,
    onChunk func(text string),
) error
```

7. `agents/adk/server.go` 提供 `NewA2AServer(config *AgentConfig, handler TaskHandler) *A2AServer` 和 `Run(addr string) error`。
8. 当前 A2A Server 暴露：
   - `GET /health`
   - `GET /.well-known/agent.json`
   - `POST /`
   - `POST /a2a/tasks/sendSubscribe`
9. `agents/adk/types.go` 当前有：
   - `AgentConfig{Name, Description, Version, URL, Skills, InputModes, OutputModes, Streaming}`
   - `Artifact{Type, Title, Content, Metadata map[string]string}`
10. 当前 ADK 缺少：
   - AgentCard 从 config 动态完整映射的能力
   - 通用 fenced block parser
   - JSON schema validation
   - structured output helper
   - tool permission interface
   - tool sandbox
   - trace/run/task id logging
   - fake LLMClient 测试注入
   - contentRef / version / artifact lifecycle
   - graceful shutdown

### 2.3 外部高质量资料的设计启发

| 来源 | 对本报告的启发 |
|---|---|
| OpenAI Agents SDK | 多 Agent 编排可以混合“LLM 决策”和“代码编排”，因此当前先把子 Agent 标准化，后续 Orchestrator 再决定调用顺序。 |
| LangChain / LangGraph Multi-agent | 常见模式包括 supervisor 调度 subagents、handoff、router。AgentHub 当前更适合先采用“统一 Orchestrator 调用专业子 Agent”的方式。 |
| Anthropic Building Effective Agents | 推荐 routing、parallelization、orchestrator-workers、evaluator-optimizer 等工作流；这支持把 Review/Test/Security 设计成内部评估型 Agent。 |
| Google A2A / Agent2Agent | A2A 用于跨 Agent 通信和协作，适合当前 AgentHub 的 Child Agent 接入层。 |
| MCP | MCP 更适合外部工具和数据源接入。AgentHub 后续可让 File/WebResearch/Deploy 这类 Agent 通过受控工具层访问外部资源，而不是直接裸调用系统能力。 |

---

## 3. 当前 ADK 到底是什么

当前 `agents/adk` 不是完整的商用 Agent Runtime，而是一个“最小可运行 Child Agent 基座”。

它已经支持：

```text
1. 从 config.yaml 加载 AgentConfig。
2. 创建 A2A Server。
3. 暴露健康检查和 AgentCard 端点。
4. 接收 A2A task。
5. 给 handler 注入 Runtime Context。
6. handler 可以 ctx.StreamText 流式输出文本。
7. handler 可以 ctx.AddArtifact 输出产物。
8. handler 可以通过 LLMClient 调 Anthropic / OpenAI。
9. handler 返回错误时，ADK 会用安全文案失败。
```

它还不支持：

```text
1. 复杂工具权限。
2. shell / file / network 工具沙箱。
3. 结构化 JSON Schema 校验。
4. Artifact 版本、contentRef、生命周期。
5. 统一 trace / metrics / debug dump。
6. 完整动态 AgentCard 能力声明。
7. Agent 模板生成器。
8. Orchestrator 编排能力。
```

所以当前新增子 Agent 的正确做法是：

```text
把每个 Agent 做成：
A2A Server + LLMClient + Handler + StreamText + AddArtifact + 单测

不要在 Agent 内部做：
Gateway 逻辑
Orchestrator 逻辑
AG-UI 事件生成
数据库持久化
真实部署
任意文件读写
任意命令执行
```

---

## 4. 开发前必须先做的 ADK 最小补丁

虽然当前 ADK 可以跑 code-agent，但要快速复制出多个 Agent，建议先做 3 个小补丁。这些不是第一层 Orchestrator，而是 Child Agent Runtime 的基础完善。

### 4.1 补丁 A：AgentCard 动态映射

Codex 报告指出当前 `NewA2AServer` 中 AgentCard / skills / modes 仍存在硬编码风险。新增多个 Agent 时，如果 `/.well-known/agent.json` 还显示 code-agent 能力，会直接影响后续 Registry / Orchestrator 判断。

建议修改：

```text
agents/adk/server.go
```

目标：

```text
AgentCard 从 AgentConfig 动态生成：
- Name
- Description
- Version
- URL
- Skills
- InputModes
- OutputModes
- Streaming
```

当前 `AgentConfig.Skills` 是 `[]string`，第一版可以先映射成简单 skill ID；不要现在强行做复杂 Skill object。

验收：

```text
curl http://localhost:8082/.well-known/agent.json
```

对 web-agent 应返回 `web-agent` 和 `web_generation`，不能还返回 `code-agent`。

### 4.2 补丁 B：抽通用 fenced block parser

当前 `parseCodeBlocks` 已在 code-agent 中经过测试，但 web-agent、document-agent、diff-agent 都会需要类似逻辑。

建议新增：

```text
agents/adk/fenced_blocks.go
agents/adk/fenced_blocks_test.go
```

提供：

```go
type FencedBlock struct {
    Language string
    Filename string
    Content  string
}

func ParseFencedBlocks(text string) []FencedBlock
func IsValidFenceLanguage(language string) bool
func DefaultFilename(language string) string
```

注意：

```text
1. 这不是必须先做，但做了会大幅减少复制粘贴。
2. 解析规则应沿用 code-agent 当前稳定规则。
3. 独立成行的 ``` 才闭合 block。
4. block 内不支持未转义且独立成行的 ```。
```

### 4.3 补丁 C：Agent handler 测试假 LLM 接口

当前 code-agent 的核心解析函数可测，但完整 handler 难测，因为 `LLMClient` 是具体类型。

建议后续将 Agent handler 内部依赖改成接口：

```go
type StreamCompleter interface {
    StreamCompletion(
        ctx context.Context,
        systemPrompt string,
        messages []adk.LLMMessage,
        onChunk func(text string),
    ) error
}
```

这样每个 Agent 都可以用 fake LLM 测试：

```text
输入 A2A messages
fake LLM 输出固定 fenced block
断言 ctx.StreamText / ctx.AddArtifact 行为
```

1–2 天内如果来不及，可以先不做这个补丁，只测试 parser。

---

## 5. 统一子 Agent 目录模板

所有第二层和第三层 Agent 都按下面结构创建：

```text
agents/{agent-name}/
  main.go
  handler.go
  handler_test.go
  config.yaml
  Dockerfile
```

不建议每个 Agent 自创结构。

### 5.1 main.go 模板职责

从 code-agent 抽象：

```text
1. adk.LoadConfig("config.yaml")
2. 读取端口环境变量
3. llm := adk.NewLLMClient()
4. taskHandler := handleTaskWithLLM(llm)
5. server := adk.NewA2AServer(config, taskHandler)
6. server.Run(":" + port)
```

### 5.2 handler.go 模板职责

```text
1. 定义 systemPrompt。
2. 从 []a2a.Message 中提取 text。
3. 转成 []adk.LLMMessage。
4. 调用 llm.StreamCompletion。
5. 在 onChunk 中：
   - fullResponse.WriteString(chunk)
   - ctx.StreamText(chunk)
6. LLM 完成后解析 fullResponse。
7. 根据解析结果 ctx.AddArtifact。
8. 返回错误时只返回安全错误，不拼接密钥、body、内部堆栈。
```

### 5.3 handler_test.go 模板职责

每个 Agent 第一版至少测试：

```text
1. 标准 fenced block。
2. 无 filename 的默认文件名。
3. 多 block。
4. 无 block。
5. 非法 language/header。
6. 普通反引号不误闭合。
7. Artifact Type / Title / Metadata 是否正确。
```

### 5.4 config.yaml 模板职责

第一版使用当前 ADK 已支持字段：

```yaml
name: xxx-agent
description: One sentence.
version: "0.1.0"
url: "http://localhost:xxxx"
skills:
  - xxx_skill
inputModes:
  - text
outputModes:
  - text
  - xxx_artifact
streaming: true
```

不要写：

```text
API Key
Token
DB Password
DATABASE_URL
system prompt 全文
内部敏感 URL
```

### 5.5 Dockerfile 模板职责

从 code-agent Dockerfile 复制，替换：

```text
COPY {agent-name}/ ./{agent-name}/
go build -o /{agent-name} .
COPY config.yaml
EXPOSE {port}
CMD ["/{agent-name}"]
```

---

## 6. 第二层：产品可见 Agent 设计

第二层 Agent 是用户可以在产品里感知、选择、@ 或单聊的 Agent。

---

### 6.1 Code Agent

#### 当前状态

已有 `agents/code-agent`，是当前样板。

#### 定位

代码生成、代码解释、轻量重构建议、代码产物输出。

#### 当前已经具备

```text
- LoadConfig
- LLMClient 启动时单例
- A2A Server
- StreamText
- parseCodeBlocks
- code artifact
- handler_test 覆盖主要解析场景
```

#### 1–2 天内建议只做小增强

```text
1. 将 parseCodeBlocks 抽到 adk/fenced_blocks.go，或暂时保留。
2. 增加 diff fenced block 的识别，但不强制。
3. config.yaml 保持 code_generation/code_review/code_refactoring。
4. 不增加真实写文件能力。
```

#### Artifact

```text
Type: code
Title: filename
Content: code
Metadata:
  language
```

---

### 6.2 Web Agent

#### 定位

生成网页、HTML/CSS/JS 页面、UI mock、landing page、login page、dashboard preview。

#### 目录

```text
agents/web-agent/
  main.go
  handler.go
  handler_test.go
  config.yaml
  Dockerfile
```

#### config.yaml

```yaml
name: web-agent
description: Generates self-contained HTML/CSS/JS pages and UI previews.
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

#### handler.go 实现

System prompt 要求 LLM 输出：

````markdown
```html:index.html
<!DOCTYPE html>
<html>
<head>
  <style>
  ...
  </style>
</head>
<body>
  ...
  <script>
  ...
  </script>
</body>
</html>
```
````

解析规则：

```text
1. 识别 language=html 的 fenced block。
2. filename 为空则默认 index.html。
3. Content 是完整 HTML。
4. 可用正则提取 <style>...</style> 到 metadata["css"]。
5. 可用正则提取 <script>...</script> 到 metadata["js"]。
6. ctx.AddArtifact(Type="webpage")。
```

#### Artifact

```text
Type: webpage
Title: index.html
Content: full HTML
Metadata:
  language=html
  css=...
  js=...
```

#### 测试重点

```text
- html:index.html 标准块
- ```html 无 filename 时默认 index.html
- 提取 css
- 提取 js
- 多个 html block
- 无 html block
- 非法语言不输出 artifact
```

#### 不负责

```text
- 不生成 AG-UI web_preview 事件
- 不渲染 iframe
- 不读写前端文件
- 不部署网页
```

---

### 6.3 Document / Markdown Agent

#### 定位

生成 Markdown 文档、README、技术方案、PRD、交接报告、Demo 脚本。

#### config.yaml

```yaml
name: document-agent
description: Writes Markdown documents, README files, technical reports, PRDs, and handoff notes.
version: "0.1.0"
url: "http://localhost:8083"
skills:
  - markdown_document
  - technical_report
  - prd_writer
  - handoff_report
inputModes:
  - text
outputModes:
  - text
  - document
streaming: true
```

#### handler.go 实现

要求 LLM 输出：

````markdown
```markdown:report.md
# Title

...
```
````

解析：

```text
1. 识别 language=markdown 或 md。
2. filename 为空默认 untitled.md。
3. ctx.AddArtifact(Type="document")。
```

#### Artifact

```text
Type: document
Title: report.md
Content: markdown
Metadata:
  format=markdown
```

#### 测试重点

```text
- markdown:xxx.md
- md:xxx.md
- 无 filename 默认 untitled.md
- 多文档
- 无文档
```

---

### 6.4 Custom Agent

#### 定位

用户自建 Agent 的运行模板。当前 1–2 天内不做完整可视化创建，只做可配置 profile 的基础运行模式。

#### 目录建议

```text
agents/custom-agent/
  main.go
  handler.go
  handler_test.go
  config.yaml
  Dockerfile
  profiles/
    example.yaml
```

#### config.yaml

```yaml
name: custom-agent
description: Runs a user-defined agent profile with restricted artifact output.
version: "0.1.0"
url: "http://localhost:8084"
skills:
  - custom_profile_execution
inputModes:
  - text
outputModes:
  - text
  - document
  - code
streaming: true
```

#### profile 示例

```yaml
id: prd-writer
displayName: PRD Writer
description: Writes product requirement documents.
systemPrompt: |
  You are a product requirement document assistant.
allowedArtifactTypes:
  - document
riskLevel: low
```

#### 安全要求

```text
1. 用户自定义 prompt 不能覆盖平台安全规则。
2. profile 不允许写真实 secret。
3. 默认只允许 document artifact。
4. code/webpage 输出要显式白名单。
5. 不允许执行命令、读写文件、访问网络。
```

#### 第一版建议

如果时间紧，Custom Agent 只做设计文件和 skeleton，不接入真实 profile loader。

---

## 7. 第三层：内部工作流 / 能力 Agent 设计

第三层 Agent 第一版也按 A2A 服务实现，但默认不作为用户联系人重点展示。它们后续由 Orchestrator 调用。

---

### 7.1 Context Agent

#### 定位

把聊天历史、pin 消息、artifact 摘要整理成可给其他 Agent 使用的 context bundle。

#### config.yaml

```yaml
name: context-agent
description: Builds compact context bundles for downstream agents.
version: "0.1.0"
url: "http://localhost:8091"
skills:
  - context_compression
  - context_selection
inputModes:
  - text
outputModes:
  - text
  - context_bundle
streaming: true
```

#### 输入第一版

通过 text 传 JSON：

```json
{
  "latestUserMessage": "...",
  "history": [],
  "pinnedMessages": [],
  "artifacts": [],
  "targetAgent": "code-agent"
}
```

#### 输出 Artifact

```text
Type: context_bundle
Title: context-bundle.json
Content:
{
  "summary": "...",
  "requirements": [],
  "constraints": [],
  "relevantMessages": [],
  "relevantArtifacts": []
}
```

#### 不负责

```text
- 不写代码
- 不运行测试
- 不部署
- 不直接读取数据库
```

---

### 7.2 Codebase Explorer Agent

#### 定位

只读分析代码库结构，定位相关文件和模块。

#### 当前 1–2 天内的现实做法

不要让它直接读本机文件系统。第一版只分析调用方提供的 file tree / snippets。

#### config.yaml

```yaml
name: codebase-explorer-agent
description: Analyzes provided file trees and code snippets to locate relevant files.
version: "0.1.0"
url: "http://localhost:8092"
skills:
  - codebase_analysis
  - file_location
inputModes:
  - text
outputModes:
  - text
  - codebase_analysis
streaming: true
```

#### 输出 Artifact

```text
Type: codebase_analysis
Title: codebase-analysis.json
Content:
{
  "targetFiles": [],
  "architectureNotes": [],
  "risks": [],
  "recommendedNextAgent": "code-agent"
}
```

#### 后续才允许

```text
- list_directory
- read_file
- grep/search
```

而且必须经过 Security / Tool Permission。

---

### 7.3 Diff Agent

#### 定位

生成、解释、规范化 unified diff；第一版不自动 apply。

#### config.yaml

```yaml
name: diff-agent
description: Generates and reviews unified diffs without applying them.
version: "0.1.0"
url: "http://localhost:8093"
skills:
  - diff_generation
  - diff_explanation
  - patch_risk_summary
inputModes:
  - text
outputModes:
  - text
  - diff
streaming: true
```

#### 输出 Artifact

```text
Type: diff
Title: patch.diff
Metadata:
  riskLevel
  applyStatus=not_applied
```

#### 第一版限制

```text
- 不执行 git apply
- 不执行 git checkout
- 不写文件
```

---

### 7.4 Review Agent

#### 定位

根据需求、diff、代码片段、测试结果做 Review，输出审查报告。

#### config.yaml

```yaml
name: review-agent
description: Reviews code changes against requirements, quality, and safety criteria.
version: "0.1.0"
url: "http://localhost:8094"
skills:
  - code_review
  - requirement_coverage_review
  - risk_review
inputModes:
  - text
outputModes:
  - text
  - review_report
streaming: true
```

#### 输出 Artifact

```text
Type: review_report
Title: review-report.json
Content:
{
  "status": "approved|changes_requested|blocked",
  "findings": [],
  "passedChecks": []
}
```

#### 不负责

```text
- 不直接改代码
- 不运行命令
- 不应用 patch
```

---

### 7.5 Test Agent

#### 定位

测试日志分析、命令建议、失败原因解释。第一版不直接执行命令。

#### config.yaml

```yaml
name: test-agent
description: Analyzes test/build logs and produces test reports.
version: "0.1.0"
url: "http://localhost:8095"
skills:
  - test_log_analysis
  - build_failure_analysis
  - test_plan_generation
inputModes:
  - text
outputModes:
  - text
  - test_report
streaming: true
```

#### 输出 Artifact

```text
Type: test_report
Title: test-report.json
Content:
{
  "status": "passed|failed|not_run",
  "commands": [],
  "failureAnalysis": [],
  "recommendedFix": ""
}
```

#### 后续工具白名单

```text
go test ./...
go build ./cmd/server
go build ./code-agent
npm test -- --run
npm run build
bash -n smoke-test.sh
docker compose config
```

第一版只写到报告里，不执行。

---

### 7.6 Artifact Agent

#### 定位

把 raw artifacts 规整成统一 artifact manifest。

#### config.yaml

```yaml
name: artifact-agent
description: Normalizes raw agent outputs into artifact manifests.
version: "0.1.0"
url: "http://localhost:8096"
skills:
  - artifact_manifest
  - artifact_classification
inputModes:
  - text
outputModes:
  - text
  - artifact_manifest
streaming: true
```

#### 输出 Artifact

```text
Type: artifact_manifest
Title: artifact-manifest.json
Content:
{
  "artifacts": [
    {
      "type": "code|webpage|document|diff|test_report",
      "title": "...",
      "previewType": "...",
      "metadata": {}
    }
  ]
}
```

---

### 7.7 Security Agent

#### 定位

扫描 secret、危险命令、权限越界、可疑输出。第一版规则扫描 + LLM 解释，不执行外部工具。

#### config.yaml

```yaml
name: security-agent
description: Reviews agent outputs, diffs, and commands for security risks.
version: "0.1.0"
url: "http://localhost:8097"
skills:
  - secret_scan
  - command_risk_review
  - permission_review
inputModes:
  - text
outputModes:
  - text
  - security_report
streaming: true
```

#### 必须内置规则

```text
阻断或高危：
- .env
- OPENAI_API_KEY
- ANTHROPIC_API_KEY
- AGENTHUB_API_TOKEN
- DB_PASSWORD
- DATABASE_URL
- PRIVATE KEY
- BEGIN RSA
- sk-[A-Za-z0-9_-]{20,}
- rm -rf
- git push
- git reset --hard
- docker system prune
- production deploy
```

#### 允许的占位

```text
${AGENTHUB_API_TOKEN}
${MYSQL_ROOT_PASSWORD}
test-token
<本地测试token>
```

#### 输出 Artifact

```text
Type: security_report
Title: security-report.json
Content:
{
  "riskLevel": "low|medium|high|critical",
  "blocked": true,
  "reasons": [],
  "safeAlternatives": []
}
```

---

### 7.8 Deploy Agent

#### 定位

部署计划、部署状态说明、本地 Docker 验证指令生成。第一版不真实部署。

#### config.yaml

```yaml
name: deploy-agent
description: Plans deployment steps and summarizes deployment status without production deployment.
version: "0.1.0"
url: "http://localhost:8098"
skills:
  - deployment_plan
  - docker_validation_plan
  - deployment_status_summary
inputModes:
  - text
outputModes:
  - text
  - deploy_status
streaming: true
```

#### 输出 Artifact

```text
Type: deploy_status
Title: deploy-status.json
Content:
{
  "deploymentId": "dep_xxx",
  "status": "planned|blocked|failed|success",
  "target": "local|preview|production",
  "requiresConfirmation": true,
  "logsSummary": ""
}
```

#### 安全限制

```text
- 不生产部署
- 不读取 .env
- 不上传文件
- 不写云平台 token
- 不执行 docker 命令
```

---

### 7.9 Version Agent

#### 定位

生成版本摘要、patch 关系、回滚计划。第一版不执行 Git 写操作。

#### config.yaml

```yaml
name: version-agent
description: Produces version summaries, patch relationships, and rollback plans.
version: "0.1.0"
url: "http://localhost:8099"
skills:
  - version_summary
  - rollback_plan
  - patch_version_linking
inputModes:
  - text
outputModes:
  - text
  - version_snapshot
streaming: true
```

#### 输出 Artifact

```text
Type: version_snapshot
Title: version-snapshot.json
```

---

### 7.10 Agent Builder Agent

#### 定位

根据用户描述生成自定义 Agent profile、config.yaml 草稿和脚手架建议。

#### config.yaml

```yaml
name: agent-builder-agent
description: Designs custom agent profiles and scaffold suggestions.
version: "0.1.0"
url: "http://localhost:8100"
skills:
  - agent_profile_design
  - agent_scaffold_plan
inputModes:
  - text
outputModes:
  - text
  - agent_profile
streaming: true
```

#### 输出 Artifact

```text
Type: agent_profile
Title: agent-profile.yaml
```

#### 不负责

```text
- 不直接写文件
- 不直接创建目录
- 不直接改 .env
```

---

### 7.11 File Agent

#### 定位

处理调用方提供的文件文本、Markdown、JSON、代码片段，生成文件摘要。第一版不解析 PDF / DOCX / ZIP / 图片。

#### config.yaml

```yaml
name: file-agent
description: Summarizes provided file content and extracts requirements.
version: "0.1.0"
url: "http://localhost:8101"
skills:
  - file_summary
  - requirement_extraction
inputModes:
  - text
outputModes:
  - text
  - file_summary
streaming: true
```

#### 输出 Artifact

```text
Type: file_summary
Title: file-summary.json
```

---

### 7.12 Web Research Agent

#### 定位

处理调用方提供的 URL、网页标题、正文摘录，生成带来源的摘要。第一版不主动联网。

#### config.yaml

```yaml
name: web-research-agent
description: Summarizes provided web content and preserves citations.
version: "0.1.0"
url: "http://localhost:8102"
skills:
  - web_summary
  - citation_summary
inputModes:
  - text
outputModes:
  - text
  - web_research
streaming: true
```

#### 输出 Artifact

```text
Type: web_research
Title: web-research.json
```

#### 第一版限制

```text
- 不 fetch URL
- 不访问内网
- 不访问 file://
- 不发送用户 secret
```

---

### 7.13 QA Acceptance Agent

#### 定位

根据原始需求、计划、产物、测试结果判断是否达到验收标准。

#### config.yaml

```yaml
name: qa-acceptance-agent
description: Checks whether outputs satisfy acceptance criteria.
version: "0.1.0"
url: "http://localhost:8103"
skills:
  - acceptance_check
  - demo_checklist
inputModes:
  - text
outputModes:
  - text
  - acceptance_report
streaming: true
```

#### 输出 Artifact

```text
Type: acceptance_report
Title: acceptance-report.json
```

---

### 7.14 PPT Agent

#### 定位

生成演示文稿大纲和 slide JSON，不直接生成 pptx。

#### config.yaml

```yaml
name: ppt-agent
description: Generates presentation outlines and slide JSON.
version: "0.1.0"
url: "http://localhost:8104"
skills:
  - slide_outline
  - presentation_json
inputModes:
  - text
outputModes:
  - text
  - presentation
streaming: true
```

#### 输出 Artifact

```text
Type: presentation
Title: slides.json
```

---

### 7.15 Release Agent

#### 定位

生成 changelog、release notes、交付说明。第一版不执行发布命令。

#### config.yaml

```yaml
name: release-agent
description: Produces changelogs, release notes, and delivery summaries.
version: "0.1.0"
url: "http://localhost:8105"
skills:
  - changelog_generation
  - release_note
  - delivery_summary
inputModes:
  - text
outputModes:
  - text
  - release_note
streaming: true
```

#### 输出 Artifact

```text
Type: release_note
Title: release-note.md
```

---

## 8. 端口规划

| Agent | Port | Env |
|---|---:|---|
| code-agent | 8081 | `CODE_AGENT_PORT` |
| web-agent | 8082 | `WEB_AGENT_PORT` |
| document-agent | 8083 | `DOCUMENT_AGENT_PORT` |
| custom-agent | 8084 | `CUSTOM_AGENT_PORT` |
| context-agent | 8091 | `CONTEXT_AGENT_PORT` |
| codebase-explorer-agent | 8092 | `EXPLORER_AGENT_PORT` |
| diff-agent | 8093 | `DIFF_AGENT_PORT` |
| review-agent | 8094 | `REVIEW_AGENT_PORT` |
| test-agent | 8095 | `TEST_AGENT_PORT` |
| artifact-agent | 8096 | `ARTIFACT_AGENT_PORT` |
| security-agent | 8097 | `SECURITY_AGENT_PORT` |
| deploy-agent | 8098 | `DEPLOY_AGENT_PORT` |
| version-agent | 8099 | `VERSION_AGENT_PORT` |
| agent-builder-agent | 8100 | `AGENT_BUILDER_AGENT_PORT` |
| file-agent | 8101 | `FILE_AGENT_PORT` |
| web-research-agent | 8102 | `WEB_RESEARCH_AGENT_PORT` |
| qa-acceptance-agent | 8103 | `QA_ACCEPTANCE_AGENT_PORT` |
| ppt-agent | 8104 | `PPT_AGENT_PORT` |
| release-agent | 8105 | `RELEASE_AGENT_PORT` |

---

## 9. 两人 1–2 天协作计划

### 总体分工

| 人员 | 主责 |
|---|---|
| A：后端 / Agent Runtime | ADK 最小补丁、模板、第二层 Agent、关键内部 Agent 的 handler。 |
| B：Agent 测试 / 集成 / 文档 | handler_test、config/Dockerfile 批量补齐、agents README、验证命令、安全扫描、后续给 Orchestrator 的接入说明。 |

如果你们一个人更熟 Go，一个人更熟前端，也没关系。当前这批主要是 Go Agent，B 可以承担测试、脚手架、文档、配置和验证，避免 A 被大量重复文件拖慢。

---

### Day 1 上午：ADK 最小通用化 + 模板

#### A 做

```text
1. 修改 agents/adk/server.go：
   - AgentCard 从 AgentConfig 动态映射。
   - 确认 /.well-known/agent.json 对新 Agent 返回正确 name/skills/modes。

2. 可选新增：
   - agents/adk/fenced_blocks.go
   - agents/adk/fenced_blocks_test.go

3. 基于 code-agent 提炼模板。
```

#### B 做

```text
1. 新建 agents/README.md 草稿。
2. 准备端口规划表。
3. 准备每个 Agent 的 config.yaml 草稿。
4. 写 ADK 补丁对应测试或至少手动验证清单。
```

#### 验收

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub\agents"
go test ./...
go build ./code-agent
```

---

### Day 1 下午：第二层 Agent

#### A 做

```text
1. 实现 agents/web-agent。
2. 实现 agents/document-agent。
3. 保留 code-agent 不做大改。
```

#### B 做

```text
1. 为 web-agent 写 handler_test.go。
2. 为 document-agent 写 handler_test.go。
3. 补 Dockerfile。
4. 更新 agents/README.md。
```

#### 验收

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub\agents"

go test ./...
go build ./web-agent
go build ./document-agent
```

---

### Day 2 上午：第三层核心内部 Agent

#### A 做

```text
1. context-agent
2. review-agent
3. test-agent
4. security-agent
```

这些都是第一版 LLM + 结构化报告 Artifact，不执行真实工具。

#### B 做

```text
1. diff-agent
2. artifact-agent
3. qa-acceptance-agent
4. version-agent
```

这些也先做结构化 Artifact，不做 Git / DB / 存储写操作。

#### 验收

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub\agents"

go test ./...
go build ./context-agent
go build ./review-agent
go build ./test-agent
go build ./security-agent
go build ./diff-agent
go build ./artifact-agent
go build ./qa-acceptance-agent
go build ./version-agent
```

---

### Day 2 下午：扩展 Agent + 总体验证

#### A 做

```text
1. custom-agent
2. agent-builder-agent
3. deploy-agent
```

#### B 做

```text
1. file-agent
2. web-research-agent
3. ppt-agent
4. release-agent
5. agents/README.md 完整化
```

#### 验收

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub\agents"

go test ./...
go build ./custom-agent
go build ./agent-builder-agent
go build ./deploy-agent
go build ./file-agent
go build ./web-research-agent
go build ./ppt-agent
go build ./release-agent
```

---

## 10. 现实优先级：1 天版与 2 天版

### 10.1 1 天必须完成版

1 天内不要强行做满所有 Agent 的“好用逻辑”，优先做：

```text
1. ADK AgentCard 动态映射。
2. web-agent。
3. document-agent。
4. context-agent。
5. review-agent。
6. test-agent。
7. security-agent。
8. agents/README.md。
```

这 7 个加上已有 code-agent，已经能支撑：

```text
code + web + markdown + context + review + test + security
```

这对 v1.0 Demo 更有价值。

### 10.2 2 天完整版

2 天再补：

```text
diff-agent
artifact-agent
qa-acceptance-agent
version-agent
custom-agent
agent-builder-agent
file-agent
web-research-agent
deploy-agent
ppt-agent
release-agent
```

这些可以先以 skeleton + basic artifact 输出为主。

---

## 11. 验证命令

### 11.1 Go 测试

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub\agents"

go test ./...
```

### 11.2 单个 Agent 构建

```powershell
go build ./code-agent
go build ./web-agent
go build ./document-agent
go build ./context-agent
go build ./review-agent
go build ./test-agent
go build ./security-agent
go build ./diff-agent
go build ./artifact-agent
go build ./qa-acceptance-agent
go build ./version-agent
go build ./custom-agent
go build ./agent-builder-agent
go build ./file-agent
go build ./web-research-agent
go build ./deploy-agent
go build ./ppt-agent
go build ./release-agent
```

### 11.3 清理构建产物

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub"

Remove-Item -Force "agents\code-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\web-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\document-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\context-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\review-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\test-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\security-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\diff-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\artifact-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\qa-acceptance-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\version-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\custom-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\agent-builder-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\file-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\web-research-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\deploy-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\ppt-agent.exe" -ErrorAction SilentlyContinue
Remove-Item -Force "agents\release-agent.exe" -ErrorAction SilentlyContinue
```

### 11.4 安全扫描

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub"

git ls-files .env

git diff | Select-String "OPENAI_API_KEY|ANTHROPIC_API_KEY|AGENTHUB_API_TOKEN|DB_PASSWORD|MYSQL_ROOT_PASSWORD|DATABASE_URL|PRIVATE KEY|BEGIN RSA"

git diff | Select-String "sk-[A-Za-z0-9_-]{20,}"
```

---

## 12. 提交策略

不要一次性 `git add .`。建议分批提交。

### Commit 1：ADK 通用化和模板

```text
feat(adk): 支持通用子 Agent 能力声明
```

暂存范围示例：

```powershell
git add agents/adk/server.go
git add agents/adk/fenced_blocks.go
git add agents/adk/fenced_blocks_test.go
git add agents/README.md
```

### Commit 2：第二层产品 Agent

```text
feat(agents): 添加 web 与 document 子 Agent
```

暂存范围：

```powershell
git add agents/web-agent/main.go
git add agents/web-agent/handler.go
git add agents/web-agent/handler_test.go
git add agents/web-agent/config.yaml
git add agents/web-agent/Dockerfile

git add agents/document-agent/main.go
git add agents/document-agent/handler.go
git add agents/document-agent/handler_test.go
git add agents/document-agent/config.yaml
git add agents/document-agent/Dockerfile
```

### Commit 3：内部核心 Agent

```text
feat(agents): 添加 context review test security 内部 Agent
```

### Commit 4：内部扩展 Agent

```text
feat(agents): 添加 diff artifact qa version 等内部 Agent
```

### Commit 5：产品扩展 Agent

```text
feat(agents): 添加 custom builder file research deploy ppt release Agent
```

每次提交前都要：

```text
1. go test ./...
2. go build 新增 Agent
3. 清理 exe
4. git ls-files .env
5. diff 安全扫描
6. staged 安全扫描
```

---

## 13. Codex 实施 Prompt

### 13.1 ADK 最小通用化

```text
请使用：
$adk-runtime-contract
$a2a-agent-contract
$security-boundary-contract
$testing-review-contract
$code-style-and-conventions

任务：基于当前真实 agents/adk 源码做最小通用化补丁，为新增多个子 Agent 做准备。

允许修改：
- agents/adk/server.go
- agents/adk/types.go
- agents/adk/fenced_blocks.go
- agents/adk/fenced_blocks_test.go
- agents/README.md

要求：
1. AgentCard 必须从 AgentConfig 动态映射 name/description/version/url/skills/inputModes/outputModes/streaming。
2. 不再硬编码 code-agent 能力到所有 Agent。
3. 可选：把 code-agent 的 fenced block parser 抽成 ADK 通用 parser。
4. 不修改 Gateway、Orchestrator、Frontend。
5. 不读取、不创建、不修改 .env。
6. 不写真实 API Key。
7. 不提交代码。

完成后运行：
cd agents
go test ./...
go build ./code-agent
```

### 13.2 实现第二层 Agent

```text
请使用：
$adk-runtime-contract
$a2a-agent-contract
$artifact-contract
$security-boundary-contract
$testing-review-contract
$code-style-and-conventions

任务：基于 agents/code-agent 和当前 ADK，实现第二层产品可见 Agent：

- agents/web-agent
- agents/document-agent

要求：
1. 每个 Agent 都有 main.go、handler.go、handler_test.go、config.yaml、Dockerfile。
2. web-agent 输出 webpage artifact。
3. document-agent 输出 document artifact。
4. 不生成 AG-UI 事件。
5. 不修改 Gateway、Orchestrator、Frontend。
6. 不读取、不创建、不修改 .env。
7. 不写真实密钥。
8. 不提交代码。

完成后运行：
cd agents
go test ./...
go build ./web-agent
go build ./document-agent
```

### 13.3 实现第三层核心 Agent

```text
请使用：
$adk-runtime-contract
$a2a-agent-contract
$artifact-contract
$security-boundary-contract
$testing-review-contract
$observability-debugging-contract
$code-style-and-conventions

任务：实现第三层核心内部 Agent：

- agents/context-agent
- agents/review-agent
- agents/test-agent
- agents/security-agent

要求：
1. 每个 Agent 都是独立 ADK + A2A 服务。
2. 第一版不执行 shell，不读写文件，不访问网络。
3. test-agent 第一版只分析调用方提供的测试日志。
4. security-agent 第一版做规则扫描 + LLM 解释。
5. 输出 context_bundle、review_report、test_report、security_report artifact。
6. 不修改第一层 Orchestrator / Planner / Registry。
7. 不读取、不创建、不修改 .env。
8. 不写真实密钥。
9. 不提交代码。

完成后运行：
cd agents
go test ./...
go build ./context-agent
go build ./review-agent
go build ./test-agent
go build ./security-agent
```

---

## 14. 风险与应对

| 风险 | 说明 | 应对 |
|---|---|---|
| ADK AgentCard 硬编码 | 多 Agent 都暴露成 code-agent 会破坏后续 Registry | 先做 AgentCard 动态映射 |
| 一次性做太多 Agent | 容易变成大量重复但不可维护代码 | 先模板化，再批量复制 |
| 内部 Agent 越权执行工具 | test/deploy/explorer 很容易误执行高危操作 | 第一版全部禁用真实工具，只做文本/JSON 分析 |
| Artifact 类型失控 | 每个 Agent 自定义格式，后续 Converter 难接 | 统一 Type/Title/Content/Metadata |
| 缺少测试 | 解析 LLM 输出容易坏 | 每个 Agent 至少测 parser 和 artifact 生成 |
| 两人冲突修改同一文件 | 同时改 ADK、模板、README 会冲突 | A 改 ADK，B 改测试/README；每半天同步一次 |
| 误提交密钥或构建产物 | 多 Agent 会增加 config 和 Dockerfile | 每次 staged 后扫密钥，不用 git add . |

---

## 15. 最终 Definition of Done

本轮“第二层与第三层子 Agent”完成标准：

```text
必须：
- code-agent 保持可用
- web-agent 可编译、可启动、可输出 webpage artifact
- document-agent 可编译、可启动、可输出 document artifact
- context-agent 可输出 context_bundle
- review-agent 可输出 review_report
- test-agent 可输出 test_report
- security-agent 可输出 security_report
- 所有新增 Agent 都有 config.yaml
- 所有新增 Agent 都有 handler_test.go
- agents go test ./... 通过
- 新增 Agent go build 通过
- 不提交 .env
- 不提交真实密钥
- 不提交 exe 构建产物

可选：
- diff/artifact/qa/version/custom/builder/file/research/deploy/ppt/release skeleton 全部补齐
- Dockerfile 全部补齐
- agents/README.md 完整列出 Agent 状态
```

---

## 16. 最终建议

如果只给 1 天，建议做：

```text
ADK 动态 AgentCard
web-agent
document-agent
context-agent
review-agent
test-agent
security-agent
agents/README.md
```

如果有 2 天，再补：

```text
diff-agent
artifact-agent
qa-acceptance-agent
version-agent
custom-agent
agent-builder-agent
file-agent
web-research-agent
deploy-agent
ppt-agent
release-agent
```

最重要的是不要在这一步强行做 Orchestrator。先把 Agent 变成统一、可运行、可测试的 A2A 服务。等第二层和第三层 Agent 都能稳定输出 Artifact 后，再做第一层统一 Agent / 意图编排 / Registry，会更顺，也更容易在答辩时解释：

> AgentHub 的核心不是把所有逻辑写进一个巨型 Orchestrator，而是先用 ADK 把每个子 Agent 标准化，再由统一 Agent 根据意图编排这些可发现、可调用、可观测的专业 Agent。
