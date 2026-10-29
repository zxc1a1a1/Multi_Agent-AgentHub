# ADK Runtime Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

`pkg/adk` is a pure engine package. `pkg/runtime` is the engineering framework over ADK.

## ADK owns

- `Agent` interface
- `Model` interface
- `Tool` interface
- `Plugin` interface
- `Runner`
- `Session` / `SessionService` interface
- `Event`, `Content`, `Part` types
- `pkg/adk/a2a` adapter

## ADK must not own

- concrete LLM providers
- YAML config loading
- MySQL implementation details beyond interfaces
- Skill repository/manager
- AG-UI browser translation
- business-specific Agent handlers

## Runtime owns

- YAML configuration
- model/tool/agent registries
- Anthropic/OpenAI/Proxy providers
- session implementations including SQL and memory wrapper
- context pruning
- skill system
- AG-UI translator and filters
- launcher

## Interface requirement

Every runtime session implementation must satisfy `adk.SessionService`, including `GetOrCreate` if the ADK interface requires it.

## v2 Harness boundary

- ADK 仍是纯执行引擎；Runtime Harness（`pkg/runtime/harness`）在模型调用与工具副作用边界实施预算、权限、checkpoint、恢复与终态。
- 不在 ADK 内加入 MySQL/Redis/MCP 客户端或具体工具。
- 同一模型/工具循环只有一个执行所有者；不允许在现有 Runner 外再套第二套同级无界循环。
- 工具调用与事件落库顺序必须可证明：结果/账本先持久化，再对外宣告完成。
- 需要恢复时使用版本化 checkpoint（见 `runtime-harness.md`），不把 `session_events` 原样搬到 Redis 冒充恢复。
- Child Agent 通过 A2A 接收受限工具许可并回报结果，不直接改 Orchestrator 执行表。

## v2 Tool boundary

- 工具治理见 `tool-broker.md`：ToolDescriptor、ToolCallEnvelope、审批绑定、workspace 隔离与 `unknown` 语义。
- ADK `Tool` 接口可被 Broker 包装；模型可见 schema 只来自当前授权集合。
- 原生工具优先，MCP 只作为受控扩展；工具注解不是授权。
