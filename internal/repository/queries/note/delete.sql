DELETE FROM notes n
USING tasks t
WHERE n.id = $1 AND t.id = n.task_id AND t.user_id = $2;
