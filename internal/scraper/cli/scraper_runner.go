package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/scraper/db"
	"app/internal/scraper/service/classifier"
	"app/internal/scraper/service/sitemap"

	"github.com/cespare/xxhash/v2"
	_ "github.com/mattn/go-sqlite3"
)

// ScraperQueries defines the subset of db.Queries methods used by ScraperRunner
// This allows for easier mocking in tests.
type ScraperQueries interface {
	GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error)
	ListActiveTargets(ctx context.Context) ([]db.ScraperTarget, error)
	GetQueueStats(ctx context.Context) (db.GetQueueStatsRow, error)
	WithTx(tx *sql.Tx) ScraperQueries // match db.Queries signature for compatibility
	DequeuePendingURL(ctx context.Context) (db.ScraperQueue, error)
	FailQueueItem(ctx context.Context, params db.FailQueueItemParams) error
	CompleteQueueItem(ctx context.Context, id int64) error
	GetPageByPath(ctx context.Context, params db.GetPageByPathParams) (db.ScraperPage, error)
	SavePage(ctx context.Context, params db.SavePageParams) (db.ScraperPage, error)
	EnqueueURL(ctx context.Context, params db.EnqueueURLParams) (db.ScraperQueue, error)
	SavePageClassifier(ctx context.Context, classifierJSON string, processable bool, targetID int64, url string) error // <-- Added missing method
}

// SitemapParser defines the interface for sitemap parsing
// This allows for easier mocking in tests.
type SitemapParser interface {
	ParseSitemapForTarget(ctx context.Context, targetID int64) (*sitemap.ParsedSitemap, error)
}

type ScraperRunner struct {
	db          *sql.DB
	queries     ScraperQueries
	parser      SitemapParser
	workers     int
	batchSize   int
	httpClient  *http.Client
	maxRetries  int
	retryDelay  time.Duration
	rateLimiter *RateLimiter
	// For testability: allows injection of batch enqueuer
	enqueueBatchFunc func(ctx context.Context, targetID int64, urls []string) (int, error)
}

type RunStats struct {
	TotalURLs int
	Processed int
	Errors    int
	Skipped   int
	StartTime time.Time
}

type ScrapedPage struct {
	URL          string
	Content      string
	ContentHash  string
	StatusCode   int
	ResponseTime time.Duration
	Error        error
}

// PageURLInfo holds URL and its lastmod time from sitemap
type PageURLInfo struct {
	URL     string
	LastMod *time.Time
}

// PageToProcess holds info for a page to be processed, including lastmod from sitemap
// Used to propagate lastmod from sitemap parser to scraper logic
type PageToProcess struct {
	TargetID int64
	URL      string
	LastMod  *time.Time
}

