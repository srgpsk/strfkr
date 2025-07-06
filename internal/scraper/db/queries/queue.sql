-- name: EnqueueURL :one
INSERT INTO scraper_queue (target_id, url, priority) 
VALUES (?, ?, ?)
RETURNING *;

-- name: DequeuePendingURL :one
UPDATE scraper_queue 
SET status = 'processing', processed_at = CURRENT_TIMESTAMP
WHERE id = (
    SELECT id FROM scraper_queue 
    WHERE status = 'pending' 
    ORDER BY priority DESC, created_at ASC 
    LIMIT 1
)
RETURNING *;

-- name: CompleteQueueItem :exec
UPDATE scraper_queue 
SET status = 'completed', processed_at = CURRENT_TIMESTAMP 
WHERE id = ?;

-- name: FailQueueItem :exec
UPDATE scraper_queue 
SET status = 'failed', attempts = attempts + 1, error_message = ?, processed_at = CURRENT_TIMESTAMP 
WHERE id = ?;

-- name: RetryFailedItem :exec
UPDATE scraper_queue 
SET status = 'pending', processed_at = NULL, error_message = NULL 
WHERE id = ? AND attempts < max_attempts;

-- name: GetQueueStats :one
SELECT 
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
FROM scraper_queue;