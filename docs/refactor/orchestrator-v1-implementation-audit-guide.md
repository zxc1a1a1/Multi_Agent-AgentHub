# AgentHub v1.0 Orchestrator 落地与审计指导

> 建议放置路径：`docs/refactor/orchestrator-v1-implementation-audit-guide.md`  
> 适用对象：项目负责人、后端开发者、前端集成开发者、使用 Claude Code / Codex 的开发者、审计者  
> 当前目标：先把 `services/` 新架构中的独立 Orchestrator 最小闭环落地，再谈 LLM Planner、动态 Agent 发现和更多 Agent 服务化。  
> 审计原则：每个 Phase 完成后必须审计；审计未通过，不进入下一 Phase。

---

## 0. 本文目的

本文不是新的宏观架构设计，也不是继续扩写 v1.0 愿景，而是一份**可执行的开发与审计指导**。

当前 AgentHub 仓库已经有比较完整的文档规划：Gateway 只做入口，Orchestrator 独立负责 LLM 编排、多 Agent 调度、计划校验和结果聚合。但当前真实代码中，新架构服务化主线还没有独立 Orchestrator 服务，Gateway 仍然直接持有 code-agent / web-agent 的 URL，并通过 `runservice` 做静态单 Agent 路由。

因此，本轮工作只解决一个核心问题：

```text
把文档里的 Orchestrator 从“规划”变成 services/ 下真实可运行的独立服务。
```

本轮完成后，系统目标拓扑应从：

```text
Frontend
  ↓
Gateway
  ↓
code-agent / web-agent
```

变成：

```text
Frontend
  ↓
Gateway
  ↓
Orchestrator
  ↓
code-agent / web-agent
```

并且 Orchestrator 内部至少具备：

```text
RulePlanner → OrchestrationPlan → PlanValidator → Executor → A2A Dispatcher → Result Aggregation
```

注意：本轮不追求一次性实现完整智能编排，而是先跑通可审计、可测试、可演示的 vertical slice。

---

## 1. 当前仓库事实基线

这一节是后续开发和审计的事实基础。任何开发指导都必须先服从当前仓库真实目录，而不是只服从理想文档。

### 1.1 仓库存在三层代码/文档体系

当前仓库不是单一新架构仓库，而是以下内容并存：

```text
legacy / 原 MVP 层：
- server/
- agents/

new-arch 服务化层：
- services/gateway/
- services/agents/code-agent/
- services/agents/web-agent/

契约与 AI 开发约束层：
- docs/contracts/
- .claude/skills/
- .agents/skills/
- pkg/adk/
- pkg/runtime/
```

开发时必须明确：本轮主战场是 `services/` 新架构，不是旧的 `server/` 主线。

### 1.2 当前 services/ 下缺少 Orchestrator

当前 `services/` 目录下只有：

```text
services/
  agents/
  gateway/
```

还没有：

```text
services/orchestrator/
```

所以本轮不是“修一下已有 Orchestrator”，而是要新增一个真正的新架构 Orchestrator 服务。

### 1.3 当前 docker-compose.new-arch.yml 仍是 Gateway 直连 Agent

当前新架构 compose 里有四个服务：

```text
code-agent-new
web-agent-new
gateway-new
frontend-new
```

其中 `gateway-new` 仍直接配置：

```yaml
AGENT_CODE_URL: "http://code-agent-new:8080"
AGENT_WEB_URL: "http://web-agent-new:8080"
CODE_AGENT_URL: "http://code-agent-new:8080"
WEB_AGENT_URL: "http://web-agent-new:8080"
```

这说明当前新架构链路仍是：

```text
Gateway → code-agent / web-agent
```

而不是：

```text
Gateway → Orchestrator → code-agent / web-agent
```

### 1.4 当前 go.work 尚未纳入 services/orchestrator

当前 workspace 包含：

```text
./agents
./server
./pkg/adk
./pkg/runtime
./services/gateway
./services/agents/code-agent
./services/agents/web-agent
```

还没有：

```text
./services/orchestrator
```

所以新增 Orchestrator 后，必须同步修改 `go.work`。

### 1.5 当前 Gateway runservice 是静态单 Agent 路由

当前 `services/gateway/runservice` 包含：

```text
registry.go
registry_test.go
remote_agent.go
remote_agent_test.go
routing.go
routing_test.go
```

其中 `RoutingRunService` 的职责是：

```text
routes one run to one static remote agent
```

当前实现一次只选择一个 `targetName`，再调用一个 `RemoteAgentRunService`。这不是 multi-agent orchestration，也不是 Planner 驱动的执行。

`RemoteAgentRunService` 的注释也明确说明它只是 Gateway 的单远程 Agent 适配器，不是 Orchestrator，也不实现 planner / executor 行为。

### 1.6 当前可服务化调用的 Agent 是两个

本轮可直接纳入新 Orchestrator 调用链路的 Agent 是：

```text
services/agents/code-agent/
services/agents/web-agent/
```

