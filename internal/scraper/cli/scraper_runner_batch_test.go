package cli

import (
	"app/internal/scraper/db"
	"context"
	"database/sql"
	"testing"
)

func TestBatchEnqueueURLs(t *testing.T) {
	dbConn, _ := sql.Open("sqlite3", ":memory:")
	runMigrations(t, dbConn, "../../scraper/db/migrations")
	queries := &dbQueriesAdapter{q: db.New(dbConn)}
	sr := &ScraperRunner{
		db:        dbConn,
		queries:   queries,
		workers:   1,
		batchSize: 3,
	}

	urls := []string{"a", "b", "c", "d", "e", "f", "g"}
	count, err := sr.BatchEnqueueURLs(context.Background(), 1, urls, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != len(urls) {
		t.Errorf("expected %d enqueued, got %d", len(urls), count)
	}
}
