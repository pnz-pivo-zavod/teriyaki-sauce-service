-- taskIDs уже отфильтрованы по пользователю вызывающим.
SELECT id, task_id, text, date
FROM notes
WHERE task_id = ANY($1)
ORDER BY date, id;
