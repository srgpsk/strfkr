package cli

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"app/internal/scraper/db"
	"app/internal/scraper/sitemap"

	"github.com/cespare/xxhash/v2"
)

// --- TEST FIXES ---
// 1. For SkipUpdateLogic: Use the test server's base URL for all queue and page records.
// 2. For ParseAndQueueURLs_HTTP: Use a mockQueries with a working EnqueueURL method.

// mockDB implements minimal db.Queries for testing
// Only methods used by ScraperRunner are implemented

// mockQueries implements ScraperQueries for testing
// No interface assertion here since WithTx returns *db.Queries

type mockQueries struct {
	EnqueueURLCalls   []db.EnqueueURLParams
	EnqueueURLErr     error
	GetTargetResp     db.ScraperTarget
	GetTargetErr      error
	ListActiveResp    []db.ScraperTarget
	ListActiveErr     error
	GetQueueStatsResp db.GetQueueStatsRow
	GetQueueStatsErr  error
}

func (m *mockQueries) EnqueueURL(ctx context.Context, params db.EnqueueURLParams) (db.ScraperQueue, error) {
	m.EnqueueURLCalls = append(m.EnqueueURLCalls, params)
	if m.EnqueueURLErr != nil {
		return db.ScraperQueue{}, m.EnqueueURLErr
	}
	return db.ScraperQueue{Url: params.Url}, nil
}
func (m *mockQueries) WithTx(tx *sql.Tx) ScraperQueries { return m }
func (m *mockQueries) GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error) {
	return m.GetTargetResp, m.GetTargetErr
}
func (m *mockQueries) ListActiveTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	return m.ListActiveResp, m.ListActiveErr
}
func (m *mockQueries) GetQueueStats(ctx context.Context) (db.GetQueueStatsRow, error) {
	return m.GetQueueStatsResp, m.GetQueueStatsErr
}
func (m *mockQueries) DequeuePendingURL(ctx context.Context) (db.ScraperQueue, error) {
	return db.ScraperQueue{}, nil
}
func (m *mockQueries) FailQueueItem(ctx context.Context, params db.FailQueueItemParams) error {
	return nil
}
func (m *mockQueries) CompleteQueueItem(ctx context.Context, id int64) error { return nil }
func (m *mockQueries) GetPageByPath(ctx context.Context, params db.GetPageByPathParams) (db.ScraperPage, error) {
	return db.ScraperPage{}, nil
}
func (m *mockQueries) SavePage(ctx context.Context, params db.SavePageParams) (db.ScraperPage, error) {
	return db.ScraperPage{}, nil
}

// Add LogMessage to mockQueries for tests
func (m *mockQueries) LogMessage(ctx context.Context, params db.LogMessageParams) error {
	return nil // or record params for assertions if needed
}

// mockParser implements SitemapParser for testing
type mockParser struct{ URLs []mockURL }
type mockURL struct {
	Loc         string
	LastModTime *time.Time
}

func (m *mockParser) ParseSitemapForTarget(ctx context.Context, id int64) (*sitemap.ParsedSitemap, error) {
	urls := make([]sitemap.URL, len(m.URLs))
	for i, u := range m.URLs {
		urls[i] = sitemap.URL{Loc: u.Loc, LastModTime: u.LastModTime}
	}
	return &sitemap.ParsedSitemap{URLs: urls}, nil
}

// Helper to run all migration SQL files in migrationsDir into dbConn
func runMigrations(t *testing.T, dbConn *sql.DB, migrationsDir string) {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("failed to list migrations: %v", err)
	}
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", path, err)
		}
		if _, err := dbConn.Exec(string(content)); err != nil {
			t.Fatalf("failed to exec migration %s: %v", path, err)
		}
	}
}

// --- Integration test for queueing and worker logic ---
func TestScraperRunner_Integration_QueueAndProcess(t *testing.T) {
	// Use in-memory SQLite
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	runMigrations(t, dbConn, "../../scraper/db/migrations")
	// TODO: Implement integration test logic or remove this test if not needed
}

