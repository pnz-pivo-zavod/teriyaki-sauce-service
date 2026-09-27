-- Меняет текст и задачу заметки (перенос). И текущая, и новая задача должны принадлежать
-- пользователю; иначе 0 строк. date не меняется — это время создания.
UPDATE notes n
SET task_id = $3, text = $4
FROM tasks cur, tasks dst
WHERE n.id = $1
  AND cur.id = n.task_id AND cur.user_id = $2
  AND dst.id = $3 AND dst.user_id = $2
RETURNING n.id, n.task_id, n.text, n.date;
