-- Add columns for quote page classifier
ALTER TABLE scraper_pages ADD COLUMN quote_classifier_json TEXT;
ALTER TABLE scraper_pages ADD COLUMN processable BOOLEAN DEFAULT 0;
