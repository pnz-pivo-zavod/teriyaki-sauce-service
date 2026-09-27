UPDATE tasks
SET is_completed = TRUE
WHERE id = $1 AND user_id = $2;
