-- AgentHub 2.0 batch 1 additive contract fields.
ALTER TABLE conversations ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE conversations ADD COLUMN mode TEXT NOT NULL DEFAULT 'direct';
ALTER TABLE conversations ADD COLUMN response_mode TEXT NOT NULL DEFAULT 'separate';
ALTER TABLE conversations ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE conversations ADD COLUMN last_message_at TEXT;
ALTER TABLE conversations ADD COLUMN archived_at TEXT;
ALTER TABLE conversations ADD COLUMN deleted_at TEXT;

ALTER TABLE messages ADD COLUMN client_message_id TEXT;
ALTER TABLE messages ADD COLUMN reply_to_message_id TEXT;
ALTER TABLE messages ADD COLUMN content_json TEXT;
ALTER TABLE messages ADD COLUMN sequence INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN deleted_at TEXT;

ALTER TABLE runs ADD COLUMN trigger_message_id TEXT;
ALTER TABLE runs ADD COLUMN mode TEXT NOT NULL DEFAULT 'direct';
ALTER TABLE runs ADD COLUMN plan_id TEXT;
ALTER TABLE runs ADD COLUMN confirmed_plan_version TEXT;
ALTER TABLE runs ADD COLUMN context_snapshot_id TEXT;
ALTER TABLE runs ADD COLUMN updated_at TEXT;

CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  invocation_id TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id),
  FOREIGN KEY (run_id) REFERENCES runs(id),
  UNIQUE (run_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_updated ON conversations(user_id, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_sequence ON messages(conversation_id, sequence ASC, id ASC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_messages_client_scope ON messages(conversation_id, client_message_id) WHERE client_message_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_runs_active_conversation ON runs(conversation_id) WHERE status IN ('pending','planning','awaiting_confirmation','executing','synthesizing');
CREATE INDEX IF NOT EXISTS idx_events_replay ON events(run_id, sequence ASC);
