-- Заметки и привязки тегов удаляются каскадом.
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;