func NewScraperRunner(workers, batchSize int) (*ScraperRunner, error) {
	// Initialize database connection
	database, err := sql.Open("sqlite3", "data/scraper.db?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := &dbQueriesAdapter{q: db.New(database)}

	// Create sitemap parser with database access
	parser := sitemap.NewParser(queries, 30*time.Second)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &ScraperRunner{
		db:          database,
		queries:     queries, // Wrap db.Queries with dbQueriesAdapter
		parser:      parser,
		workers:     workers,
		batchSize:   batchSize,
		httpClient:  httpClient,
		maxRetries:  3,               // Default to 3 retries
		retryDelay:  2 * time.Second, // Default to 2 second delay between retries
		rateLimiter: NewRateLimiter(),
	}, nil
}

func (sr *ScraperRunner) Close() error {
	return sr.db.Close()
}

func (sr *ScraperRunner) Run(targetID int64, showProgress, verbose, dryRun bool) error {
	ctx := context.Background()
	stats := &RunStats{
		StartTime: time.Now(),
	}

	fmt.Printf("🚀 Starting scraper with %d workers, batch size %d\n", sr.workers, sr.batchSize)
	if dryRun {
		fmt.Printf("🧪 DRY RUN MODE - No actual crawling will be performed\n")
	}

	// Get targets to process
	var targets []db.ScraperTarget

	if targetID > 0 {
		// Process specific target
		target, err := sr.queries.GetTarget(ctx, targetID)
		if err != nil {
			return fmt.Errorf("failed to get target %d: %w", targetID, err)
		}
		targets = []db.ScraperTarget{target}
		fmt.Printf("📍 Processing target: %s\n", target.WebsiteUrl)
	} else {
		// Process all active targets
		allTargets, err := sr.queries.ListActiveTargets(ctx)
		if err != nil {
			return fmt.Errorf("failed to list active targets: %w", err)
		}
		targets = allTargets
		fmt.Printf("📍 Processing %d active targets\n", len(targets))
	}

	if len(targets) == 0 {
		fmt.Printf("⚠️  No targets found to process\n")
		return nil
	}

	// Phase 1: Parse sitemaps and populate queue (if targets have sitemaps)
	newURLs := 0
	for i, target := range targets {
		fmt.Printf("\n[%d/%d] Processing target: %s\n", i+1, len(targets), target.WebsiteUrl)

		// Only try to parse sitemap if target has one configured
		if target.SitemapUrl.Valid && target.SitemapUrl.String != "" {
			urls, err := sr.parseAndQueueURLs(ctx, target, dryRun)
			if err != nil {
				fmt.Printf("❌ Failed to parse sitemap for target %s: %v\n", target.WebsiteUrl, err)
				// Don't increment stats.Errors here - this is just sitemap parsing
			} else {
				newURLs += len(urls)
				fmt.Printf("✅ Queued %d new URLs from sitemap for target %s\n", len(urls), target.WebsiteUrl)
			}
		} else {
			fmt.Printf("ℹ️  No sitemap configured for target %s, will process existing queue items\n", target.WebsiteUrl)
		}
	}

	// Phase 2: Check total pending queue items
	queueStats, err := sr.queries.GetQueueStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get queue statistics: %w", err)
	}

	totalPending := int(queueStats.Pending)
	stats.TotalURLs = totalPending

	fmt.Printf("\n📊 Queue Status:\n")
	fmt.Printf("  - Pending: %d\n", queueStats.Pending)
	fmt.Printf("  - Processing: %d\n", queueStats.Processing)
	fmt.Printf("  - Completed: %d\n", queueStats.Completed)
	fmt.Printf("  - Failed: %d\n", queueStats.Failed)

	if totalPending == 0 {
		fmt.Printf("\nℹ️  No pending URLs found in queue.\n")
		if newURLs > 0 {
			fmt.Printf("📋 %d new URLs were discovered but not processed in dry-run mode.\n", newURLs)
		}
		sr.printSummary(stats)
		return nil
	}

	// Phase 3: Process URLs with workers (if not dry run)
	if !dryRun {
		fmt.Printf("\n🔄 Starting %d workers to process %d pending URLs...\n", sr.workers, totalPending)
		err := sr.processQueueWithWorkers(ctx, stats, showProgress, verbose)
		if err != nil {
			return err
		}
	} else {
		stats.Processed = totalPending // In dry run, all pending URLs are "processed"
		fmt.Printf("🧪 Dry run completed - would have processed %d URLs\n", totalPending)
	}

	// Print final summary
	sr.printSummary(stats)
	return nil
}

// parseAndQueueURLs parses sitemap and adds URLs to the queue
func (sr *ScraperRunner) parseAndQueueURLs(ctx context.Context, target db.ScraperTarget, dryRun bool) ([]PageToProcess, error) {
	// Parse sitemap to get URLs
	fmt.Printf("📄 Parsing sitemap for target %d...\n", target.ID)

	result, err := sr.parser.ParseSitemapForTarget(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sitemap: %w", err)
	}

	if len(result.URLs) == 0 {
		fmt.Printf("⚠️  No URLs found in sitemap\n")
		return nil, nil
	}

	fmt.Printf("📊 Found %d URLs in sitemap\n", len(result.URLs))

	pages := make([]PageToProcess, len(result.URLs))
	for i, url := range result.URLs {
		pages[i] = PageToProcess{
			TargetID: target.ID,
			URL:      url.Loc,
			LastMod:  url.LastModTime,
		}
	}

	if dryRun {
		return pages, nil
	}

	// Add URLs to queue using batch processing (unchanged)
	urls := make([]string, len(result.URLs))
	for i, url := range result.URLs {
		urls[i] = url.Loc
	}
	batchSize := 50
	queuedCount, err := sr.BatchEnqueueURLs(ctx, target.ID, urls, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to batch enqueue URLs: %w", err)
	}

	fmt.Printf("📊 Successfully queued %d out of %d URLs\n", queuedCount, len(urls))
	return pages[:queuedCount], nil
}

