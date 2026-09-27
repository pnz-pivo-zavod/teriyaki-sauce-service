UPDATE tags
SET name = $3, color = $4
WHERE id = $1 AND user_id = $2
RETURNING id;
