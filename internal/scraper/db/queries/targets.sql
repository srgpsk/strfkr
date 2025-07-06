-- name: CreateTarget :one
INSERT INTO scraper_targets (
    website_url, sitemap_url, follow_sitemap, crawl_delay_seconds, 
    max_concurrent_requests, user_agent, custom_headers, notes,
    sitemap_patterns, url_patterns, domain_name
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTarget :one
SELECT * FROM scraper_targets WHERE id = ?;

-- name: GetTargetByURL :one
SELECT * FROM scraper_targets WHERE website_url = ?;

-- name: GetTargetByDomain :one
SELECT * FROM scraper_targets WHERE domain_name = ?;

-- name: ListActiveTargets :many
SELECT * FROM scraper_targets WHERE is_active = true ORDER BY created_at;

-- name: ListAllTargets :many
SELECT * FROM scraper_targets ORDER BY created_at;

-- name: UpdateTargetLastVisited :exec
UPDATE scraper_targets SET last_visited_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
WHERE id = ?;

-- name: UpdateTargetPatterns :exec
UPDATE scraper_targets 
SET sitemap_patterns = ?, url_patterns = ?, updated_at = CURRENT_TIMESTAMP 
WHERE id = ?;

-- name: DeactivateTarget :exec
UPDATE scraper_targets SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = ?;