// BatchEnqueueURLs adds multiple URLs to the queue in batches for better performance
func (sr *ScraperRunner) BatchEnqueueURLs(ctx context.Context, targetID int64, urls []string, batchSize int) (int, error) {
	totalQueued := 0

	for i := 0; i < len(urls); i += batchSize {
		end := i + batchSize
		if end > len(urls) {
			end = len(urls)
		}

		batch := urls[i:end]
		var queued int
		var err error
		if sr.enqueueBatchFunc != nil {
			queued, err = sr.enqueueBatchFunc(ctx, targetID, batch)
		} else {
			queued, err = sr.enqueueBatch(ctx, targetID, batch)
		}
		if err != nil {
			return totalQueued, fmt.Errorf("failed to enqueue batch starting at index %d: %w", i, err)
		}

		totalQueued += queued
	}

	return totalQueued, nil
}

// enqueueBatch adds a batch of URLs to the queue
func (sr *ScraperRunner) enqueueBatch(ctx context.Context, targetID int64, urls []string) (int, error) {
	// Start transaction for batch insert
	tx, err := sr.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // ignore error, as per Go best practices for Rollback in defer
	}()

	qtx := sr.queries.WithTx(tx)
	queued := 0

	for _, url := range urls {
		_, err := qtx.EnqueueURL(ctx, db.EnqueueURLParams{
			TargetID: targetID,
			Url:      url,
			Priority: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			// Log error but continue with other URLs
			fmt.Printf("⚠️  Failed to queue URL %s: %v\n", url, err)
			continue
		}
		queued++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return queued, nil
}

// processQueueWithWorkers starts worker goroutines to process URLs from the queue
func (sr *ScraperRunner) processQueueWithWorkers(ctx context.Context, stats *RunStats, showProgress, verbose bool) error {
	// Phase 1: Build lastModMap from all sitemaps for all targets
	lastModMap := make(map[string]*time.Time)
	targets, err := sr.queries.ListActiveTargets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active targets: %w", err)
	}
	for _, target := range targets {
		if target.SitemapUrl.Valid && target.SitemapUrl.String != "" {
			ctx2 := context.Background()
			result, err := sr.parser.ParseSitemapForTarget(ctx2, target.ID)
			if err == nil {
				for _, url := range result.URLs {
					lastModMap[url.Loc] = url.LastModTime
				}
			}
		}
	}

	// Initialize progress reporter
	var reporter *ProgressReporter
	if showProgress {
		reporter = NewProgressReporter(stats.TotalURLs, verbose)
	}

	// Create channels for worker communication
	resultChan := make(chan ScrapedPage, sr.batchSize)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < sr.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				queueItem, err := sr.queries.DequeuePendingURL(ctx)
				if err != nil {
					if err == sql.ErrNoRows {
						break
					}
					resultChan <- ScrapedPage{Error: fmt.Errorf("failed to dequeue URL: %w", err)}
					continue
				}
				lastMod := lastModMap[queueItem.Url]
				pageToProcess := PageToProcess{
					TargetID: queueItem.TargetID,
					URL:      queueItem.Url,
					LastMod:  lastMod,
				}
				page := sr.scrapeURLAttempt(ctx, pageToProcess, lastMod)
				resultChan <- page

				// --- Quote Page Classifier Integration ---
				classifier := classifier.NewQuotePageClassifierService()
				classifierResult, err := classifier.ClassifyPage(pageToProcess.URL, page.Content) // <-- Use page.Content instead of page.HtmlContent
				if err == nil && classifierResult != nil {
					jsonStr, _ := json.Marshal(classifierResult)
					_ = sr.queries.SavePageClassifier(ctx, string(jsonStr), classifierResult.Decision.Processable, queueItem.TargetID, queueItem.Url)
				}
				// --- End Integration ---

				if page.Error != nil {
					if err := sr.queries.FailQueueItem(ctx, db.FailQueueItemParams{
						ID:           queueItem.ID,
						ErrorMessage: sql.NullString{String: page.Error.Error(), Valid: true},
					}); err != nil {
						fmt.Printf("failed to mark queue item as failed: %v\n", err)
					}
				} else {
					if err := sr.queries.CompleteQueueItem(ctx, queueItem.ID); err != nil {
						fmt.Printf("failed to mark queue item as complete: %v\n", err)
					}
				}
			}
		}()
	}

	// Start result collector
	done := make(chan bool)
	go sr.resultCollector(resultChan, stats, reporter, verbose, done)

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Wait for result collector to finish
	<-done

	if reporter != nil {
		reporter.Finish()
	}

	return nil
}

