package cli

import (
	"fmt"
	"sync"
	"time"
)

type ProgressReporter struct {
	mu         sync.Mutex
	startTime  time.Time
	totalURLs  int
	processed  int
	errors     int
	skipped    int
	retries    int
	verbose    bool
	lastUpdate time.Time
	errorTypes map[string]int
}

func NewProgressReporter(totalURLs int, verbose bool) *ProgressReporter {
	return &ProgressReporter{
		startTime:  time.Now(),
		totalURLs:  totalURLs,
		verbose:    verbose,
		lastUpdate: time.Now(),
		errorTypes: make(map[string]int),
	}
}

func (pr *ProgressReporter) UpdateProgress(processed, errors, skipped int) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(pr.startTime)

	// Calculate instantaneous rate (last 10 seconds)
	instantRate := float64(processed-pr.processed) / now.Sub(pr.lastUpdate).Seconds()
	if pr.lastUpdate.IsZero() {
		instantRate = 0
	}

	pr.processed = processed
	pr.errors = errors
	pr.skipped = skipped
	pr.lastUpdate = now

	// Show both average and instantaneous rates
	avgRate := float64(processed) / elapsed.Seconds()

	fmt.Printf("\r🔄 Progress: %d/%d (%.1f%%) | Errors: %d | Skipped: %d | Avg: %.1f/s | Current: %.1f/s | Elapsed: %s",
		processed, pr.totalURLs,
		float64(processed)/float64(pr.totalURLs)*100,
		errors, skipped, avgRate, instantRate, elapsed.Round(time.Second))
}

func (pr *ProgressReporter) UpdateWithMessage(processed, errors, skipped int, message string) {
	pr.UpdateProgress(processed, errors, skipped)
	if pr.verbose && message != "" {
		fmt.Printf("\n  %s", message)
	}
}

func (pr *ProgressReporter) Finish() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	elapsed := time.Since(pr.startTime)
	fmt.Printf("\n✅ Completed: %d processed, %d errors, %d skipped, %d retries in %s\n",
		pr.processed, pr.errors, pr.skipped, pr.retries, elapsed.Round(time.Second))

	if pr.processed > 0 {
		avgRate := float64(pr.processed) / elapsed.Seconds()
		fmt.Printf("📊 Average rate: %.2f URLs/second\n", avgRate)
	}

	// Show error breakdown if there were errors
	if pr.errors > 0 && len(pr.errorTypes) > 0 {
		fmt.Printf("🔍 Error breakdown:\n")
		for errorType, count := range pr.errorTypes {
			fmt.Printf("  - %s: %d\n", errorType, count)
		}
	}
}

func (pr *ProgressReporter) LogError(message string) {
	if pr.verbose {
		fmt.Printf("\n❌ Error: %s", message)
	}
}

func (pr *ProgressReporter) LogInfo(message string) {
	if pr.verbose {
		fmt.Printf("\n📝 %s", message)
	}
}

func (pr *ProgressReporter) LogSuccess(message string) {
	if pr.verbose {
		fmt.Printf("\n%s", message)
	}
}

func (pr *ProgressReporter) IncrementRetries() {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.retries++
}

func (pr *ProgressReporter) RecordError(errorType string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.errorTypes[errorType]++
}

func (pr *ProgressReporter) GetErrorBreakdown() map[string]int {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	breakdown := make(map[string]int)
	for k, v := range pr.errorTypes {
		breakdown[k] = v
	}
	return breakdown
}
