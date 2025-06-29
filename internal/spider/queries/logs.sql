-- name: LogMessage :exec
INSERT INTO spider_logs (log_type, target_id, url, message, details)
VALUES (?, ?, ?, ?, ?);

-- name: GetRecentLogs :many
SELECT * FROM spider_logs 
ORDER BY created_at DESC 
LIMIT ?;

-- name: GetLogsByTarget :many
SELECT * FROM spider_logs 
WHERE target_id = ? 
ORDER BY created_at DESC 
LIMIT ?;

-- name: GetLogsByLevel :many
SELECT * FROM spider_logs 
WHERE log_type = ? 
ORDER BY created_at DESC 
LIMIT ?;
