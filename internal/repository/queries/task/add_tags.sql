-- Привязывает к задаче только теги этого пользователя: чужие и несуществующие id отсекаются,
-- вызывающий сверяет число вставленных строк с числом переданных id.
INSERT INTO task_tags (task_id, tag_id)
SELECT $1, id
FROM tags
WHERE user_id = $2 AND id = ANY($3);
