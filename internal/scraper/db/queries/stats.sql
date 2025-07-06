-- name: GetTargetCount :one
SELECT COUNT(*) FROM scraper_targets WHERE is_active = true;

-- name: GetPendingQueueCount :one
SELECT COUNT(*) FROM scraper_queue WHERE status = 'pending';

-- name: GetTotalPagesCount :one
SELECT COUNT(*) FROM scraper_pages;

-- name: GetRecentErrorsCount :one
SELECT COUNT(*) FROM scraper_queue 
WHERE status = 'failed' 
AND processed_at > datetime('now', '-24 hours');