-- name: GetMessageByClientID :one
SELECT id, conversation_id, run_id, step_id, message_id, role, sender_type, sender_name, agent_name, client_message_id, content, content_json, reply_to_message_id, status, sequence, error_code, error_message, created_at, updated_at, deleted_at
FROM messages WHERE conversation_id = ? AND client_message_id = ? AND deleted_at IS NULL;

-- name: NextMessageSequence :one
SELECT COALESCE(MAX(sequence), 0) + 1 FROM messages WHERE conversation_id = ?;

-- name: CreateMessage :one
INSERT INTO messages (id, conversation_id, run_id, step_id, message_id, role, sender_type, sender_name, agent_name, content, content_json, reply_to_message_id, status, sequence, error_code, error_message, created_at, updated_at, metadata_json, client_message_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', ?)
RETURNING id, conversation_id, run_id, step_id, message_id, role, sender_type, sender_name, agent_name, client_message_id, content, content_json, reply_to_message_id, status, sequence, error_code, error_message, created_at, updated_at, deleted_at;

-- name: ListMessages :many
SELECT id, conversation_id, run_id, step_id, message_id, role, sender_type, sender_name, agent_name, client_message_id, content, content_json, reply_to_message_id, status, sequence, error_code, error_message, created_at, updated_at, deleted_at
FROM messages WHERE conversation_id = ? AND deleted_at IS NULL ORDER BY sequence ASC, id ASC LIMIT ? OFFSET ?;

-- name: SoftDeleteMessage :execrows
UPDATE messages SET deleted_at = ?, updated_at = ? WHERE id = ? AND conversation_id = ? AND deleted_at IS NULL;

-- name: UpdateMessageContentStatus :execrows
UPDATE messages SET content = ?, status = ?, error_code = ?, error_message = ?, updated_at = ? WHERE id = ?;
