INSERT INTO tasks (user_id, name, description, date, notify_at, priority)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;
