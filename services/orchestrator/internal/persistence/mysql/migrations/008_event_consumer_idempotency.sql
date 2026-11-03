CREATE TABLE IF NOT EXISTS event_consumer_idempotency (
  consumer_key VARCHAR(128) NOT NULL,
  event_id VARCHAR(160) NOT NULL,
  event_hash CHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL,
  result_ref VARCHAR(255) NOT NULL DEFAULT '',
  attempts INT NOT NULL DEFAULT 1,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (consumer_key, event_id),
  KEY idx_event_consumer_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
