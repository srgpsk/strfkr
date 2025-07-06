package cli

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"app/internal/scraper/db"
	"app/internal/scraper/sitemap"

	_ "github.com/mattn/go-sqlite3"
)

type ScraperRunner struct {
	db          *sql.DB
	queries     *db.Queries
	parser      *sitemap.Parser
	workers     int
	batchSize   int
	httpClient  *http.Client
	maxRetries  int
	retryDelay  time.Duration
	rateLimiter *RateLimiter
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

type QueueItem struct {
	ID       int64
	TargetID int64
	URL      string
	Priority int64
}

func NewScraperRunner(workers, batchSize int) (*ScraperRunner, error) {
	// Initialize database connection
	database, err := sql.Open("sqlite3", "data/scraper.db?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := db.New(database)

	// Create sitemap parser with database access
	parser := sitemap.NewParser(queries, 30*time.Second)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &ScraperRunner{
		db:          database,
		queries:     queries,
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
func (sr *ScraperRunner) parseAndQueueURLs(ctx context.Context, target db.ScraperTarget, dryRun bool) ([]string, error) {
	// Parse sitemap to get URLs
	fmt.Printf("📄 Parsing sitemap for target %d...\n", target.ID)

	result, err := sr.parser.ParseSitemapForTarget(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sitemap: %w", err)
	}

	if len(result.URLs) == 0 {
		fmt.Printf("⚠️  No URLs found in sitemap\n")
		return []string{}, nil
	}

	fmt.Printf("📊 Found %d URLs in sitemap\n", len(result.URLs))

	if dryRun {
		// In dry run, just return the URLs without queuing
		urls := make([]string, len(result.URLs))
		for i, url := range result.URLs {
			urls[i] = url.Loc
		}
		return urls, nil
	}

	// Add URLs to queue using batch processing
	urls := make([]string, len(result.URLs))
	for i, url := range result.URLs {
		urls[i] = url.Loc
	}

	// Use batch processing for better performance
	batchSize := 50 // Process 50 URLs at a time
	queuedCount, err := sr.BatchEnqueueURLs(ctx, target.ID, urls, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to batch enqueue URLs: %w", err)
	}

	fmt.Printf("📊 Successfully queued %d out of %d URLs\n", queuedCount, len(urls))
	return urls[:queuedCount], nil
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
		queued, err := sr.enqueueBatch(ctx, targetID, batch)
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
		go sr.worker(ctx, resultChan, &wg, reporter)
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

// worker processes URLs from the queue
func (sr *ScraperRunner) worker(ctx context.Context, resultChan chan<- ScrapedPage, wg *sync.WaitGroup, reporter *ProgressReporter) {
	defer wg.Done()

	for {
		// Get next URL from queue
		queueItem, err := sr.queries.DequeuePendingURL(ctx)
		if err != nil {
			if err == sql.ErrNoRows {
				// No more URLs to process
				break
			}
			// Log error and continue
			resultChan <- ScrapedPage{Error: fmt.Errorf("failed to dequeue URL: %w", err)}
			continue
		}

		// Scrape the URL
		page := sr.scrapeURL(ctx, queueItem)
		resultChan <- page

		// Update queue status
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
}

// scrapeURL fetches and processes a single URL with retry logic
func (sr *ScraperRunner) scrapeURL(ctx context.Context, queueItem db.ScraperQueue) ScrapedPage {
	var page ScrapedPage
	var lastError error

	for attempt := 0; attempt <= sr.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			time.Sleep(sr.retryDelay * time.Duration(attempt))
		}

		page = sr.scrapeURLAttempt(ctx, queueItem)

		// If successful, break out of retry loop
		if page.Error == nil {
			break
		}

		lastError = page.Error

		// Check if error is retryable
		if !sr.isRetryableError(page.Error) {
			break
		}

		// Log retry attempt
		if attempt < sr.maxRetries {
			fmt.Printf("⚠️  Retrying %s (attempt %d/%d): %v\n", queueItem.Url, attempt+1, sr.maxRetries, page.Error)
			// Note: We'll need to pass the reporter here to track retries
		}
	}

	// If we exhausted retries, use the last error
	if page.Error != nil {
		page.Error = fmt.Errorf("failed after %d retries: %w", sr.maxRetries, lastError)
	}

	return page
}

// scrapeURLAttempt performs a single scraping attempt
func (sr *ScraperRunner) scrapeURLAttempt(ctx context.Context, queueItem db.ScraperQueue) ScrapedPage {
	startTime := time.Now()

	page := ScrapedPage{
		URL: queueItem.Url,
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", queueItem.Url, nil)
	if err != nil {
		page.Error = fmt.Errorf("failed to create request: %w", err)
		return page
	}

	// Get target details for user agent
	target, err := sr.queries.GetTarget(ctx, queueItem.TargetID)
	if err != nil {
		page.Error = fmt.Errorf("failed to get target: %w", err)
		return page
	}

	// Set user agent
	userAgent := target.UserAgent.String
	if userAgent == "" {
		userAgent = "ScraperBot/1.0"
	}
	req.Header.Set("User-Agent", userAgent)

	// Rate limiting
	// per target
	rate := 1.0
	if target.RequestsPerSecond.Valid && target.RequestsPerSecond.Float64 > 0 {
		rate = target.RequestsPerSecond.Float64
	}
	sr.rateLimiter.WaitN(ctx, queueItem.TargetID, rate)

	// if target.CrawlDelaySeconds.Valid && target.CrawlDelaySeconds.Int64 > 0 {
	// 	sr.rateLimiter.Wait(target.ID, time.Duration(target.CrawlDelaySeconds.Int64)*time.Second)
	// }

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

	// Calculate content hash
	hash := sha256.Sum256(body)
	page.ContentHash = fmt.Sprintf("%x", hash)

	// Save page to database
	err = sr.savePage(ctx, queueItem.TargetID, page)
	if err != nil {
		page.Error = fmt.Errorf("failed to save page: %w", err)
		return page
	}

	return page
}

// savePage stores the scraped page in the database
func (sr *ScraperRunner) savePage(ctx context.Context, targetID int64, page ScrapedPage) error {
	// Extract URL path for database storage
	parsedURL, err := url.Parse(page.URL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	urlPath := parsedURL.Path
	if urlPath == "" {
		urlPath = "/"
	}

	_, err = sr.queries.SavePage(ctx, db.SavePageParams{
		TargetID:       targetID,
		UrlPath:        urlPath,
		FullUrl:        page.URL,
		HtmlContent:    sql.NullString{String: page.Content, Valid: true},
		ContentHash:    sql.NullString{String: page.ContentHash, Valid: true},
		HttpStatusCode: sql.NullInt64{Int64: int64(page.StatusCode), Valid: true},
		ResponseTimeMs: sql.NullInt64{Int64: page.ResponseTime.Milliseconds(), Valid: true},
		ContentLength:  sql.NullInt64{Int64: int64(len(page.Content)), Valid: true},
	})

	return err
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

// isRetryableError determines if an error is worth retrying
func (sr *ScraperRunner) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())

	// Retry network-related errors
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"network is unreachable",
		"temporary failure",
		"dns lookup failed",
		"context deadline exceeded",
		"i/o timeout",
		"connection reset by peer",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errorStr, retryable) {
			return true
		}
	}

	// Retry certain HTTP status codes
	if strings.Contains(errorStr, "HTTP 5") || // 5xx server errors
		strings.Contains(errorStr, "HTTP 429") || // Rate limiting
		strings.Contains(errorStr, "HTTP 408") { // Request timeout
		return true
	}

	return false
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
