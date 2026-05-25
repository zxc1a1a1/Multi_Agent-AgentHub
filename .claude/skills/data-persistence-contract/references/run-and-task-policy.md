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
phase
strategy
intent_summary
trace_id
request_id
started_at
finished_at
error_code
error_message
```

- `status`：粗粒度生命周期状态，只允许 `accepted | running | completed | failed | cancelled`。
- `phase`：可选细粒度当前阶段（如 `planning` / `dispatching` / `aggregating`），不等同于 `status`。

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

- `step_type`：持久化详细步骤类型：`planning | dispatch | agent_call | tool_call | artifact | retry | fallback | aggregate`。比 `runs.phase` 更细粒度，用于审计和排障。
- `status`（step 级别）：`pending | running | succeeded | failed`，表示单个步骤的执行结果。

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

## 状态分层

Run.status 是粗粒度生命周期状态（5 值）：

```text
accepted
running
completed
failed
cancelled
```

内部阶段（phase）到 status 的映射：

| 内部阶段 | Run.status |
|---|---|
| `accepted` | `accepted` |
| `context_loaded` ~ `aggregating` | `running` |
| `completed` | `completed` |
| `failed` | `failed` |
| `cancelled` | `cancelled` |

run_steps.step_type 是持久化详细步骤类型，用于审计：

```text
planning
dispatch
agent_call
tool_call
artifact
retry
fallback
aggregate
```

## 策略

推荐策略：

```text
single
ordered_parallel
sequential
```

legacy `parallel` / `ordered-parallel` 仅作为兼容输入别名，入库前归一化为 `ordered_parallel`。

## 禁止

- Run 绑定唯一 Agent。
- AgentTask 只存在内存。
- 失败时不记录 error_code。
- 保存完整敏感 prompt 到 input_summary。
- 只保存最终回复，丢失步骤链路。
- 将内部细粒度阶段（如 `planning`、`dispatching`）写入 `runs.status`。
- 通过扩展 `runs.status` 枚举代替使用 `phase` 或 `run_steps.step_type`。