顶层 `agents/` 中存在更多 Agent 能力目录，但它们不是本轮要全部服务化运行的对象。不要把“19 个 Agent 能力目录”误解为“当前都已经是新架构可运行服务”。

---

## 2. 本轮目标与非目标

### 2.1 本轮目标

本轮目标是完成一个可运行、可审计、可测试的 Orchestrator vertical slice。

必须完成：

```text
1. 新增 services/orchestrator 独立服务。
2. 新增 orchestrator-new compose 服务。
3. Gateway 不再直接持有 code-agent / web-agent 调用职责。
4. Gateway 改为调用 Orchestrator 的内部 stream endpoint。
5. Orchestrator 接管 code-agent / web-agent 的 Registry 与 A2A Dispatcher。
6. Orchestrator 生成 OrchestrationPlan。
7. Orchestrator 执行 PlanValidator。
8. Orchestrator 支持 single executor。
9. Orchestrator 支持 ordered_parallel 的受控串行展示版。
10. smoke test 覆盖 single code、single web、mixed ordered_parallel。
```

### 2.2 本轮非目标

本轮不要做这些事：

```text
1. 不容器化 19 个 Agent。
2. 不把顶层 agents/ 全部迁到 services/agents/。
3. 不把旧 server/internal/orchestrator 当成本轮主路径。
4. 不马上实现完整 LLM Planner。
5. 不马上实现 AgentCard 动态发现。
6. 不马上实现真实并发 fan-out。
7. 不马上实现 review_loop。
8. 不马上实现 deploy-agent / custom-agent-builder。
9. 不引入复杂权限系统。
10. 不把 Gateway 继续扩成“内嵌 Orchestrator”。
```

### 2.3 最小完成定义

本轮完成的最小定义是：

```text
用户请求：帮我做一个登录页面和 Go 登录接口

Frontend → Gateway → Orchestrator

Orchestrator 生成 plan：
- task_web: web-agent
- task_code: code-agent
- strategy: ordered_parallel

PlanValidator 通过后：
- 先调用 web-agent
- 再调用 code-agent
- 最后输出 Orchestrator 汇总

前端能看到两个 Agent 的独立输出或至少能从事件中区分两个 Agent。
```

---

## 3. 审计机制总则

本项目当前最大风险不是“不知道怎么设计”，而是“文档说得很完整，代码没有真的落地”。因此本轮必须引入审计闸门。

### 3.1 审计者职责

审计者由 ChatGPT 承担。每个 Phase 完成后，开发者需要把以下材料发给审计者：

```text
1. 本 Phase 的 git diff 或 PR 链接。
2. 相关文件树，例如 tree services -L 3。
3. 关键文件内容或链接。
4. 测试命令和输出。
5. docker compose 配置或运行结果。
6. 若已经推到 GitHub，提供分支链接。
```

审计者只做判断，不自动修复。审计输出必须明确：

```text
审计结论：通过 / 不通过
是否允许进入下一 Phase：是 / 否
P0 阻塞问题：...
P1 必修问题：...
P2 建议问题：...
证据：文件路径 / 代码片段 / 命令输出
```

### 3.2 审计不可和修复混在一起

禁止使用这种提示：

```text
帮我审计并顺手修掉问题。
```

必须拆成两步：

```text
第一步：只审计，不修改文件。
第二步：根据审计报告再开启修复任务。
```

这样做是为了防止 AI 工具一边审计一边改动，导致问题被掩盖。

### 3.3 审计分三层

#### 第一层：事实审计

开发前先审计仓库当前状态是否与指导文档一致。

#### 第二层：阶段审计

每个 Phase 完成后审计改动是否满足该 Phase 的边界和验收标准。

#### 第三层：最终架构审计

所有 Phase 完成后审计整体架构是否真的从 Gateway 直连 Agent 变成 Gateway → Orchestrator → Agent。

---

## 4. Skill 调用总规则

项目中同时存在 Claude Code Skill 和 Codex Skill：

```text
.claude/skills/
.agents/skills/
```

两边当前 Skill 名称基本对应，但调用方式不同。

### 4.1 Claude Code 调用方式

Claude Code 项目 Skill 位于：

```text
.claude/skills/<skill-name>/SKILL.md
```

Claude Code 中可以通过斜杠命令直接调用：

```text
/<skill-name> 任务描述
```

示例：

```text
/project-architecture
基于当前仓库 services/、server/、agents/、docs/contracts/ 的实际目录，分析新增 services/orchestrator 的边界。
不要修改文件，只输出允许修改范围、禁止修改范围和风险点。
```

### 4.2 Codex 调用方式

Codex 项目 Skill 位于：

```text
.agents/skills/<skill-name>/SKILL.md
```

Codex 支持两类触发方式：

```text
1. 显式调用：在 CLI / IDE 中运行 /skills 或输入 $ 来 mention 某个 skill。
2. 隐式调用：任务描述匹配 skill description 时由 Codex 自动选择。
```

本项目建议优先使用显式调用，避免错用 Skill。

Codex 示例：