// scrapeURLAttempt performs a single scraping attempt
func (sr *ScraperRunner) scrapeURLAttempt(ctx context.Context, pageToProcess PageToProcess, lastMod *time.Time) ScrapedPage {
	startTime := time.Now()
	page := ScrapedPage{
		URL: pageToProcess.URL,
	}

	// Get target details for user agent
	target, err := sr.queries.GetTarget(ctx, pageToProcess.TargetID)
	if err != nil {
		page.Error = fmt.Errorf("failed to get target: %w", err)
		return page
	}

	// Set user agent
	userAgent := target.UserAgent.String
	if userAgent == "" {
		userAgent = "ScraperBot/1.0"
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", pageToProcess.URL, nil)
	if err != nil {
		page.Error = fmt.Errorf("failed to create request: %w", err)
		return page
	}
	req.Header.Set("User-Agent", userAgent)

	// Rate limiting
	rate := 1.0
	if target.RequestsPerSecond.Valid && target.RequestsPerSecond.Float64 > 0 {
		rate = target.RequestsPerSecond.Float64
	}
	sr.rateLimiter.WaitN(ctx, pageToProcess.TargetID, rate)

	// Make HTTP request
	resp, err := sr.httpClient.Do(req)
	if err != nil {
		page.Error = fmt.Errorf("HTTP request failed: %w", err)
		return page
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}()

	page.StatusCode = resp.StatusCode
	page.ResponseTime = time.Since(startTime)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		page.Error = fmt.Errorf("failed to read response body: %w", err)
		return page
	}

	page.Content = string(body)

	// Use xxhash for content hash
	hash := xxhash.Sum64(body)
	page.ContentHash = fmt.Sprintf("%x", hash)

	// Fetch page record from DB
	fmt.Printf("[DEBUG] Checking for existing page record: TargetID=%d, UrlPath=%s\n", pageToProcess.TargetID, pageToProcess.URL)
	pageRecord, err := sr.queries.GetPageByPath(ctx, db.GetPageByPathParams{
		TargetID: pageToProcess.TargetID,
		UrlPath:  pageToProcess.URL,
	})
	fmt.Printf("[DEBUG] GetPageByPath err: %v, pageRecord: %+v\n", err, pageRecord)
	var lastVisitedAt, lastUpdatedAt time.Time
	var storedHash string
	if err == nil {
		lastVisitedAt = pageRecord.LastVisitedAt.Time
		lastUpdatedAt = pageRecord.LastUpdatedAt.Time
		storedHash = pageRecord.ContentHash.String
		fmt.Printf("[DEBUG] lastVisitedAt: %v, lastUpdatedAt: %v, storedHash: %s\n", lastVisitedAt, lastUpdatedAt, storedHash)
	}

	// If lastVisitedAt > lastUpdatedAt, skip
	if !lastVisitedAt.IsZero() && !lastUpdatedAt.IsZero() && lastVisitedAt.After(lastUpdatedAt) {
		fmt.Printf("[DEBUG] Skipping: lastVisitedAt > lastUpdatedAt\n")
		page.Error = fmt.Errorf("skipped: already processed after last update")
		return page
	}

	// If lastVisitedAt < lastUpdatedAt, check hash
	if !lastVisitedAt.IsZero() && !lastUpdatedAt.IsZero() && lastVisitedAt.Before(lastUpdatedAt) {
		fmt.Printf("[DEBUG] lastVisitedAt < lastUpdatedAt, storedHash: %s, newHash: %s\n", storedHash, page.ContentHash)
		if storedHash == page.ContentHash {
			fmt.Printf("[DEBUG] Skipping: hash matches, no update needed\n")
			page.Error = fmt.Errorf("skipped: hash matches, no update needed")
			return page
		}
	}

	fmt.Printf("[DEBUG] Saving page: TargetID=%d, UrlPath=%s\n", pageToProcess.TargetID, pageToProcess.URL)
	// Save page with last_updated_at from sitemap if available
	_, err = sr.queries.SavePage(ctx, db.SavePageParams{
		TargetID:       pageToProcess.TargetID,
		UrlPath:        pageToProcess.URL, // adjust if needed
		FullUrl:        pageToProcess.URL,
		HtmlContent:    sql.NullString{String: page.Content, Valid: true},
		ContentHash:    sql.NullString{String: page.ContentHash, Valid: true},
		HttpStatusCode: sql.NullInt64{Int64: int64(page.StatusCode), Valid: true},
		ResponseTimeMs: sql.NullInt64{Int64: page.ResponseTime.Milliseconds(), Valid: true},
		ContentLength:  sql.NullInt64{Int64: int64(len(page.Content)), Valid: true},
		LastUpdatedAt:  sql.NullTime{Time: lastModOrNow(lastMod), Valid: true},
	})
	if err != nil {
		fmt.Printf("[DEBUG] Failed to save page: %v\n", err)
		page.Error = fmt.Errorf("failed to save page: %w", err)
		return page
	}

	fmt.Printf("[DEBUG] Page saved successfully: %s\n", pageToProcess.URL)
	return page
}

