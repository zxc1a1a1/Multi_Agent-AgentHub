-- 001_gateway.sql
-- AgentHub v2 Gateway authoritative store (MySQL 8.4).

CREATE TABLE IF NOT EXISTS conversations (
  id VARCHAR(64) PRIMARY KEY,
  title VARCHAR(255) NOT NULL DEFAULT '',
  conversation_type VARCHAR(32) NOT NULL DEFAULT 'single',
  agent_name VARCHAR(128) NULL,
  user_id VARCHAR(128) NOT NULL DEFAULT '',
  workspace_id VARCHAR(128) NOT NULL DEFAULT 'default',
  pinned TINYINT(1) NOT NULL DEFAULT 0,
  pinned_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_conversations_user_updated (user_id, updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS messages (
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
  KEY idx_messages_conversation_created (conversation_id, created_at),
  CONSTRAINT fk_messages_conversation FOREIGN KEY (conversation_id)
    REFERENCES conversations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS chat_requests (
  request_id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  user_message_id VARCHAR(64) NOT NULL,
  run_id VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'accepted',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_chat_requests_conversation (conversation_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS run_views (
  run_id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  request_id VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL,
  last_seq BIGINT NOT NULL DEFAULT 0,
  agent_name VARCHAR(128) NULL,
  error_code VARCHAR(64) NULL,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_run_views_conversation (conversation_id, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS event_projection_offsets (
  run_id VARCHAR(64) NOT NULL,
  consumer_key VARCHAR(128) NOT NULL,
  last_seq BIGINT NOT NULL DEFAULT 0,
  last_sub_index INT NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, consumer_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS artifact_views (
  artifact_id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  conversation_id VARCHAR(64) NOT NULL,
  artifact_type VARCHAR(64) NOT NULL,
  title VARCHAR(255) NULL,
  storage_ref VARCHAR(512) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_artifact_views_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
