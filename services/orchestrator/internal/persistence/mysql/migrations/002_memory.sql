-- 002_memory.sql
-- AgentHub v2 MemoryController authoritative storage.

CREATE TABLE IF NOT EXISTS memory_scopes (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS memory_entries (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS memory_revisions (
  memory_id VARCHAR(64) NOT NULL,
  revision INT NOT NULL,
  content TEXT NOT NULL,
  source_refs JSON NOT NULL,
  extractor_version VARCHAR(64) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (memory_id, revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS memory_mutations (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
