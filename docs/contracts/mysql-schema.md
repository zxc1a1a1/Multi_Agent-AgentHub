# MySQL Schema Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：冻结 Gateway / Orchestrator / Memory / Runtime 表与唯一约束）

## 1. 实例与逻辑 schema

- 可共用一个 MySQL 8.4 LTS 实例，但使用独立逻辑 schema 与账号：
  - `agenthub_gateway`：会话、用户消息、请求幂等、运行视图、投影位置。
  - `agenthub_orchestrator`：executions、task、plan、approval、tool ledger、checkpoint、event journal、outbox、catalog。
  - `agenthub_memory`：scope、entries、revisions、mutations。
  - `agenthub_runtime`：sessions、session_events、session_context_snapshots。
- 模块不得直接改对方表；集成/迁移脚本按所有者拆分。
- 所有表 `ENGINE=InnoDB`、`utf8mb4`、时间列 `DATETIME(3)`。
- 不在模型推理、等待审批或外部工具执行期间保持数据库长事务。

## 2. Gateway schema（`agenthub_gateway`）

```sql
CREATE TABLE conversations (
  id VARCHAR(64) PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  conversation_type VARCHAR(32) NOT NULL DEFAULT 'single',
  agent_name VARCHAR(128) NULL,
  workspace_id VARCHAR(128) NOT NULL DEFAULT 'default',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_conversations_updated (updated_at DESC)
) ENGINE=InnoDB;

CREATE TABLE messages (
  id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  sender_type VARCHAR(32) NOT NULL,
  sender_name VARCHAR(128) NULL,
  content LONGTEXT NULL,
  metadata JSON NULL,
  request_id VARCHAR(64) NULL,
  run_id VARCHAR(64) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_message_request (conversation_id, request_id),
  KEY idx_messages_conversation_created (conversation_id, created_at)
) ENGINE=InnoDB;

CREATE TABLE chat_requests (
  request_id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  user_message_id VARCHAR(64) NOT NULL,
  run_id VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'accepted',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_chat_requests_conversation (conversation_id, created_at)
) ENGINE=InnoDB;

CREATE TABLE run_views (
  run_id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  request_id VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL,
  last_seq BIGINT NOT NULL DEFAULT 0,
  agent_name VARCHAR(128) NULL,
  error_code VARCHAR(64) NULL,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_run_views_conversation (conversation_id, updated_at)
) ENGINE=InnoDB;

CREATE TABLE event_projection_offsets (
  run_id VARCHAR(64) NOT NULL,
  consumer_key VARCHAR(128) NOT NULL,
  last_seq BIGINT NOT NULL DEFAULT 0,
  last_sub_index INT NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, consumer_key)
) ENGINE=InnoDB;

CREATE TABLE artifact_views (
  artifact_id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  conversation_id VARCHAR(64) NOT NULL,
  artifact_type VARCHAR(64) NOT NULL,
  title VARCHAR(255) NULL,
  storage_ref VARCHAR(512) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_artifact_views_run (run_id)
) ENGINE=InnoDB;
```

规则：

- 请求以稳定 `requestId` 幂等；Gateway 在本地事务写入用户消息 + `chat_requests`。
- 调用 Orchestrator 超时后按同一 `requestId` 重试/查询，不创建第二条用户消息或新运行。
- Gateway 与 Orchestrator 两次事务不假装跨服务原子。
- Assistant 文本投影与 `event_projection_offsets` 在同一事务更新，至少一次投递不得重复追加文本。

## 3. Orchestrator schema（`agenthub_orchestrator`）

