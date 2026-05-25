# Run 与 Task 规则

## Run

Run 表示一次用户请求触发的运行。

一个 Run 可以包含：

- planning。
- dispatch。
- 多个 AgentTask。
- 多条 message。
- 多个 artifact。
- 多个 tool call。
- retry / fallback。

## runs 字段

```text
id
conversation_id
user_id
status
strategy
intent_summary
trace_id
request_id
started_at
finished_at
error_code
error_message
```

## run_steps 字段

```text
run_id
step_type
step_order
agent_name
status
input_summary
output_summary
error_code
error_message
started_at
finished_at
```

## agent_tasks 字段

```text
run_id
step_id
agent_name
status
task_ref
input_summary
output_summary
error_code
error_message
started_at
finished_at
```

## 状态

推荐状态：

```text
pending
running
succeeded
failed
cancelled
retrying
fallback
```

## 策略

推荐策略：

```text
single
parallel
sequential
```

## 禁止

- Run 绑定唯一 Agent。
- AgentTask 只存在内存。
- 失败时不记录 error_code。
- 保存完整敏感 prompt 到 input_summary。
- 只保存最终回复，丢失步骤链路。
