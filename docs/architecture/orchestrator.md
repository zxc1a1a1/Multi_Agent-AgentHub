# Orchestrator 架构说明

## 1. Orchestrator 定位

Orchestrator 是 AgentHub 的统一 Agent / 编排器，使用 Go 实现。

它负责：

- 意图理解
- AgentCard 读取
- ExecutionPlan 生成
- A2A 调度
- 结果聚合
- 协议转换
- fallback

---

## MVP v0.1 Orchestrator 范围

MVP v0.1 中，Orchestrator 作为 Gateway 进程内模块存在。

MVP Orchestrator 负责：

- 从 AG-UI RunRequest 中读取目标 Agent。
- 直接路由到 `code-agent`。
- 调用 A2A `sendSubscribe`。
- 接收 A2A stream event。
- 将 A2A `status/text/artifact` 转换为 AG-UI `TEXT_MESSAGE_*` 和 `TOOL_CALL_*` 事件。
- 将 `code` Artifact 映射为 `code_preview` Tool Call。

MVP Orchestrator 暂不负责：

- LLM 意图分析。
- 多 Agent ExecutionPlan。
- 并行 / 串行任务编排。
- 多 Agent 结果聚合。
- 群聊 Agent 选择。

这些能力保留为 Post-MVP。

---

## 2. 推荐目录

```text
orchestrator/
  cmd/
    orchestrator/
      main.go
  internal/
    engine/
      intent_engine.go
      planner.go
      converter.go
    a2a/
      client.go
      types.go
    registry/
      agent_registry.go
    aggregator/
      result_aggregator.go
    llm/
      client.go
      prompts.go
    handler/
      internal_runs.go
  go.mod
  Dockerfile
```

---

## 3. 内部接口

Orchestrator 不对 Frontend 开放，只给 Gateway 调用：

```text
POST /internal/runs
POST /internal/runs/{runId}/tool-result
POST /internal/runs/{runId}/cancel
GET  /internal/runs/{runId}/events
```

---

## 4. ExecutionPlan

最小结构：

```json
{
  "intent": "用户意图摘要",
  "strategy": "single",
  "tasks": [
    {
      "agent_name": "code-agent",
      "task_content": "实现 Go 登录接口",
      "depends_on": [],
      "priority": 1
    }
  ],
  "fallback": null
}
```

strategy 只能是：

```text
single
parallel
sequential
```

---

## 5. 协议转换职责

A2A 文本流映射为：

```text
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
```

A2A status 映射为：

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR
STATE_UPDATE
```

A2A Artifact 映射为：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

---

## 6. Orchestrator 禁止事项

- 禁止暴露给 Frontend。
- 禁止依赖 React 组件。
- 禁止直接处理用户登录态。
- 禁止绕过 A2A 调 Child Agent。
- 禁止把 Artifact 当作纯文本返回。
