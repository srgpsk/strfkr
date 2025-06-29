-- name: GetConfig :one
SELECT value FROM spider_config WHERE key = ?;

-- name: SetConfig :exec
INSERT INTO spider_config (key, value, description) 
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET 
    value = excluded.value, 
    updated_at = CURRENT_TIMESTAMP;

-- name: ListAllConfig :many
SELECT * FROM spider_config ORDER BY key;