```text
使用 $project-architecture、$gateway-orchestrator-contract、$testing-review-contract。
先审计当前 services/gateway/runservice 与 docs/contracts/gateway-orchestrator.md 的差异。
不要修改文件。
输出：事实、差异、风险、允许修改文件、禁止修改文件。
```

### 4.3 本轮高频 Skill

本轮最常用 Skill：

```text
/project-architecture
/gateway-orchestrator-contract
/intent-orchestration-contract
/a2a-agent-contract
/agui-event-contract
/artifact-contract
/docker-compose-delivery
/testing-review-contract
/observability-debugging-contract
/security-boundary-contract
/code-style-and-conventions
```

Codex 中对应写作：

```text
$project-architecture
$gateway-orchestrator-contract
$intent-orchestration-contract
$a2a-agent-contract
$agui-event-contract
$artifact-contract
$docker-compose-delivery
$testing-review-contract
$observability-debugging-contract
$security-boundary-contract
$code-style-and-conventions
```

### 4.4 Skill 使用纪律

每次调用 Skill 前，都要写清楚四件事：

```text
1. 目标：这次要完成什么。
2. 范围：允许修改哪些路径。
3. 禁止：禁止修改哪些路径或做哪些扩展。
4. 验收：做完后必须通过什么测试或审计。
```

禁止这种模糊提示：

```text
/gateway-orchestrator-contract 帮我完成 Orchestrator。
```

推荐这种提示：

```text
/gateway-orchestrator-contract
目标：新增 services/orchestrator 的 /health 与 /v1/runs/stream mock endpoint。
范围：只允许修改 services/orchestrator、go.work、docker-compose.new-arch.yml。
禁止：不要修改 legacy server/；不要实现 LLM Planner；不要调用 code-agent/web-agent。
验收：go test ./services/orchestrator/... 通过；curl /health 返回 200。
```

---

## 5. Phase 0：事实审计

### 5.1 目标

开发前先确认仓库事实，避免基于错误目录写开发计划。

### 5.2 开发动作

无开发动作，只审计。

### 5.3 推荐 Claude Code Prompt

```text
/project-architecture
请审计当前仓库目录结构，不要修改文件。
重点检查：
1. 根目录是否同时存在 server/、agents/、services/、docs/contracts/、.claude/skills/、.agents/skills/。
2. services/ 下是否只有 agents 和 gateway。
3. services/agents 下是否只有 code-agent 和 web-agent。
4. go.work 是否尚未包含 ./services/orchestrator。
5. docker-compose.new-arch.yml 是否仍然只有 code-agent-new、web-agent-new、gateway-new、frontend-new。
6. services/gateway/runservice 是否仍然是静态单 Agent 路由。
输出事实审计报告，不要修改文件。
```

### 5.4 推荐 Codex Prompt

```text
使用 $project-architecture、$testing-review-contract。
只审计当前仓库事实，不修改文件。
检查：services/、server/、agents/、docs/contracts/、.claude/skills/、.agents/skills、go.work、docker-compose.new-arch.yml、services/gateway/runservice。
输出：事实、证据文件、对后续开发计划的影响。
```

### 5.5 审计清单

```text
[ ] 根目录确实同时存在 legacy、新架构、契约、Skill 体系。
[ ] services/ 下没有 orchestrator。
[ ] go.work 没有 ./services/orchestrator。
[ ] docker-compose.new-arch.yml 没有 orchestrator-new。
[ ] gateway-new 仍直接配置 code/web agent URL。
[ ] services/gateway/runservice 是静态单 Agent 路由。
```

### 5.6 通过标准

审计结果必须明确：

```text
本轮应在 services/orchestrator 新增独立服务。
本轮不应把旧 server/internal/orchestrator 作为主路径。
本轮不应服务化顶层 agents/ 的全部 Agent。
```

---

## 6. Phase 1：新增空 Orchestrator 服务

### 6.1 目标

新增一个最小可运行的 `services/orchestrator` 服务，只实现基础 HTTP 服务和健康检查。

### 6.2 允许修改路径

```text
services/orchestrator/**
go.work
docker-compose.new-arch.yml
```

### 6.3 禁止修改路径

```text
server/**
agents/**
services/agents/**
frontend/**
```

除非是修正编译路径，不要改 `services/gateway/**`。

### 6.4 预期目录

```text
services/orchestrator/
  cmd/orchestrator/main.go
  config/
  httpapi/
  go.mod
  Dockerfile
```

### 6.5 必须实现

```text
GET /health → 200 OK
```

### 6.6 推荐 Claude Code Prompt

```text
/gateway-orchestrator-contract
目标：新增 services/orchestrator 空服务，只实现 /health。
范围：只允许新增 services/orchestrator，修改 go.work，必要时修改 docker-compose.new-arch.yml。
禁止：不要修改 server/；不要修改 agents/；不要实现 Planner；不要调用 code-agent/web-agent；不要改前端。
验收：go test ./services/orchestrator/... 通过；服务启动后 GET /health 返回 200。
```

### 6.7 推荐 Codex Prompt

