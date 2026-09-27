-- Фильтры опциональны: NULL в параметре отключает условие.
-- При фильтре по дате задачи без date не попадают (NULL >= x ложно).
SELECT id, name, description, date, notify_at, priority, is_completed
FROM tasks
WHERE user_id = $1
  AND ($2::timestamptz IS NULL OR date >= $2)
  AND ($3::timestamptz IS NULL OR date < $3)
  AND ($4::boolean IS NULL OR is_completed = $4)
ORDER BY date NULLS LAST, id;
