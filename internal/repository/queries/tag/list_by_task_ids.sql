SELECT tt.task_id, t.id, t.name, t.color
FROM task_tags tt
JOIN tags t ON t.id = tt.tag_id
WHERE tt.task_id = ANY($1)
ORDER BY t.name;