// lastModOrNow returns lastmod if not nil, otherwise now
func lastModOrNow(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Now()
}

// resultCollector processes results from workers and updates statistics
func (sr *ScraperRunner) resultCollector(resultChan <-chan ScrapedPage, stats *RunStats, reporter *ProgressReporter, verbose bool, done chan<- bool) {
	defer func() { done <- true }()

	for page := range resultChan {
		if page.Error != nil {
			stats.Errors++
			if reporter != nil {
				// Classify error type
				errorType := sr.classifyError(page.Error)
				reporter.RecordError(errorType)

				if verbose {
					reporter.LogError(fmt.Sprintf("Error processing %s: %v", page.URL, page.Error))
				}
			}
		} else {
			stats.Processed++
			if reporter != nil && verbose {
				reporter.LogSuccess(fmt.Sprintf("✅ Scraped %s (%d bytes, %v)",
					page.URL, len(page.Content), page.ResponseTime.Round(time.Millisecond)))
			}
		}

		if reporter != nil {
			reporter.UpdateProgress(stats.Processed, stats.Errors, stats.Skipped)
		}
	}
}

func (sr *ScraperRunner) printSummary(stats *RunStats) {
	elapsed := time.Since(stats.StartTime)
	separator := strings.Repeat("=", 50)
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("📊 SCRAPING SUMMARY\n")
	fmt.Printf("%s\n", separator)
	fmt.Printf("⏱️  Total time: %s\n", elapsed.Round(time.Second))
	fmt.Printf("🔢 Total URLs: %d\n", stats.TotalURLs)
	fmt.Printf("✅ Processed: %d\n", stats.Processed)
	fmt.Printf("❌ Errors: %d\n", stats.Errors)
	fmt.Printf("⏭️  Skipped: %d\n", stats.Skipped)

	if stats.Processed > 0 {
		rate := float64(stats.Processed) / elapsed.Seconds()
		fmt.Printf("⚡ Average rate: %.2f URLs/second\n", rate)
	}

	successRate := float64(stats.Processed) / float64(stats.TotalURLs) * 100
	fmt.Printf("📈 Success rate: %.1f%%\n", successRate)
	fmt.Printf("%s\n", separator)
}

