SELECT id, name, description, date, notify_at, priority, is_completed
FROM tasks
WHERE id = $1 AND user_id = $2;
