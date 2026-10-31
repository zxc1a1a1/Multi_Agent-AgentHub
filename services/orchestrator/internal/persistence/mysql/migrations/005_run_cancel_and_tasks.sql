-- 005_run_cancel_and_tasks.sql
-- Cross-instance cancellation: any instance can record a cancel request and
-- locate the remote A2A tasks dispatched by the owning instance.

CREATE TABLE IF NOT EXISTS run_task_refs (
  run_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  agent_name VARCHAR(128) NOT NULL,
  agent_url VARCHAR(512) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, task_id, agent_name),
  KEY idx_run_task_refs_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS run_cancels (
  run_id VARCHAR(64) PRIMARY KEY,
  requested_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  consumed_at DATETIME(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