// SetRetryConfig configures retry behavior
func (sr *ScraperRunner) SetRetryConfig(maxRetries int, retryDelay time.Duration) {
	sr.maxRetries = maxRetries
	sr.retryDelay = retryDelay
}

// GetRetryConfig returns current retry configuration
func (sr *ScraperRunner) GetRetryConfig() (int, time.Duration) {
	return sr.maxRetries, sr.retryDelay
}

// classifyError categorizes errors for better reporting
func (sr *ScraperRunner) classifyError(err error) string {
	if err == nil {
		return "unknown"
	}

	errorStr := strings.ToLower(err.Error())

	// Network errors
	if strings.Contains(errorStr, "timeout") || strings.Contains(errorStr, "context deadline exceeded") {
		return "timeout"
	}
	if strings.Contains(errorStr, "connection refused") || strings.Contains(errorStr, "connection reset") {
		return "connection_error"
	}
	if strings.Contains(errorStr, "dns") {
		return "dns_error"
	}
	if strings.Contains(errorStr, "network") {
		return "network_error"
	}

	// HTTP errors
	if strings.Contains(errorStr, "HTTP 4") {
		return "client_error"
	}
	if strings.Contains(errorStr, "HTTP 5") {
		return "server_error"
	}
	if strings.Contains(errorStr, "HTTP 429") {
		return "rate_limited"
	}

	// Database errors
	if strings.Contains(errorStr, "database") || strings.Contains(errorStr, "sql") {
		return "database_error"
	}

	// Parse errors
	if strings.Contains(errorStr, "parse") || strings.Contains(errorStr, "decode") {
		return "parse_error"
	}

	return "other"
}

// RateLimiter manages per-target rate limiting
type RateLimiter struct {
	limits map[int64]time.Time
	mu     sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: make(map[int64]time.Time),
	}
}

func (rl *RateLimiter) Wait(targetID int64, delay time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if lastRequest, exists := rl.limits[targetID]; exists {
		elapsed := time.Since(lastRequest)
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
	}

	rl.limits[targetID] = time.Now()
}

func (rl *RateLimiter) WaitN(ctx context.Context, targetID int64, rate float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Calculate delay based on rate
	delay := time.Duration(float64(time.Second) / rate)

	if lastRequest, exists := rl.limits[targetID]; exists {
		elapsed := time.Since(lastRequest)
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
	}

	rl.limits[targetID] = time.Now()
}

// Helper for sql.NullInt64: returns a valid or invalid value based on input
// func nullInt64(val interface{}) sql.NullInt64 {
// 	if val == nil {
// 		return sql.NullInt64{Valid: false}
// 	}
// 	switch v := val.(type) {
// 	case int64:
// 		return sql.NullInt64{Int64: v, Valid: true}
// 	case *int64:
// 		if v == nil {
// 			return sql.NullInt64{Valid: false}
// 		}
// 		return sql.NullInt64{Int64: *v, Valid: true}
// 	case int:
// 		return sql.NullInt64{Int64: int64(v), Valid: true}
// 	case *int:
// 		if v == nil {
// 			return sql.NullInt64{Valid: false}
// 		}
// 		return sql.NullInt64{Int64: int64(*v), Valid: true}
// 	}
// 	return sql.NullInt64{Valid: false}
// }

// Helper for sql.NullString: returns a valid or invalid value based on input
// func nullString(val interface{}) sql.NullString {
// 	if val == nil {
// 		return sql.NullString{Valid: false}
// 	}
// 	switch v := val.(type) {
// 	case string:
// 		return sql.NullString{String: v, Valid: true}
// 	case *string:
// 		if v == nil {
// 			return sql.NullString{Valid: false}
// 		}
// 		return sql.NullString{String: *v, Valid: true}
// 	}
// 	return sql.NullString{Valid: false}
// }

