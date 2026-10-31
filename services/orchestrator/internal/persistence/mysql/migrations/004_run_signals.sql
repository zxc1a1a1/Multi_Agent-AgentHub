-- 004_run_signals.sql
-- Durable HITL signals. A confirm handled by any Orchestrator instance appends
-- a signal; the instance currently waiting for approval claims and executes it.
-- (run_id, idempotency_key) is unique so a retried confirm is appended once.

CREATE TABLE IF NOT EXISTS run_signals (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  action_id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  plan_revision INT NOT NULL DEFAULT 0,
  action VARCHAR(32) NOT NULL,
  feedback TEXT NULL,
  reject_reason VARCHAR(512) NULL,
  selected_participants JSON NULL,
  idempotency_key VARCHAR(128) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  consumed_at DATETIME(3) NULL,
  UNIQUE KEY uniq_run_signals_idempotency (run_id, idempotency_key),
  KEY idx_run_signals_pending (run_id, status, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
