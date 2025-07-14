-- Remove columns for quote page classifier
ALTER TABLE scraper_pages DROP COLUMN quote_classifier_json;
ALTER TABLE scraper_pages DROP COLUMN processable;
