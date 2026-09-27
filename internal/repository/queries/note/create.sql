-- Вставляет заметку, только если задача принадлежит пользователю; иначе 0 строк.
INSERT INTO notes (task_id, text)
SELECT id, $3
FROM tasks
WHERE id = $1 AND user_id = $2
RETURNING id, task_id, text, date;
