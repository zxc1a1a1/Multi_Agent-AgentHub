-- 001_initial_schema.sql
-- AgentHub v1.0 persistence foundation: conversations, participants, runs, steps, messages, artifacts.

CREATE TABLE IF NOT EXISTS conversations (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    metadata_json   TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at);

CREATE TABLE IF NOT EXISTS conversation_participants (
    id               TEXT PRIMARY KEY,
    conversation_id  TEXT NOT NULL,
    participant_type TEXT NOT NULL,
    participant_name TEXT NOT NULL,
    display_name     TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_participants_conversation ON conversation_participants(conversation_id);

CREATE TABLE IF NOT EXISTS runs (
    id               TEXT PRIMARY KEY,
    conversation_id  TEXT NOT NULL,
    status           TEXT NOT NULL,
    planning_mode    TEXT NOT NULL DEFAULT '',
    started_at       TEXT NOT NULL,
    finished_at      TEXT,
    error_code       TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_runs_conversation_started ON runs(conversation_id, started_at);

CREATE TABLE IF NOT EXISTS run_steps (
    id               TEXT PRIMARY KEY,
    run_id           TEXT NOT NULL,
    conversation_id  TEXT NOT NULL,
    task_id          TEXT NOT NULL DEFAULT '',
    step_index       INTEGER NOT NULL,
    agent_name       TEXT NOT NULL DEFAULT '',
    capability_id    TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL,
    started_at       TEXT,
    finished_at      TEXT,
    error_code       TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_run_steps_run ON run_steps(run_id, step_index);
CREATE INDEX IF NOT EXISTS idx_run_steps_task_id ON run_steps(task_id);

CREATE TABLE IF NOT EXISTS messages (
    id               TEXT PRIMARY KEY,
    conversation_id  TEXT NOT NULL,
    run_id           TEXT,
    step_id          TEXT,
    message_id       TEXT NOT NULL DEFAULT '',
    role             TEXT NOT NULL,
    sender_type      TEXT NOT NULL,
    sender_name      TEXT NOT NULL DEFAULT '',
    agent_name       TEXT NOT NULL DEFAULT '',
    content          TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL,
    error_code       TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (step_id) REFERENCES run_steps(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_run_id ON messages(run_id);
CREATE INDEX IF NOT EXISTS idx_messages_step_id ON messages(step_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);

CREATE TABLE IF NOT EXISTS artifacts (
    id               TEXT PRIMARY KEY,
    conversation_id  TEXT NOT NULL,
    run_id           TEXT,
    step_id          TEXT,
    message_id       TEXT,
    artifact_type    TEXT NOT NULL,
    title            TEXT NOT NULL DEFAULT '',
    mime_type        TEXT NOT NULL DEFAULT '',
    preview_type     TEXT NOT NULL DEFAULT '',
    content_ref      TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (step_id) REFERENCES run_steps(id)
);

CREATE INDEX IF NOT EXISTS idx_artifacts_conversation ON artifacts(conversation_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_run_id ON artifacts(run_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_message_id ON artifacts(message_id);

ALTER TABLE conversations ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversations ADD COLUMN pinned_at TEXT;

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
ALTER TABLE runs ADD COLUMN created_at TEXT;
UPDATE runs SET created_at = started_at WHERE created_at IS NULL;

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

