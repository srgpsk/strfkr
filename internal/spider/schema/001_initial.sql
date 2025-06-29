CREATE TABLE IF NOT EXISTS spider_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    website_url TEXT NOT NULL UNIQUE,
    sitemap_url TEXT NOT NULL,
    follow_sitemap BOOLEAN DEFAULT true,
    crawl_delay_seconds INTEGER DEFAULT 1,
    max_concurrent_requests INTEGER DEFAULT 5,
    user_agent TEXT,
    custom_headers TEXT, -- JSON string for custom headers
    notes TEXT,
    is_active BOOLEAN DEFAULT true,
    last_visited_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS spider_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    url TEXT NOT NULL UNIQUE,
    content_hash TEXT,
    html_content TEXT,
    title TEXT,
    last_modified DATETIME,
    content_length INTEGER,
    status_code INTEGER,
    crawled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (target_id) REFERENCES spider_targets(id)
);

CREATE TABLE IF NOT EXISTS spider_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending', -- pending, processing, completed, failed
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    processed_at DATETIME,
    FOREIGN KEY (target_id) REFERENCES spider_targets(id)
);

CREATE TABLE IF NOT EXISTS spider_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER,
    level TEXT NOT NULL, -- info, warn, error
    message TEXT NOT NULL,
    details TEXT, -- JSON data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (target_id) REFERENCES spider_targets(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_spider_pages_target_url ON spider_pages(target_id, url);
CREATE INDEX IF NOT EXISTS idx_spider_pages_hash ON spider_pages(content_hash);
CREATE INDEX IF NOT EXISTS idx_spider_queue_status ON spider_queue(status);
CREATE INDEX IF NOT EXISTS idx_spider_queue_priority ON spider_queue(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_spider_logs_created ON spider_logs(created_at);