```sql
CREATE TABLE executions (
  run_id VARCHAR(64) PRIMARY KEY,
  request_id VARCHAR(64) NOT NULL,
  conversation_id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(128) NOT NULL,
  principal_id VARCHAR(128) NOT NULL,
  execution_path VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL,
  terminal_reason VARCHAR(64) NULL,
  current_seq BIGINT NOT NULL DEFAULT 0,
  plan_id VARCHAR(64) NULL,
  plan_revision INT NOT NULL DEFAULT 0,
  fencing_version BIGINT NOT NULL DEFAULT 0,
  lease_owner VARCHAR(128) NULL,
  lease_expires_at DATETIME(3) NULL,
  budget_json JSON NULL,
  deadline_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_executions_request (request_id),
  KEY idx_executions_status (status, updated_at)
) ENGINE=InnoDB;

CREATE TABLE task_executions (
  run_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  agent_name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  depends_on JSON NULL,
  result_ref VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, task_id)
) ENGINE=InnoDB;

CREATE TABLE plan_revisions (
  run_id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  revision INT NOT NULL,
  plan_json JSON NOT NULL,
  plan_hash CHAR(64) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, plan_id, revision)
) ENGINE=InnoDB;

CREATE TABLE approvals (
  approval_id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  plan_revision INT NOT NULL,
  tool_call_id VARCHAR(64) NULL,
  tool_version VARCHAR(32) NULL,
  args_hash CHAR(64) NULL,
  workspace_id VARCHAR(128) NOT NULL,
  principal_id VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  consumed_at DATETIME(3) NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_approvals_idempotency (idempotency_key),
  KEY idx_approvals_run_tool (run_id, tool_call_id),
  KEY idx_approvals_status (status, expires_at)
) ENGINE=InnoDB;

CREATE TABLE tool_calls (
  run_id VARCHAR(64) NOT NULL,
  tool_call_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NULL,
  tool_name VARCHAR(128) NOT NULL,
  tool_version VARCHAR(32) NOT NULL,
  args_hash CHAR(64) NOT NULL,
  idempotency_key VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL,
  approval_id VARCHAR(64) NULL,
  result_ref VARCHAR(512) NULL,
  result_json JSON NULL,
  fencing_version BIGINT NOT NULL,
  dispatched_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, tool_call_id),
  UNIQUE KEY uniq_tool_calls_idempotency (run_id, idempotency_key)
) ENGINE=InnoDB;

CREATE TABLE checkpoints (
  run_id VARCHAR(64) NOT NULL,
  version BIGINT NOT NULL,
  checkpoint_version INT NOT NULL DEFAULT 1,
  state_json JSON NOT NULL,
  budget_json JSON NOT NULL,
  fencing_version BIGINT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, version)
) ENGINE=InnoDB;

CREATE TABLE execution_events (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  payload_json JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_execution_events_seq (run_id, seq),
  KEY idx_execution_events_run_created (run_id, created_at)
) ENGINE=InnoDB;

CREATE TABLE outbox (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  available_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  published_at DATETIME(3) NULL,
  UNIQUE KEY uniq_outbox_run_seq (run_id, seq),
  KEY idx_outbox_pending (status, available_at)
) ENGINE=InnoDB;

CREATE TABLE agent_catalog (
  agent_name VARCHAR(128) PRIMARY KEY,
  version VARCHAR(32) NOT NULL,
  descriptor_json JSON NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB;
```

唯一约束与并发（冻结）：

- `requestId` 唯一：重复请求不重复创建运行。
- `(runId, seq)` 唯一：同 run 并发更新不能分配重复 seq。
- `(runId, toolCallId)` 唯一：工具账本单条真值。
- 审批 `idempotencyKey` 唯一；审批与 `toolCallId` 绑定且一次性消费。
- 执行版本 `fencing_version` 条件更新；旧版本不能更新终态或预算。
- 状态与 outbox 同事务提交；不跨模型/工具调用保持事务。

## 4. Memory schema（`agenthub_memory`）

