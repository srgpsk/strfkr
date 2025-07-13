DROP INDEX IF EXISTS idx_scraper_pages_target_id;
DROP INDEX IF EXISTS idx_scraper_pages_content_hash;
DROP INDEX IF EXISTS idx_scraper_queue_status;
DROP INDEX IF EXISTS idx_scraper_queue_target_id;
DROP INDEX IF EXISTS idx_scraper_logs_type;
DROP INDEX IF EXISTS idx_scraper_logs_created_at;
DROP INDEX IF EXISTS idx_scraper_targets_domain;

DROP TABLE IF EXISTS scraper_pages;
DROP TABLE IF EXISTS scraper_queue;
DROP TABLE IF EXISTS scraper_logs;
DROP TABLE IF EXISTS scraper_targets;
DROP TABLE IF EXISTS scraper_config;