```text
使用 $gateway-orchestrator-contract、$code-style-and-conventions、$testing-review-contract。
实现 Phase 1：新增 services/orchestrator 空服务。
只允许：services/orchestrator/**、go.work、docker-compose.new-arch.yml。
禁止：server/**、agents/**、services/agents/**、frontend/**。
先给计划，再修改。完成后输出变更文件和测试命令。
```

### 6.8 Phase 1 审计清单

```text
[ ] services/orchestrator/ 存在。
[ ] services/orchestrator/cmd/orchestrator/main.go 存在。
[ ] services/orchestrator/go.mod 存在。
[ ] services/orchestrator/Dockerfile 存在。
[ ] go.work 已加入 ./services/orchestrator。
[ ] /health 能返回 200。
[ ] 没有修改 legacy server/ 作为主路径。
[ ] 没有在 Phase 1 提前实现 Planner / Agent 调用。
```

### 6.9 Phase 1 审计提交给 ChatGPT 的材料

```text
1. git diff。
2. tree services/orchestrator -L 3。
3. go.work 内容。
4. go test ./services/orchestrator/... 输出。
5. curl http://localhost:<orchestrator-port>/health 输出。
```

---

## 7. Phase 2：Gateway 改为调用 Orchestrator

### 7.1 目标

让 Gateway 只调用 Orchestrator，不再在新链路中直接 dispatch 到 code-agent / web-agent。

第一版 Orchestrator 可以返回 mock stream，不需要真的调用 Agent。

### 7.2 允许修改路径

```text
services/gateway/**
services/orchestrator/**
docs/contracts/gateway-orchestrator.md（如需补充实现说明）
docker-compose.new-arch.yml
```

### 7.3 禁止修改路径

```text
server/**
agents/**
services/agents/**
frontend/**
```

### 7.4 必须实现

Gateway 新增内部客户端：

```text
OrchestratorClient
```

Gateway 的运行链路变为：

```text
HTTP/SSE request
  ↓
Gateway handler
  ↓
OrchestratorClient
  ↓
Orchestrator /v1/runs/stream
  ↓
Gateway SSE flush
```

### 7.5 环境变量调整方向

Gateway 侧新增：

```text
ORCHESTRATOR_URL=http://orchestrator-new:8080
```

Gateway 新链路不应继续依赖：

```text
AGENT_CODE_URL
AGENT_WEB_URL
CODE_AGENT_URL
WEB_AGENT_URL
```

这些变量后续应移动到 Orchestrator 服务侧。

### 7.6 推荐 Claude Code Prompt

```text
/gateway-orchestrator-contract
目标：Phase 2，把 Gateway 新链路改为调用 Orchestrator 的 /v1/runs/stream。
范围：services/gateway、services/orchestrator、docker-compose.new-arch.yml。
禁止：不要调用 code-agent/web-agent；不要实现 Planner；不要修改 server/ legacy；不要改 services/agents。
要求：Orchestrator 可以先返回 mock stream；Gateway 负责 SSE 转发和错误脱敏。
验收：Gateway 的新路径只依赖 ORCHESTRATOR_URL；mock run 能从前端/Gateway 收到流式响应。
```

### 7.7 推荐 Codex Prompt

```text
使用 $gateway-orchestrator-contract、$agui-event-contract、$testing-review-contract。
实现 Phase 2：Gateway 调用 Orchestrator mock stream。
不要调用 code-agent/web-agent。
不要修改 legacy server/。
完成后输出：修改文件、关键接口、测试命令、未完成项。
```

### 7.8 Phase 2 审计清单

```text
[ ] Gateway 存在 OrchestratorClient 或等价内部客户端。
[ ] Gateway 新链路通过 ORCHESTRATOR_URL 调用 Orchestrator。
[ ] Orchestrator 提供 /v1/runs/stream mock endpoint。
[ ] Gateway 仍负责 SSE 连接和 flush。
[ ] Phase 2 没有调用 code-agent/web-agent。
[ ] Phase 2 没有把 Planner 塞回 Gateway。
[ ] docker-compose.new-arch.yml 初步包含 orchestrator-new 或为 Phase 8 预留明确 TODO。
```

### 7.9 Phase 2 审计提交材料

```text
1. git diff。
2. services/gateway 中 OrchestratorClient 相关代码。
3. services/orchestrator 中 /v1/runs/stream mock 代码。
4. Gateway 环境变量配置。
5. smoke 或 curl 输出。
```

---

## 8. Phase 3：Orchestrator 接管 Registry 与 A2A Dispatcher

### 8.1 目标

把“知道有哪些 Agent、如何调用 Agent”的职责从 Gateway 迁到 Orchestrator。

### 8.2 允许修改路径

```text
services/orchestrator/**
services/gateway/**（只删除或绕开旧直连职责）
docker-compose.new-arch.yml
```

### 8.3 必须实现

Orchestrator 内部新增：

```text
registry/
  static_registry.go
  static_registry_test.go

dispatcher/
  a2a_dispatcher.go
  a2a_dispatcher_test.go
```

第一版 Static Registry 可直接读取环境变量：

