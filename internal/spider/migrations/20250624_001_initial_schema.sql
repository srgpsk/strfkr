-- Initial spider database schema

-- Target websites configuration
CREATE TABLE spider_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    website_url TEXT NOT NULL UNIQUE,
    sitemap_url TEXT,
    follow_sitemap BOOLEAN DEFAULT true,
    last_visited_at DATETIME,
    is_active BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Rate limiting settings
    crawl_delay_seconds INTEGER DEFAULT 1,
    max_concurrent_requests INTEGER DEFAULT 5,
    
    -- Additional metadata
    user_agent TEXT,
    custom_headers TEXT, -- JSON string for custom headers
    notes TEXT
);

-- Crawled pages storage
CREATE TABLE spider_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    url_path TEXT NOT NULL, -- relative path from domain
    full_url TEXT NOT NULL,
    html_content TEXT,
    content_hash TEXT, -- SHA256 of content for change detection
    http_status_code INTEGER,
    response_time_ms INTEGER,
    content_length INTEGER,
    last_visited_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    first_discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    visit_count INTEGER DEFAULT 1,
    
    FOREIGN KEY (target_id) REFERENCES spider_targets(id) ON DELETE CASCADE,
    UNIQUE(target_id, url_path)
);

-- Simple SQLite-based queue
CREATE TABLE spider_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    status TEXT DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    processed_at DATETIME,
    error_message TEXT,
    
    FOREIGN KEY (target_id) REFERENCES spider_targets(id) ON DELETE CASCADE
);

-- Logging system
CREATE TABLE spider_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    log_type TEXT NOT NULL, -- 'manager', 'worker', 'error'
    target_id INTEGER,
    url TEXT,
    message TEXT NOT NULL,
    details TEXT, -- JSON string for additional data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (target_id) REFERENCES spider_targets(id) ON DELETE SET NULL
);

-- Configuration management
CREATE TABLE spider_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_spider_pages_target_id ON spider_pages(target_id);
CREATE INDEX idx_spider_pages_content_hash ON spider_pages(content_hash);
CREATE INDEX idx_spider_queue_status ON spider_queue(status);
CREATE INDEX idx_spider_queue_target_id ON spider_queue(target_id);
CREATE INDEX idx_spider_logs_type ON spider_logs(log_type);
CREATE INDEX idx_spider_logs_created_at ON spider_logs(created_at);

-- Default configuration values
INSERT INTO spider_config (key, value, description) VALUES 
('default_user_agent', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36', 'Default user agent string'),
('max_page_size_mb', '10', 'Maximum page size to download in MB'),
('connection_timeout_seconds', '30', 'HTTP connection timeout'),
('default_crawl_delay', '1', 'Default delay between requests in seconds'),
('max_concurrent_workers', '5', 'Maximum concurrent spider workers'),
('queue_batch_size', '10', 'Number of URLs to process in each batch');