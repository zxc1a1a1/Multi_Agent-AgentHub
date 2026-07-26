-- name: CreateRunStep :exec
INSERT INTO run_steps (id, run_id, conversation_id, task_id, step_index, agent_name, capability_id, status, started_at, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '{}');

-- name: UpdateRunStepStatus :execrows
UPDATE run_steps SET status = ?, error_code = ?, error_message = ?, finished_at = ? WHERE id = ?;

-- name: ListRunStepsByRun :many
SELECT id, run_id, conversation_id, task_id, step_index, agent_name, capability_id, status, started_at, finished_at, error_code, error_message, metadata_json
FROM run_steps
WHERE run_id = ?
ORDER BY step_index ASC;
