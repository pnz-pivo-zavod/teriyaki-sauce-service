INSERT INTO tags (user_id, name, color)
VALUES ($1, $2, $3)
RETURNING id;
