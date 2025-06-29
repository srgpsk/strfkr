-- name: SavePage :one
INSERT INTO spider_pages (
    target_id, url_path, full_url, html_content, content_hash, 
    http_status_code, response_time_ms, content_length
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(target_id, url_path) DO UPDATE SET
    html_content = excluded.html_content,
    content_hash = excluded.content_hash,
    http_status_code = excluded.http_status_code,
    response_time_ms = excluded.response_time_ms,
    content_length = excluded.content_length,
    last_visited_at = CURRENT_TIMESTAMP,
    visit_count = visit_count + 1
RETURNING *;

-- name: GetPageByPath :one
SELECT * FROM spider_pages WHERE target_id = ? AND url_path = ?;

-- name: ListPagesByTarget :many
SELECT * FROM spider_pages WHERE target_id = ? ORDER BY last_visited_at DESC LIMIT ?;

-- name: GetPageContentHash :one
SELECT content_hash FROM spider_pages WHERE target_id = ? AND url_path = ? LIMIT 1;