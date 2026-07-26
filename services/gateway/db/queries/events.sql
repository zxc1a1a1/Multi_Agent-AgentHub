-- name: NextEventSequence :one
SELECT COALESCE(MAX(sequence), 0) + 1 FROM events WHERE run_id = ?;

-- name: CreateEvent :one
INSERT INTO events (id, conversation_id, run_id, invocation_id, event_type, payload, sequence, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, conversation_id, run_id, invocation_id, event_type, payload, sequence, created_at;

-- name: ListEventsAfter :many
SELECT id, conversation_id, run_id, invocation_id, event_type, payload, sequence, created_at
FROM events WHERE run_id = ? AND sequence > ? ORDER BY sequence ASC LIMIT ?;
