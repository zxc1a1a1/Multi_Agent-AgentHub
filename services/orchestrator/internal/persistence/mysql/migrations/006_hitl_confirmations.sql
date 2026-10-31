-- 006_hitl_confirmations.sql
-- Cross-instance HITL confirmation idempotency. The first request with a key
-- owns processing; repeats receive the stored response without re-consuming the
-- approval or appending another signal.

CREATE TABLE IF NOT EXISTS hitl_confirmations (
  run_id VARCHAR(64) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  request_hash CHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'processing',
  http_status INT NOT NULL DEFAULT 0,
  response_json JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
