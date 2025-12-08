-- name: UserGet :one
SELECT *
FROM users
WHERE id = $1;