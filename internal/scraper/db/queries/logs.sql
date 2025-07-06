-- name: LogMessage :exec
INSERT INTO scraper_logs (log_type, target_id, url, message, details)
VALUES (?, ?, ?, ?, ?);

-- name: GetRecentLogs :many
SELECT * FROM scraper_logs 
ORDER BY created_at DESC 
LIMIT ?;

-- name: GetLogsByTarget :many
SELECT * FROM scraper_logs 
WHERE target_id = ? 
ORDER BY created_at DESC 
LIMIT ?;

-- name: GetLogsByLevel :many
SELECT * FROM scraper_logs 
WHERE log_type = ? 
ORDER BY created_at DESC 
LIMIT ?;
