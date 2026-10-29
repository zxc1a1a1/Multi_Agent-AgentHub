-- 001_execution_ledger.sql
-- AgentHub v2 Orchestrator execution ledger (MySQL 8.4).
-- Authoritative state for runs, approvals, tool calls, checkpoints, events and outbox.

CREATE TABLE IF NOT EXISTS executions (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_executions (
  run_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  agent_name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  depends_on JSON NULL,
  result_ref VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS plan_revisions (
  run_id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  revision INT NOT NULL,
  plan_json JSON NOT NULL,
  plan_hash CHAR(64) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, plan_id, revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS approvals (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tool_calls (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS checkpoints (
  run_id VARCHAR(64) NOT NULL,
  version BIGINT NOT NULL,
  checkpoint_version INT NOT NULL DEFAULT 1,
  state_json JSON NOT NULL,
  budget_json JSON NOT NULL,
  fencing_version BIGINT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS execution_events (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  payload_json JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_execution_events_seq (run_id, seq),
  KEY idx_execution_events_run_created (run_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS outbox (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS agent_catalog (
  agent_name VARCHAR(128) PRIMARY KEY,
  version VARCHAR(32) NOT NULL,
  descriptor_json JSON NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
