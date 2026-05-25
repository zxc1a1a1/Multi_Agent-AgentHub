# MySQL 当前 Schema 建议

## 目的

本文件给出 v1.0 当前 MySQL 8 Profile 的推荐表结构。字段可根据现有代码渐进迁移，但语义不应倒退到单 Agent / 单聊专用模型。

## 推荐表

```sql
CREATE TABLE users (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE conversations (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  title VARCHAR(255) NOT NULL DEFAULT 'New Conversation',
  conversation_type VARCHAR(32) NOT NULL DEFAULT 'single',
  primary_agent_name VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
  is_archived BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL,
  INDEX idx_conversations_user_updated (user_id, updated_at),
  INDEX idx_conversations_type (conversation_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE conversation_participants (
  id CHAR(36) PRIMARY KEY,
  conversation_id CHAR(36) NOT NULL,
  participant_type VARCHAR(32) NOT NULL,
  participant_id VARCHAR(128) NOT NULL,
  display_name VARCHAR(255) NULL,
  role VARCHAR(32) NOT NULL DEFAULT 'member',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  left_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_conv_participants_conv (conversation_id),
  INDEX idx_conv_participants_identity (participant_type, participant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE messages (
  id CHAR(36) PRIMARY KEY,
  conversation_id CHAR(36) NOT NULL,
  run_id CHAR(36) NULL,
  sender_type VARCHAR(32) NOT NULL,
  sender_id VARCHAR(128) NULL,
  sender_name VARCHAR(255) NULL,
  content LONGTEXT NULL,
  content_format VARCHAR(32) NOT NULL DEFAULT 'text',
  status VARCHAR(32) NOT NULL DEFAULT 'sent',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL,
  INDEX idx_messages_conversation_created (conversation_id, created_at),
  INDEX idx_messages_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE agents (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(128) NOT NULL UNIQUE,
  display_name VARCHAR(255) NULL,
  description TEXT NULL,
  url VARCHAR(1024) NOT NULL,
  version VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'enabled' COMMENT 'enabled|disabled|experimental|deprecated — 生命周期/启用状态',
  health VARCHAR(32) NOT NULL DEFAULT 'unknown' COMMENT 'healthy|degraded|unhealthy|unknown — 归一化后健康状态',
  agent_card JSON NULL,
  skills JSON NULL,
  input_modes JSON NULL,
  output_modes JSON NULL,
  last_check_at DATETIME NULL,
  last_error_code VARCHAR(128) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL,
  INDEX idx_agents_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE agent_health_checks (
  id CHAR(36) PRIMARY KEY,
  agent_name VARCHAR(128) NOT NULL,
  agent_url VARCHAR(1024) NOT NULL,
  status VARCHAR(32) NOT NULL COMMENT 'healthy|degraded|unhealthy|unknown — 归一化后健康状态',
  latency_ms INT NULL,
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  checked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_agent_health_agent_checked (agent_name, checked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE runs (
  id CHAR(36) PRIMARY KEY,
  conversation_id CHAR(36) NOT NULL,
  user_id CHAR(36) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'accepted' COMMENT 'accepted|running|completed|failed|cancelled — 粗粒度生命周期状态',
  phase VARCHAR(32) NULL COMMENT '可选细粒度当前阶段，如 planning/dispatching/aggregating',
  strategy VARCHAR(32) NULL,
  intent_summary TEXT NULL,
  trace_id VARCHAR(128) NULL,
  request_id VARCHAR(128) NULL,
  started_at DATETIME NULL,
  finished_at DATETIME NULL,
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_runs_conversation_created (conversation_id, created_at),
  INDEX idx_runs_trace (trace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE run_steps (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NOT NULL,
  step_type VARCHAR(64) NOT NULL COMMENT 'planning|dispatch|agent_call|tool_call|artifact|retry|fallback|aggregate',
  step_order INT NOT NULL DEFAULT 0,
  agent_name VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  input_summary TEXT NULL,
  output_summary TEXT NULL,
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  started_at DATETIME NULL,
  finished_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_run_steps_run_order (run_id, step_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE agent_tasks (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NOT NULL,
  step_id CHAR(36) NULL,
  agent_name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  task_ref VARCHAR(255) NULL,
  input_summary TEXT NULL,
  output_summary TEXT NULL,
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  started_at DATETIME NULL,
  finished_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_agent_tasks_run (run_id),
  INDEX idx_agent_tasks_agent_created (agent_name, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE artifacts (
  id CHAR(36) PRIMARY KEY,
  conversation_id CHAR(36) NOT NULL,
  message_id CHAR(36) NULL,
  run_id CHAR(36) NULL,
  agent_name VARCHAR(128) NULL,
  type VARCHAR(64) NOT NULL,
  title VARCHAR(255) NULL,
  mime_type VARCHAR(128) NULL,
  content LONGTEXT NULL,
  content_ref VARCHAR(1024) NULL,
  metadata JSON NULL,
  version INT NOT NULL DEFAULT 1,
  status VARCHAR(32) NOT NULL DEFAULT 'ready',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL,
  INDEX idx_artifacts_message (message_id),
  INDEX idx_artifacts_run (run_id),
  INDEX idx_artifacts_agent_created (agent_name, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
CREATE TABLE tool_calls (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NULL,
  message_id CHAR(36) NULL,
  artifact_id CHAR(36) NULL,
  tool_name VARCHAR(128) NOT NULL,
  args JSON NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tool_calls_message (message_id),
  INDEX idx_tool_calls_artifact (artifact_id),
  INDEX idx_tool_calls_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## 兼容说明

- 如果当前项目已有旧表，应使用 migration 渐进升级。
- 旧的 `a2a_tasks` 表可在兼容期继续存在，但新契约语义建议统一为 `agent_tasks`。
- 不建议只改 `init.sql` 而不提供迁移记录。