```sql
CREATE TABLE memory_scopes (
  scope_id VARCHAR(64) PRIMARY KEY,
  env VARCHAR(32) NOT NULL,
  workspace_id VARCHAR(128) NOT NULL,
  principal_id VARCHAR(128) NOT NULL,
  session_id VARCHAR(64) NOT NULL DEFAULT '',
  agent_name VARCHAR(128) NOT NULL DEFAULT '',
  thread_id VARCHAR(64) NOT NULL DEFAULT '',
  mode VARCHAR(32) NOT NULL DEFAULT 'session_only',
  policy_json JSON NOT NULL,
  epoch BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_memory_scope (env, workspace_id, principal_id, session_id, agent_name, thread_id)
) ENGINE=InnoDB;

CREATE TABLE memory_entries (
  memory_id VARCHAR(64) PRIMARY KEY,
  scope_id VARCHAR(64) NOT NULL,
  kind VARCHAR(32) NOT NULL,
  logical_key VARCHAR(191) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'candidate',
  current_revision INT NOT NULL DEFAULT 0,
  content_hash CHAR(64) NOT NULL,
  source_refs JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  verified_at DATETIME(3) NULL,
  expires_at DATETIME(3) NULL,
  last_used_at DATETIME(3) NULL,
  UNIQUE KEY uniq_memory_logical_key (scope_id, logical_key),
  KEY idx_memory_scope_status (scope_id, status, expires_at)
) ENGINE=InnoDB;

CREATE TABLE memory_revisions (
  memory_id VARCHAR(64) NOT NULL,
  revision INT NOT NULL,
  content TEXT NOT NULL,
  source_refs JSON NOT NULL,
  extractor_version VARCHAR(64) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (memory_id, revision)
) ENGINE=InnoDB;

CREATE TABLE memory_mutations (
  mutation_id VARCHAR(64) PRIMARY KEY,
  scope_id VARCHAR(64) NOT NULL,
  memory_id VARCHAR(64) NULL,
  operation VARCHAR(32) NOT NULL,
  expected_revision INT NULL,
  status VARCHAR(16) NOT NULL,
  purge_status VARCHAR(16) NOT NULL DEFAULT 'pending',
  actor VARCHAR(128) NOT NULL,
  tool_call_id VARCHAR(64) NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  completed_at DATETIME(3) NULL,
  UNIQUE KEY uniq_memory_mutation_idempotency (idempotency_key),
  KEY idx_memory_mutations_scope (scope_id, status)
) ENGINE=InnoDB;
```

规则：

- 状态 `active | candidate | revoked | expired`；候选不可被 search 或模型读取。
- CAS 使用 `current_revision` 与 `expectedRevision`；冲突返回 `MEMORY_REVISION_CONFLICT`，不覆盖新事实。
- 遗忘写 tombstone + `scope epoch++` + mutation；`purge_status` 独立返回。
- 全文索引（如 `FULLTEXT(content) WITH PARSER ngram`）可按实际语料启用；实现不得把它描述为向量语义搜索。

## 5. Runtime session schema（`agenthub_runtime`）

```sql
CREATE TABLE sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(128) NULL,
  workspace_id VARCHAR(128) NOT NULL DEFAULT 'default',
  state_json JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB;

CREATE TABLE session_events (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  author VARCHAR(128) NULL,
  content_json JSON NOT NULL,
  actions_json JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_session_seq (session_id, seq)
) ENGINE=InnoDB;

CREATE TABLE session_context_snapshots (
  snapshot_id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  agent_name VARCHAR(128) NOT NULL DEFAULT '',
  version BIGINT NOT NULL,
  source_event_from BIGINT NOT NULL,
  source_event_to BIGINT NOT NULL,
  summary MEDIUMTEXT NOT NULL,
  memory_refs JSON NULL,
  model VARCHAR(128) NULL,
  template_version VARCHAR(64) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_session_snapshot_version (session_id, agent_name, version)
) ENGINE=InnoDB;
```

`SQLSessionService` 必须完整实现 `adk.SessionService`（含 `GetOrCreate`），不得静默改变 `Get` 语义。

## 6. 版本与许可证锁定（P0 冻结责任）

| 组件 | 目标 | 锁定/复核 |
|---|---|---|
| Go | `go.work` = go 1.26.0 | P2 不升级 toolchain；Dockerfile 与 go.mod 对齐 |
| MySQL | 8.4 LTS 受支持补丁 | 镜像固定精确 tag，不用 `latest`；部署文档记录补丁与许可证（GPLv2，服务端使用） |
| Redis 协议服务器 | Redis 7.2.x（BSD-3-Clause）或 Valkey 8.x（BSD-3-Clause） | 固定精确 tag；如改用 Redis 7.4+/8.x 需先复核 RSALv2/SSPLv1/AGPLv3 许可 |
| MySQL 驱动 | `github.com/go-sql-driver/mysql` v1.9.x | MPL-2.0；版本写入 go.mod 并审查 |
| Redis 客户端 | `github.com/redis/go-redis/v9` | BSD-2-Clause；版本写入 go.mod |
| gRPC | `google.golang.org/grpc` | Apache-2.0；proto 生成物与运行时版本锁定 |

版本升级必须单独审查并更新本表；本表不是“已安装”证据。

## 7. 迁移与数据转换

- 迁移文件按 schema 所有者拆分；执行记录与版本号可查询。
- SQLite→MySQL 转换测试使用合成数据验证字段映射；真实用户库导入留到 P7 且需明确授权。
- 回滚边界：代码回滚不等于数据库回滚；新 schema 接受新写后不得直接启动旧 SQLite/规则模式。
