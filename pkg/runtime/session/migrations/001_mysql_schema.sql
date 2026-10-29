-- 001_mysql_schema.sql
-- Runtime session storage for MySQL 8.4. Owner: pkg/runtime/session.

CREATE TABLE IF NOT EXISTS sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(128) NULL,
  workspace_id VARCHAR(128) NOT NULL DEFAULT 'default',
  state_json JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS session_events (
  id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  author VARCHAR(128) NULL,
  content_json JSON NOT NULL,
  actions_json JSON NULL,
  partial TINYINT(1) NOT NULL DEFAULT 0,
  final TINYINT(1) NOT NULL DEFAULT 0,
  timestamp DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uniq_session_seq (session_id, seq),
  KEY idx_session_events_session (session_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS session_context_snapshots (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