// Helper for sql.NullBool: returns a valid or invalid value based on input
// func nullBool(val *bool) sql.NullBool {
// 	if val == nil {
// 		return sql.NullBool{Valid: false}
// 	}
// 	return sql.NullBool{Bool: *val, Valid: true}
// }

// --- Fix for TestScraperRunner_SkipUpdateLogic ---
// Use the test server's base URL for all queue and page records
// This should be done in the test file, not here, but document for clarity
// --- Fix for TestScraperRunner_ParseAndQueueURLs_HTTP ---
// Use a mockQueries with a working EnqueueURL method in the test file

// func extractURLs(pages []sitemap.URL) []string {
// 	urls := make([]string, len(pages))
// 	for i, page := range pages {
// 		urls[i] = page.Loc
// 	}
// 	return urls
// }

// func (sr *ScraperRunner) worker(ctx context.Context, resultChan chan<- ScrapedPage, wg *sync.WaitGroup, reporter *ProgressReporter, lastModMap map[string]*time.Time) {
// 	// Worker logic here
// }

// func (sr *ScraperRunner) scrapeURL(ctx context.Context, pageToProcess PageToProcess) ScrapedPage {
// 	// Scraping logic here
// }

// func (sr *ScraperRunner) isRetryableError(err error) bool {
// 	// Retryable error logic here
// }

// dbQueriesAdapter wraps *db.Queries to implement ScraperQueries with the correct WithTx signature
// This allows production code to use the interface, and tests to use mocks

type dbQueriesAdapter struct {
	q *db.Queries
}

func (a *dbQueriesAdapter) GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error) {
	return a.q.GetTarget(ctx, id)
}
func (a *dbQueriesAdapter) ListActiveTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	return a.q.ListActiveTargets(ctx)
}
func (a *dbQueriesAdapter) GetQueueStats(ctx context.Context) (db.GetQueueStatsRow, error) {
	return a.q.GetQueueStats(ctx)
}
func (a *dbQueriesAdapter) WithTx(tx *sql.Tx) ScraperQueries {
	return &dbQueriesAdapter{q: a.q.WithTx(tx)}
}

// Add missing LogMessage method to satisfy logger.LoggerQueries
func (a *dbQueriesAdapter) LogMessage(ctx context.Context, params db.LogMessageParams) error {
	return a.q.LogMessage(ctx, params)
}
func (a *dbQueriesAdapter) DequeuePendingURL(ctx context.Context) (db.ScraperQueue, error) {
	return a.q.DequeuePendingURL(ctx)
}
func (a *dbQueriesAdapter) FailQueueItem(ctx context.Context, params db.FailQueueItemParams) error {
	return a.q.FailQueueItem(ctx, params)
}
func (a *dbQueriesAdapter) CompleteQueueItem(ctx context.Context, id int64) error {
	return a.q.CompleteQueueItem(ctx, id)
}
func (a *dbQueriesAdapter) GetPageByPath(ctx context.Context, params db.GetPageByPathParams) (db.ScraperPage, error) {
	return a.q.GetPageByPath(ctx, params)
}
func (a *dbQueriesAdapter) SavePage(ctx context.Context, params db.SavePageParams) (db.ScraperPage, error) {
	return a.q.SavePage(ctx, params)
}
func (a *dbQueriesAdapter) EnqueueURL(ctx context.Context, params db.EnqueueURLParams) (db.ScraperQueue, error) {
	return a.q.EnqueueURL(ctx, params)
}
func (a *dbQueriesAdapter) SavePageClassifier(ctx context.Context, classifierJSON string, processable bool, targetID int64, url string) error {
	params := db.SavePageClassifierParams{
		QuoteClassifierJson: sql.NullString{String: classifierJSON, Valid: true},
		Processable:         sql.NullBool{Bool: processable, Valid: true},
		TargetID:            targetID,
		UrlPath:             url,
	}
	return a.q.SavePageClassifier(ctx, params)
}
