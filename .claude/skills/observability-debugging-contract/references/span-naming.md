# Span 命名规则

## 1. 目的

本文定义 AgentHub OpenTelemetry span 命名建议。

正式开发阶段启用 OpenTelemetry 时使用。

## 2. 推荐 Span 名称

```text
agenthub.gateway.run
agenthub.gateway.sse
agenthub.orchestrator.plan
agenthub.orchestrator.execute
agenthub.a2a.send
agenthub.a2a.stream
agenthub.agent.handle_task
agenthub.llm.generate
agenthub.llm.stream
agenthub.artifact.normalize
agenthub.artifact.persist
agenthub.artifact.preview_map
agenthub.agui.emit
agenthub.frontend.receive
agenthub.frontend.render_tool
```

## 3. 推荐属性

```text
agenthub.run_id
agenthub.conversation_id
agenthub.message_id
agenthub.execution_plan_id
agenthub.step_id
agenthub.a2a_task_id
agenthub.agent_name
agenthub.agent_skill
agenthub.artifact_id
agenthub.tool_call_id
agenthub.llm_provider
agenthub.llm_model
agenthub.error_code
```

## 4. 规则

优先复用 OpenTelemetry 官方 Semantic Conventions 中已有的 HTTP、messaging、error、gen-ai 等属性。

AgentHub 自定义属性必须使用统一前缀：

```text
agenthub.
```

## 5. 禁止事项

不得：

- 每个服务发明不同 span 名称。
- 在 span attribute 中写 secret。
- 在 span name 中写用户输入。
- 在 attributes 中放高基数大文本。