func TestScraperRunner_SkipUpdateLogic(t *testing.T) {
	// Use in-memory SQLite
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	runMigrations(t, dbConn, "../../scraper/db/migrations")
	queries := &dbQueriesAdapter{q: db.New(dbConn)}
	// Insert a target
	_, _ = dbConn.Exec(`INSERT INTO scraper_targets (id, website_url, sitemap_url, user_agent, requests_per_second) VALUES (1, 'http://test', '', 'TestAgent', 1.0)`)
	// Mock HTTP server to serve content
	mux := http.NewServeMux()
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "new page1 content"); err != nil {
			t.Fatalf("failed to write new page1 content: %v", err)
		}
	})
	content := "hello world"
	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, content); err != nil {
			t.Fatalf("failed to write content: %v", err)
		}
	})
	mux.HandleFunc("/page3", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "updated content"); err != nil {
			t.Fatalf("failed to write updated content: %v", err)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	base := server.URL
	// Insert queue items using the test server's base URL
	_, _ = dbConn.Exec(`INSERT INTO scraper_queue (id, url, target_id, priority) VALUES (1, ?, 1, 0)`, base+"/page1")
	_, _ = dbConn.Exec(`INSERT INTO scraper_queue (id, url, target_id, priority) VALUES (2, ?, 1, 0)`, base+"/page2")
	_, _ = dbConn.Exec(`INSERT INTO scraper_queue (id, url, target_id, priority) VALUES (3, ?, 1, 0)`, base+"/page3")
	// Insert a page record for page2 with same hash and up-to-date
	lastVisited := time.Now().Add(-1 * time.Hour)
	lastUpdated := time.Now().Add(-2 * time.Hour)
	hash := fmt.Sprintf("%x", xxhash.Sum64([]byte(content)))
	_, _ = dbConn.Exec(`INSERT INTO scraper_pages (target_id, url_path, full_url, html_content, content_hash, http_status_code, response_time_ms, content_length, last_visited_at, last_updated_at) VALUES (1, ?, ?, ?, ?, 200, 100, 11, ?, ?)`, base+"/page2", base+"/page2", content, hash, lastVisited, lastUpdated)
	// Insert a page record for page3 with different hash (should update)
	oldContent := "old content"
	oldHash := fmt.Sprintf("%x", xxhash.Sum64([]byte(oldContent)))
	// For page3, set lastVisitedAt to an hour before lastUpdatedAt to trigger update
	lastVisited3 := time.Now().Add(-3 * time.Hour)
	lastUpdated3 := time.Now().Add(-2 * time.Hour)
	_, _ = dbConn.Exec(`INSERT INTO scraper_pages (target_id, url_path, full_url, html_content, content_hash, http_status_code, response_time_ms, content_length, last_visited_at, last_updated_at) VALUES (1, ?, ?, ?, ?, 200, 100, 11, ?, ?)`, base+"/page3", base+"/page3", oldContent, oldHash, lastVisited3, lastUpdated3)
	// Use the test server's base URL for queue items
	sr := &ScraperRunner{
		db:          dbConn,
		queries:     queries,
		workers:     1,
		batchSize:   2,
		httpClient:  server.Client(),
		rateLimiter: NewRateLimiter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := &RunStats{TotalURLs: 3, StartTime: time.Now()}
	err = sr.processQueueWithWorkers(ctx, stats, false, false)
	if err != nil {
		t.Fatalf("processQueueWithWorkers failed: %v", err)
	}

	// Check results: page1 should be new, page2 skipped, page3 updated
	row := dbConn.QueryRow(`SELECT html_content FROM scraper_pages WHERE url_path = ?`, base+"/page1")
	var got1 string
	_ = row.Scan(&got1)
	if !strings.Contains(got1, "new page1 content") {
		t.Errorf("page1 not scraped correctly: got %q", got1)
	}
	row = dbConn.QueryRow(`SELECT html_content FROM scraper_pages WHERE url_path = ?`, base+"/page2")
	var got2 string
	_ = row.Scan(&got2)
	if got2 != content {
		t.Errorf("page2 should be skipped, got %q", got2)
	}
	row = dbConn.QueryRow(`SELECT html_content FROM scraper_pages WHERE url_path = ?`, base+"/page3")
	var got3 string
	_ = row.Scan(&got3)
	if !strings.Contains(got3, "updated content") {
		t.Errorf("page3 not updated: got %q", got3)
	}

	// Check that the correct full_url values are set
	var fullURL1, fullURL2, fullURL3 string
	row = dbConn.QueryRow(`SELECT full_url FROM scraper_pages WHERE url_path = ?`, base+"/page1")
	_ = row.Scan(&fullURL1)
	row = dbConn.QueryRow(`SELECT full_url FROM scraper_pages WHERE url_path = ?`, base+"/page2")
	_ = row.Scan(&fullURL2)
	row = dbConn.QueryRow(`SELECT full_url FROM scraper_pages WHERE url_path = ?`, base+"/page3")
	_ = row.Scan(&fullURL3)
	if fullURL1 != base+"/page1" || fullURL2 != base+"/page2" || fullURL3 != base+"/page3" {
		t.Errorf("full_url values are incorrect: %q, %q, %q", fullURL1, fullURL2, fullURL3)
	}

	// Debug output: print all rows in the pages table
	rows, _ := dbConn.Query(`SELECT url_path, full_url, html_content FROM scraper_pages`)
	for rows.Next() {
		var urlPath, fullUrl, html string
		_ = rows.Scan(&urlPath, &fullUrl, &html)
		fmt.Printf("[TEST DEBUG] DB row: url_path=%q full_url=%q html_content=%q\n", urlPath, fullUrl, html)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("failed to close rows: %v", err)
	}
}

func TestScraperRunner_ParseAndQueueURLs_HTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := fmt.Fprintln(w, `<?xml version="1.0"?><urlset><url><loc>http://a</loc></url></urlset>`); err != nil {
			t.Fatalf("failed to write sitemap XML: %v", err)
		}
	}))
	defer server.Close()
	q := &enqueueMockQueries{}
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = dbConn.Exec(`CREATE TABLE queue (id INTEGER PRIMARY KEY, url TEXT, target_id INTEGER, priority INTEGER)`)
	target := db.ScraperTarget{ID: 1, SitemapUrl: sql.NullString{String: server.URL, Valid: true}}
	parser := &mockParser{URLs: []mockURL{{Loc: "http://a"}}}
	sr := &ScraperRunner{queries: q, parser: parser, db: dbConn}
	pages, err := sr.parseAndQueueURLs(context.Background(), target, false)
	if err != nil || len(pages) != 1 {
		t.Errorf("parseAndQueueURLs HTTP failed: %v %v", pages, err)
	}
}

