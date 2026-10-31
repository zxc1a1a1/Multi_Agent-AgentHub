-- 003_conversation_participants.sql
-- Group-chat roster. The user owns the member list; the platform never adds a
-- participant that was not selected. (conversation_id, agent_name) is unique so
-- replaying a create/update request cannot duplicate a member.

CREATE TABLE IF NOT EXISTS conversation_participants (
  conversation_id VARCHAR(64) NOT NULL,
  agent_name VARCHAR(128) NOT NULL,
  role VARCHAR(32) NOT NULL DEFAULT 'member',
  selected TINYINT(1) NOT NULL DEFAULT 1,
  required TINYINT(1) NOT NULL DEFAULT 0,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (conversation_id, agent_name),
  KEY idx_conversation_participants_agent (agent_name),
  CONSTRAINT fk_conversation_participants_conversation FOREIGN KEY (conversation_id)
    REFERENCES conversations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
