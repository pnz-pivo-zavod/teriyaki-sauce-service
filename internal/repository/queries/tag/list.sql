SELECT id, name, color
FROM tags
WHERE user_id = $1
ORDER BY name;
