-- name: CreateTarget :one
INSERT INTO spider_targets (
    website_url, sitemap_url, follow_sitemap, crawl_delay_seconds, 
    max_concurrent_requests, user_agent, custom_headers, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTarget :one
SELECT * FROM spider_targets WHERE id = ?;

-- name: GetTargetByURL :one
SELECT * FROM spider_targets WHERE website_url = ?;

-- name: ListActiveTargets :many
SELECT * FROM spider_targets WHERE is_active = true ORDER BY created_at;

-- name: UpdateTargetLastVisited :exec
UPDATE spider_targets SET last_visited_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
WHERE id = ?;

-- name: DeactivateTarget :exec
UPDATE spider_targets SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = ?;