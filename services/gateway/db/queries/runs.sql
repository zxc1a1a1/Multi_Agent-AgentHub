-- name: CreateRun :one
INSERT INTO runs (id, conversation_id, trigger_message_id, mode, status, plan_id, confirmed_plan_version, context_snapshot_id, started_at, error_code, metadata_json, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', '{}', ?)
RETURNING id, conversation_id, trigger_message_id, mode, status, plan_id, confirmed_plan_version, context_snapshot_id, started_at, finished_at, error_code, metadata_json, created_at, updated_at;

-- name: GetRun :one
SELECT id, conversation_id, trigger_message_id, mode, status, plan_id, confirmed_plan_version, context_snapshot_id, started_at, finished_at, error_code, metadata_json, created_at, updated_at FROM runs WHERE id = ?;

-- name: ListRunsByConversation :many
SELECT id, conversation_id, trigger_message_id, mode, status, plan_id, confirmed_plan_version, context_snapshot_id, started_at, finished_at, error_code, metadata_json, created_at, updated_at FROM runs WHERE conversation_id = ? ORDER BY started_at DESC, id DESC LIMIT ? OFFSET ?;

-- name: CompareAndSetRunStatus :execrows
UPDATE runs SET status = ?, finished_at = ?, updated_at = ? WHERE id = ? AND status = ?;