```text
CODE_AGENT_URL=http://code-agent-new:8080
WEB_AGENT_URL=http://web-agent-new:8080
```

这是允许的，因为本轮重点是服务边界先落地，不是马上实现动态 AgentCard 发现。

### 8.4 Gateway 侧要求

Gateway 不应再作为新链路的 Agent Registry 所有者。

旧 `services/gateway/runservice` 可以暂时保留，避免大范围破坏测试，但新链路不应继续依赖它做编排。

### 8.5 推荐 Claude Code Prompt

```text
/a2a-agent-contract
/gateway-orchestrator-contract
目标：Phase 3，让 Orchestrator 接管 StaticAgentRegistry 与 A2A Dispatcher。
范围：services/orchestrator 为主，services/gateway 只做去直连或兼容保留。
禁止：不要实现 LLM Planner；不要服务化顶层 agents/；不要改旧 server/。
要求：Orchestrator 从 CODE_AGENT_URL / WEB_AGENT_URL 读取两个 Agent 地址，并能封装调用入口。
验收：单元测试覆盖 registry missing URL、unknown agent、dispatcher bad request。
```

### 8.6 Phase 3 审计清单

```text
[ ] Orchestrator 内部有 Registry。
[ ] Orchestrator 内部有 Dispatcher。
[ ] CODE_AGENT_URL / WEB_AGENT_URL 已归属 Orchestrator。
[ ] Gateway 新链路不再读取 code/web agent URL。
[ ] 仍只接入 services/agents/code-agent 和 services/agents/web-agent。
[ ] 没有引入 19 个 Agent 服务化。
```

---

## 9. Phase 4：RulePlanner 与 OrchestrationPlan

### 9.1 目标

先实现规则 Planner，不接 LLM Planner。

目标是让所有请求都先转换成标准 OrchestrationPlan，再执行。

### 9.2 推荐目录

```text
services/orchestrator/
  planner/
    rule_planner.go
    rule_planner_test.go
  plan/
    types.go
    plan_id.go
```

### 9.3 RulePlanner 初始规则

```text
显式 @code-agent → single code-agent
显式 @web-agent → single web-agent
包含 “页面 / UI / HTML / React / 登录页 / 前端” → web-agent
包含 “Go / API / 后端 / 接口 / server” → code-agent
同时包含前端类词和后端类词 → ordered_parallel: web-agent + code-agent
无法判断 → single code-agent
```

### 9.4 OrchestrationPlan 最小字段

第一版至少包含：

```json
{
  "planId": "plan_xxx",
  "runId": "run_xxx",
  "conversationId": "conv_xxx",
  "planningMode": "rule",
  "strategy": "single | ordered_parallel",
  "intentSummary": "...",
  "tasks": [
    {
      "id": "task_web",
      "agentName": "web-agent",
      "capabilityIds": ["web_generation"],
      "taskContent": "...",
      "expectedOutputs": ["webpage"],
      "dependsOn": [],
      "priority": 1,
      "timeoutMs": 120000,
      "riskLevel": "low"
    }
  ],
  "aggregation": {
    "required": true,
    "mode": "summary"
  },
  "fallback": {
    "enabled": true,
    "reason": "rule_default"
  },
  "validation": {
    "validated": false
  }
}
```

注意：`validation.validated` 只能由本地 Validator 设置，Planner 不得直接宣称计划已通过。

### 9.5 推荐 Claude Code Prompt

```text
/intent-orchestration-contract
目标：Phase 4，实现 RulePlanner 与 OrchestrationPlan 类型。
范围：services/orchestrator/planner、services/orchestrator/plan。
禁止：不要接 LLM；不要执行 Agent；不要改 Gateway；不要修改 server/。
要求：所有 Planner 输出必须是 OrchestrationPlan；validation.validated 初始必须是 false。
验收：单元测试覆盖 @code-agent、@web-agent、前端任务、后端任务、前后端混合任务、unknown fallback。
```

### 9.6 Phase 4 审计清单

```text
[ ] 有 RulePlanner。
[ ] 有 OrchestrationPlan 类型。
[ ] mixed task 能生成 ordered_parallel plan。
[ ] tasks 至少包含 agentName、capabilityIds、taskContent、expectedOutputs、dependsOn、priority、timeoutMs、riskLevel。
[ ] Planner 不直接执行 Agent。
[ ] Planner 不设置 validation.validated=true。
```

---

## 10. Phase 5：PlanValidator

### 10.1 目标

所有计划必须先校验再执行。Validator 是防止幻觉 Agent、非法 capability 和越权执行的第一道闸门。

### 10.2 推荐目录

```text
services/orchestrator/
  validator/
    validator.go
    validator_test.go
```

### 10.3 Validator 最小规则

```text
1. planId 必填。
2. runId 必填。
3. conversationId 必填。
4. strategy 只能是 single / ordered_parallel。
5. tasks 非空。
6. v1.0 tasks 最多 3 个。
7. agentName 必须存在于 Registry。
8. capabilityIds 必须属于目标 Agent。
9. expectedOutputs 必须被目标 Agent 支持。
10. dependsOn 只能引用已有 task。
11. timeoutMs 必须在 5s 到 180s 之间。
12. riskLevel 必须是 low / medium / high。
13. high risk 默认不得自动执行。
```

