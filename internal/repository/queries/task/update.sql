UPDATE tasks
SET name = $3, description = $4, date = $5, notify_at = $6, priority = $7, is_completed = $8
WHERE id = $1 AND user_id = $2;
