CREATE TABLE IF NOT EXISTS run_public_events (
  run_id VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  sub_index INT NOT NULL DEFAULT 0,
  event_type VARCHAR(64) NOT NULL,
  payload_json JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, seq, sub_index),
  KEY idx_run_public_events_created (run_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
