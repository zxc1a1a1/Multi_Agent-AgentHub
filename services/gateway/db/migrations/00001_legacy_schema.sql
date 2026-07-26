-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS run_steps;
DROP TABLE IF EXISTS runs;
DROP TABLE IF EXISTS conversation_participants;
DROP TABLE IF EXISTS conversations;