// In TestScraperRunner_ParseAndQueueURLs_HTTP, use a mock for EnqueueURL
type enqueueMockQueries struct {
	mockQueries
	EnqueueURLCalls []db.EnqueueURLParams
}

func (m *enqueueMockQueries) EnqueueURL(ctx context.Context, params db.EnqueueURLParams) (db.ScraperQueue, error) {
	m.EnqueueURLCalls = append(m.EnqueueURLCalls, params)
	return db.ScraperQueue{Url: params.Url}, nil
}

// Fix: enqueueMockQueries should implement WithTx to return itself as ScraperQueries
func (m *enqueueMockQueries) WithTx(tx *sql.Tx) ScraperQueries {
	return m
}

// --- Testability report ---
// Some methods (e.g., processQueueWithWorkers, scrapeURL, worker) are tightly coupled to DB, HTTP, and concurrency.
// To improve testability:
// - Use interfaces for DB and HTTP client dependencies (already partially done)
// - Allow injecting mockable worker/queue logic
// - Decouple result collection from worker logic for easier assertions
// - Consider splitting scrapeURL logic for easier unit testing

// Fix: Use sql.NullInt64{Int64: 0, Valid: false} for sql.NullInt64 fields
// Ensure dequeueMockQueries is defined at the top level
// Use = instead of := if variable is already declared