### 10.4 推荐 Claude Code Prompt

```text
/intent-orchestration-contract
/security-boundary-contract
目标：Phase 5，实现 PlanValidator。
范围：services/orchestrator/validator、services/orchestrator/plan、必要的 registry 接口。
禁止：不要接 LLM；不要执行 Agent；不要改前端；不要修改 legacy server/。
要求：Validator 必须基于 Registry 校验 agentName / capabilityIds / expectedOutputs。
验收：单元测试覆盖 unknown agent、unknown capability、empty tasks、invalid strategy、invalid dependsOn、timeout 越界、合法 plan。
```

### 10.5 Phase 5 审计清单

```text
[ ] Executor 之前必过 Validator。
[ ] Validator 依赖 Registry，而不是硬编码字符串散落在代码里。
[ ] 非法 agentName 被拒绝。
[ ] 非法 capability 被拒绝。
[ ] 非法 strategy 被拒绝。
[ ] validation.validated 只由 Validator 设置。
[ ] 测试覆盖失败路径。
```

---

## 11. Phase 6：single executor

### 11.1 目标

跑通单 Agent 任务：

```text
用户 → Gateway → Orchestrator → Plan(single) → Validator → code-agent 或 web-agent
```

### 11.2 推荐目录

```text
services/orchestrator/
  executor/
    executor.go
    single_executor.go
    single_executor_test.go
```

### 11.3 必须支持场景

```text
@code-agent 帮我写一个 Go HTTP server
→ single code-agent

@web-agent 帮我写一个登录页面
→ single web-agent

帮我写一个 Go API
→ single code-agent

帮我写一个登录页
→ single web-agent
```

### 11.4 推荐 Claude Code Prompt

```text
/a2a-agent-contract
/intent-orchestration-contract
/agui-event-contract
目标：Phase 6，实现 single executor。
范围：services/orchestrator/executor、dispatcher、httpapi。
禁止：不要实现 ordered_parallel；不要接 LLM；不要改旧 server/；不要服务化更多 Agent。
要求：执行前必须验证 plan.validation.validated=true；执行结果要保留 agentName / taskId / runId。
验收：single code 和 single web 的集成测试或 smoke 测试通过。
```

### 11.5 Phase 6 审计清单

```text
[ ] single executor 只执行一个 task。
[ ] 执行前检查 validation.validated=true。
[ ] code-agent 可被 Orchestrator 调用。
[ ] web-agent 可被 Orchestrator 调用。
[ ] Gateway 不直接调用 code-agent/web-agent。
[ ] 返回事件中能识别 agentName。
[ ] 错误信息脱敏。
```

---

## 12. Phase 7：ordered_parallel executor（受控串行展示版）

### 12.1 目标

实现多 Agent 协作的最小演示能力。

注意：第一版 `ordered_parallel` 不要求真实 goroutine 并发。建议采用：

```text
逻辑上是多 Agent 计划；执行上受控串行；UI 上按 Agent 分段展示。
```

这样可以降低 SSE token 交错、前端消息归属混乱和持久化复杂度。

### 12.2 执行顺序

```text
STATE_UPDATE planning
STATE_UPDATE plan_validated
TEXT_MESSAGE_START sender=web-agent taskId=task_web
web-agent output
TEXT_MESSAGE_END sender=web-agent taskId=task_web
TEXT_MESSAGE_START sender=code-agent taskId=task_code
code-agent output
TEXT_MESSAGE_END sender=code-agent taskId=task_code
TEXT_MESSAGE_START sender=orchestrator
summary
TEXT_MESSAGE_END sender=orchestrator
```

### 12.3 必须支持场景

```text
帮我做一个登录页面和 Go 登录接口
→ plan.strategy = ordered_parallel
→ web-agent 生成页面
→ code-agent 生成接口
→ Orchestrator 汇总
```

### 12.4 推荐 Claude Code Prompt

```text
/intent-orchestration-contract
/agui-event-contract
/artifact-contract
目标：Phase 7，实现 ordered_parallel 的受控串行展示版。
范围：services/orchestrator/executor、eventstream、adapter。
禁止：不要做真实并发 fan-out；不要接 review_loop；不要改旧 server/；不要服务化更多 Agent。
要求：每个 task 输出必须保留 agentName、taskId、runId；最后输出 Orchestrator summary。
验收：mixed task 能依次调用 web-agent 与 code-agent。
```

### 12.5 Phase 7 审计清单

```text
[ ] mixed task 能生成两个 task。
[ ] ordered_parallel 至少调用 web-agent 与 code-agent。
[ ] 两个 Agent 的输出不会混成同一个 sender。
[ ] taskId / agentName / runId 能被追踪。
[ ] Orchestrator 生成最终 summary。
[ ] 没有引入真实并发导致流式事件交错。
```

---

## 13. Phase 8：Docker Compose 与 smoke test

### 13.1 目标

