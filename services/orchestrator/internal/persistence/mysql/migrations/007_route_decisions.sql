-- 007_route_decisions.sql
-- Explainable routing: candidates, selected agents and rejection reasons for
-- each run. decision_seq allows future replans without overwriting history.

CREATE TABLE IF NOT EXISTS route_decisions (
  run_id VARCHAR(64) NOT NULL,
  decision_seq INT NOT NULL DEFAULT 0,
  execution_path VARCHAR(32) NOT NULL,
  strategy VARCHAR(32) NOT NULL,
  candidates_json JSON NOT NULL,
  selected_json JSON NOT NULL,
  rejected_json JSON NOT NULL,
  planner_source VARCHAR(32) NOT NULL,
  reason_summary VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (run_id, decision_seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
