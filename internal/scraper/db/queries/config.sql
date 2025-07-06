-- name: GetConfig :one
SELECT value FROM scraper_config WHERE key = ?;

-- name: SetConfig :exec
INSERT INTO scraper_config (key, value, description) 
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET 
    value = excluded.value, 
    updated_at = CURRENT_TIMESTAMP;

-- name: ListAllConfig :many
SELECT * FROM scraper_config ORDER BY key;