让新架构以五服务形态跑起来：

```text
frontend-new
gateway-new
orchestrator-new
code-agent-new
web-agent-new
```

### 13.2 docker-compose.new-arch.yml 目标方向

新增：

```yaml
orchestrator-new:
  build:
    context: .
    dockerfile: services/orchestrator/Dockerfile
  environment:
    ORCHESTRATOR_ADDR: ":8080"
    CODE_AGENT_URL: "http://code-agent-new:8080"
    WEB_AGENT_URL: "http://web-agent-new:8080"
  depends_on:
    code-agent-new:
      condition: service_healthy
    web-agent-new:
      condition: service_healthy
  ports:
    - "8090:8080"
```

调整 Gateway：

```yaml
gateway-new:
  environment:
    GATEWAY_ADDR: ":8080"
    ORCHESTRATOR_URL: "http://orchestrator-new:8080"
    GATEWAY_ENABLE_AUTH: "false"
    GATEWAY_ALLOWED_ORIGINS: "http://localhost:3000,http://127.0.0.1:3000"
  depends_on:
    orchestrator-new:
      condition: service_healthy
```

Gateway 新链路不应继续依赖：

```text
AGENT_CODE_URL
AGENT_WEB_URL
CODE_AGENT_URL
WEB_AGENT_URL
```

### 13.3 smoke test 至少覆盖

```text
1. /health: code-agent-new healthy
2. /health: web-agent-new healthy
3. /health: orchestrator-new healthy
4. /health: gateway-new healthy
5. single code: 用户请求 Go API，经 Orchestrator 调 code-agent
6. single web: 用户请求登录页，经 Orchestrator 调 web-agent
7. mixed task: 用户请求登录页 + Go 接口，经 Orchestrator ordered_parallel 调两个 Agent
```

### 13.4 推荐 Claude Code Prompt

```text
/docker-compose-delivery
/testing-review-contract
目标：Phase 8，更新 docker-compose.new-arch.yml 和 smoke test，让 frontend/gateway/orchestrator/code-agent/web-agent 五服务闭环可运行。
范围：docker-compose.new-arch.yml、smoke 脚本、必要的 README 说明。
禁止：不要改业务逻辑；不要服务化更多 Agent；不要修改 legacy docker-compose.yml 作为主路径。
要求：gateway-new 只依赖 ORCHESTRATOR_URL；orchestrator-new 持有 CODE_AGENT_URL / WEB_AGENT_URL。
验收：docker compose -f docker-compose.new-arch.yml up --build 后 smoke test 通过。
```

### 13.5 Phase 8 审计清单

```text
[ ] compose 中有 orchestrator-new。
[ ] gateway-new depends_on orchestrator-new。
[ ] orchestrator-new depends_on code-agent-new 与 web-agent-new。
[ ] gateway-new 不再配置 code/web agent URL。
[ ] orchestrator-new 配置 code/web agent URL。
[ ] smoke test 覆盖 single code。
[ ] smoke test 覆盖 single web。
[ ] smoke test 覆盖 mixed ordered_parallel。
```

---

## 14. 最终架构审计

所有 Phase 完成后，必须做一次最终架构审计。

### 14.1 最终审计输入

开发者需要提供：

```text
1. PR 链接或完整 git diff。
2. tree services -L 3。
3. go.work。
4. docker-compose.new-arch.yml。
5. services/gateway 关键入口代码。
6. services/orchestrator 关键目录和接口。
7. smoke test 脚本。
8. smoke test 输出。
```

### 14.2 最终审计硬指标

```text
[ ] services/orchestrator 是独立 Go module 或 workspace 成员。
[ ] go.work 包含 ./services/orchestrator。
[ ] docker-compose.new-arch.yml 有 orchestrator-new。
[ ] Gateway 新链路只调用 Orchestrator。
[ ] Gateway 新链路不直接调用 code-agent / web-agent。
[ ] Orchestrator 持有 Agent Registry。
[ ] Orchestrator 生成 OrchestrationPlan。
[ ] Orchestrator 执行 PlanValidator。
[ ] Orchestrator 支持 single。
[ ] Orchestrator 支持 ordered_parallel 受控串行展示版。
[ ] Orchestrator 调用 services/agents/code-agent。
[ ] Orchestrator 调用 services/agents/web-agent。
[ ] smoke test 证明 mixed task 经过 Orchestrator 调两个 Agent。
[ ] 没有把旧 server/ 当作新架构主路径。
[ ] 没有把 19 个 Agent 一次性服务化。
```

### 14.3 最终审计报告模板

```text
# AgentHub Orchestrator Vertical Slice 最终审计报告

审计结论：通过 / 不通过
是否允许进入 LLM Planner 阶段：是 / 否

## 1. 证据
- services/orchestrator：...
- go.work：...
- docker-compose.new-arch.yml：...
- Gateway 调用链路：...
- Orchestrator plan / validation / execution：...
- smoke test：...

## 2. P0 阻塞问题
- ...

## 3. P1 必修问题
- ...

## 4. P2 建议问题
- ...

## 5. 结论
- 当前是否已经从 Gateway 直连 Agent 切换为 Gateway → Orchestrator → Agent。
- 当前是否具备进入 LLM Planner / AgentCard 动态发现 / 真并发 fan-out 的基础。
```

