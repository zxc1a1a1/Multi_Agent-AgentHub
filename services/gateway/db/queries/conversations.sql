-- name: CreateConversation :one
INSERT INTO conversations (id, user_id, title, mode, response_mode, status, version, created_at, updated_at, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, '{}')
RETURNING id, user_id, title, mode, response_mode, status, version, last_message_at, pinned_at, archived_at, deleted_at, created_at, updated_at;

-- name: GetConversationForUser :one
SELECT id, user_id, title, mode, response_mode, status, version, last_message_at, pinned_at, archived_at, deleted_at, created_at, updated_at
FROM conversations WHERE id = ? AND user_id = ? AND deleted_at IS NULL;

-- name: ListConversationsForUser :many
SELECT id, user_id, title, mode, response_mode, status, version, last_message_at, pinned_at, archived_at, deleted_at, created_at, updated_at
FROM conversations WHERE user_id = ? AND deleted_at IS NULL
ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?;

-- name: UpdateConversationVersion :one
UPDATE conversations SET title = ?, mode = ?, response_mode = ?, version = version + 1, updated_at = ?
WHERE id = ? AND user_id = ? AND version = ? AND deleted_at IS NULL
RETURNING id, user_id, title, mode, response_mode, status, version, last_message_at, pinned_at, archived_at, deleted_at, created_at, updated_at;

-- name: SetConversationArchived :one
UPDATE conversations SET archived_at = ?, version = version + 1, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL
RETURNING id, user_id, title, mode, response_mode, status, version, last_message_at, pinned_at, archived_at, deleted_at, created_at, updated_at;

-- name: SetConversationPinned :one
UPDATE conversations SET pinned = ?, pinned_at = ?, version = version + 1, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL
RETURNING id, user_id, title, mode, response_mode, status, version, pinned, pinned_at, last_message_at, archived_at, deleted_at, created_at, updated_at;

-- name: TouchConversation :execrows
UPDATE conversations SET last_message_at = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL;

-- name: SoftDeleteConversation :execrows
UPDATE conversations SET deleted_at = ?, version = version + 1, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL;