---

## 15. 开发者每次提交给审计者的格式

每个 Phase 完成后，建议使用以下格式发给审计者：

```text
请审计 Phase X，不要修改文件。

目标：...

我改了这些文件：
- ...

关键 diff：
```diff
...
```

文件树：
```text
...
```

测试命令：
```bash
...
```

测试输出：
```text
...
```

我认为满足的验收项：
- [ ] ...

请输出：
1. 审计结论：通过 / 不通过
2. P0 阻塞问题
3. P1 必修问题
4. P2 建议问题
5. 是否允许进入下一 Phase
```

如果已经推到 GitHub，可以提供分支链接，审计者会基于远程文件重新核对。

---

## 16. 后续开发建议

以下内容必须在最终架构审计通过之后再做。

### 16.1 LLM Planner

只有 RulePlanner + PlanValidator + Executor 稳定后，才接 LLM Planner。

推荐接口：

```go
type Planner interface {
    Plan(ctx context.Context, input PlannerInput) (*OrchestrationPlan, error)
}
```

实现：

```text
RulePlanner
LLMPlanner
FallbackPlanner
```

运行策略：

```text
显式 @Agent → RulePlanner
简单任务 → RulePlanner
复杂任务 → LLMPlanner
LLM 超时 / JSON 非法 / Validation 失败 → RulePlanner fallback
```

LLM 只能输出 plan，不能直接调用 Agent。

### 16.2 AgentCard 动态发现

第二阶段再把 Static Registry 升级为：

```text
AgentCard fetch
health check
capability index
outputModes index
artifactTypes index
```

但第一版不要为了动态发现阻塞 Orchestrator 服务拆分。

### 16.3 真并发 fan-out

`ordered_parallel` 稳定后，再做真实并发。

引入真实并发前必须先解决：

```text
1. 多 Agent 流式事件交错。
2. messageId / taskId / agentName 归属。
3. 前端展示顺序。
4. 错误隔离。
5. 部分失败聚合。
```

### 16.4 Run / RunStep / Artifact 持久化

在编排主链路稳定后，再将：

```text
run
run_step
plan
artifact
trace
```

持久化到数据库。

不要在 Orchestrator 服务拆分第一阶段引入复杂数据迁移。

### 16.5 Review / Test Agent

先不要服务化所有 Agent。建议下一批只考虑：

```text
review-agent
test-agent
```

用于形成：

```text
code → review → test → fix
```

但这属于 Orchestrator vertical slice 之后的质量闭环。

### 16.6 19 个 Agent 能力清理

顶层 `agents/` 的 19 个 Agent 应先做能力审计，而不是直接全部容器化。

建议分类：

```text
保留并服务化：code-agent、web-agent、review-agent、test-agent
保留但暂不服务化：document-agent、ppt-agent、release-agent
合并或降级为工具：过细、重复、没有独立运行价值的 Agent
删除或归档：只剩样例代码、没有测试、没有明确 capability 的 Agent
```

### 16.7 前端编排可视化

后续可增强：

```text
1. Orchestrator planning 状态。
2. Agent task card。
3. Plan 展开视图。
4. 每个 Agent 的输出分段。
5. 部分失败提示。
6. Artifact 与 Agent 输出绑定。
```

但前端增强不应阻塞本轮后端 Orchestrator 服务拆分。

---

## 17. 参考资料

### 17.1 仓库事实来源

- GitHub 仓库根目录：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/tree/dev`
- `services/` 目录：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/tree/dev/services`
- `docker-compose.new-arch.yml`：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/blob/dev/docker-compose.new-arch.yml`
- `go.work`：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/blob/dev/go.work`
- `services/gateway/runservice`：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/tree/dev/services/gateway/runservice`
- `docs/contracts`：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/tree/dev/docs/contracts`
- `.claude/skills`：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/tree/dev/.claude/skills`
- `.agents/skills`：`https://github.com/zxc1a1a1/Multi_Agent-AgentHub/tree/dev/.agents/skills`

### 17.2 Skill 调用参考

- Claude Code Skills 文档：`https://code.claude.com/docs/en/skills`
- OpenAI Codex Agent Skills 文档：`https://developers.openai.com/codex/skills`

---

## 18. 最终执行口径

本轮只认一个核心成果：

```text
Gateway 不再直接编排 Agent；Orchestrator 成为真实服务；所有执行先经过 Plan 与 Validator。
```

开发顺序必须是：

```text
事实审计
  ↓
空 Orchestrator
  ↓
Gateway 调 Orchestrator
  ↓
Orchestrator 接管 Registry / Dispatcher
  ↓
RulePlanner + OrchestrationPlan
  ↓
PlanValidator
  ↓
single executor
  ↓
ordered_parallel executor
  ↓
compose + smoke test
  ↓
最终架构审计
```

审计不通过，不进入下一阶段。

