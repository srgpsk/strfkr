package sitemap

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"app/internal/scraper/config"
	"app/internal/scraper/db"
	"app/internal/scraper/logger"
)

// ParserQueries defines the interface needed for sitemap parsing
type ParserQueries interface {
	GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error)
}

// defaultIfEmpty returns defaultSlice if slice is empty
func defaultIfEmpty(slice []string, defaultSlice []string) []string {
	if len(slice) == 0 {
		return defaultSlice
	}
	return slice
}

// SitemapIndex represents the root sitemap.xml structure
type SitemapIndex struct {
	XMLName  xml.Name  `xml:"sitemapindex"`
	Sitemaps []Sitemap `xml:"sitemap"`
}

// URLSet represents individual sitemap files with URLs
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

// Sitemap represents a reference to a sitemap file
type Sitemap struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// URL represents an individual URL entry
type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

// ParsedSitemap contains the final results
type ParsedSitemap struct {
	URLs        []URL
	SubSitemaps []Sitemap
	LastMod     time.Time
}

// Parser handles sitemap parsing with database-driven configuration
type Parser struct {
	client  *http.Client
	queries ParserQueries
	logger  *logger.DBLogger
}

// NewParser creates a new sitemap parser with database access
func NewParser(queries ParserQueries, timeout time.Duration) *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: timeout,
		},
		queries: queries,
		logger:  logger.NewDBLogger(queries.(logger.LoggerQueries)), // Type assertion for logger
	}
}

// ParseSitemapForTarget parses sitemap using target-specific patterns from database
func (p *Parser) ParseSitemapForTarget(ctx context.Context, targetID int64) (*ParsedSitemap, error) {
	p.logger.Info(ctx, &targetID, "", fmt.Sprintf("Starting sitemap parsing for target %d", targetID))

	// Get target configuration from database
	target, err := p.queries.GetTarget(ctx, targetID)
	if err != nil {
		p.logger.Error(ctx, &targetID, "", "Failed to get target from database", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get target: %w", err)
	}

	// Get sitemap URL
	sitemapURL := ""
	if target.SitemapUrl.Valid {
		sitemapURL = target.SitemapUrl.String
	} else {
		p.logger.Error(ctx, &targetID, "", "Target has no sitemap URL configured")
		return nil, fmt.Errorf("target has no sitemap URL configured")
	}

	p.logger.Info(ctx, &targetID, sitemapURL, "Starting sitemap parsing", map[string]interface{}{
		"website_url": target.WebsiteUrl,
	})

	// Parse pattern configuration
	var sitemapPatterns, urlPatterns []string

	if target.SitemapPatterns.Valid && target.SitemapPatterns.String != "" {
		var patterns []string
		if err := json.Unmarshal([]byte(target.SitemapPatterns.String), &patterns); err == nil {
			sitemapPatterns = patterns
		} else {
			p.logger.Warn(ctx, &targetID, "", "Failed to parse sitemap patterns", map[string]interface{}{
				"error":    err.Error(),
				"patterns": target.SitemapPatterns.String,
			})
		}
	}

	if target.UrlPatterns.Valid && target.UrlPatterns.String != "" {
		var patterns []string
		if err := json.Unmarshal([]byte(target.UrlPatterns.String), &patterns); err == nil {
			urlPatterns = patterns
		} else {
			p.logger.Warn(ctx, &targetID, "", "Failed to parse URL patterns", map[string]interface{}{
				"error":    err.Error(),
				"patterns": target.UrlPatterns.String,
			})
		}
	}

	// Use defaults if patterns are empty
	defaults := config.DefaultPatterns()
	sitemapPatterns = defaultIfEmpty(sitemapPatterns, defaults.SitemapPatterns)
	urlPatterns = defaultIfEmpty(urlPatterns, defaults.URLPatterns)

	// Compile patterns using config package
	compiledSitemapPatterns, err := config.CompilePatterns(sitemapPatterns)
	if err != nil {
		p.logger.Error(ctx, &targetID, sitemapURL, "Invalid sitemap patterns", map[string]interface{}{
			"error":    err.Error(),
			"patterns": sitemapPatterns,
		})
		return nil, fmt.Errorf("invalid sitemap patterns: %w", err)
	}

	compiledURLPatterns, err := config.CompilePatterns(urlPatterns)
	if err != nil {
		p.logger.Error(ctx, &targetID, sitemapURL, "Invalid URL patterns", map[string]interface{}{
			"error":    err.Error(),
			"patterns": urlPatterns,
		})
		return nil, fmt.Errorf("invalid URL patterns: %w", err)
	}

	// Parse sitemap with target-specific configuration
	userAgent := target.UserAgent.String
	if userAgent == "" {
		userAgent = "QuotesBot/1.0"
	}

	result, err := p.parseSitemapWithPatterns(ctx, targetID, sitemapURL, userAgent, compiledSitemapPatterns, compiledURLPatterns)
	if err != nil {
		p.logger.Error(ctx, &targetID, sitemapURL, "Sitemap parsing failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	p.logger.Info(ctx, &targetID, sitemapURL, "Sitemap parsing completed", map[string]interface{}{
		"url_count":     len(result.URLs),
		"sitemap_count": len(result.SubSitemaps),
	})

	return result, nil
}

// parseSitemapWithPatterns does the actual parsing with compiled patterns
func (p *Parser) parseSitemapWithPatterns(ctx context.Context, targetID int64, sitemapURL, userAgent string,
	sitemapPatterns, urlPatterns []*regexp.Regexp) (*ParsedSitemap, error) {

	// First, try to fetch and parse as sitemap index
	sitemapIndex, err := p.fetchSitemapIndex(ctx, targetID, sitemapURL, userAgent)
	if err == nil && len(sitemapIndex.Sitemaps) > 0 {
		// This is a sitemap index, process sub-sitemaps
		return p.processSitemapIndex(ctx, targetID, sitemapIndex, userAgent, sitemapPatterns, urlPatterns)
	}

	// Not a sitemap index, try parsing as regular sitemap
	urlSet, err := p.fetchURLSet(ctx, targetID, sitemapURL, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sitemap %s: %w", sitemapURL, err)
	}

	result := &ParsedSitemap{
		URLs: filterURLs(urlSet.URLs, urlPatterns),
	}

	return result, nil
}

// fetchSitemapIndex fetches and parses a sitemap index
func (p *Parser) fetchSitemapIndex(ctx context.Context, targetID int64, url, userAgent string) (*SitemapIndex, error) {
	resp, err := p.fetchURL(ctx, targetID, url, userAgent)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn(ctx, &targetID, url, "Failed to close response body", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	var sitemapIndex SitemapIndex
	if err := xml.NewDecoder(resp.Body).Decode(&sitemapIndex); err != nil {
		p.logger.Error(ctx, &targetID, url, "Failed to decode sitemap index", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to decode sitemap index: %w", err)
	}

	p.logger.Info(ctx, &targetID, url, "Successfully parsed sitemap index", map[string]interface{}{
		"sitemap_count": len(sitemapIndex.Sitemaps),
	})

	return &sitemapIndex, nil
}

// fetchURLSet fetches and parses a regular sitemap
func (p *Parser) fetchURLSet(ctx context.Context, targetID int64, url, userAgent string) (*URLSet, error) {
	resp, err := p.fetchURL(ctx, targetID, url, userAgent)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn(ctx, &targetID, url, "Failed to close response body", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	var urlSet URLSet
	if err := xml.NewDecoder(resp.Body).Decode(&urlSet); err != nil {
		p.logger.Error(ctx, &targetID, url, "Failed to decode URL set", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to decode URL set: %w", err)
	}

	p.logger.Info(ctx, &targetID, url, "Successfully parsed sitemap", map[string]interface{}{
		"url_count": len(urlSet.URLs),
	})

	return &urlSet, nil
}

// fetchURL performs HTTP request with proper headers
func (p *Parser) fetchURL(ctx context.Context, targetID int64, url, userAgent string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/xml, text/xml, */*")

	p.logger.Info(ctx, &targetID, url, "Fetching sitemap", map[string]interface{}{
		"user_agent": userAgent,
	})

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error(ctx, &targetID, url, "HTTP request failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		p.logger.Error(ctx, &targetID, url, "HTTP request returned error status", map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
		})
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn(ctx, &targetID, url, "Failed to close response body after error", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// processSitemapIndex processes a sitemap index and fetches relevant sub-sitemaps
func (p *Parser) processSitemapIndex(ctx context.Context, targetID int64, index *SitemapIndex, userAgent string,
	sitemapPatterns, urlPatterns []*regexp.Regexp) (*ParsedSitemap, error) {
	result := &ParsedSitemap{}

	// Filter relevant sub-sitemaps
	relevantSitemaps := filterSitemaps(index.Sitemaps, sitemapPatterns)

	p.logger.Info(ctx, &targetID, "", "Processing sitemap index", map[string]interface{}{
		"total_sitemaps":    len(index.Sitemaps),
		"relevant_sitemaps": len(relevantSitemaps),
	})

	for _, sitemap := range relevantSitemaps {
		// Fetch each relevant sub-sitemap
		urlSet, err := p.fetchURLSet(ctx, targetID, sitemap.Loc, userAgent)
		if err != nil {
			p.logger.Warn(ctx, &targetID, sitemap.Loc, "Failed to fetch sub-sitemap, continuing with others", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Add filtered URLs to result
		filteredURLs := filterURLs(urlSet.URLs, urlPatterns)
		result.URLs = append(result.URLs, filteredURLs...)

		p.logger.Info(ctx, &targetID, sitemap.Loc, "Processed sub-sitemap", map[string]interface{}{
			"total_urls":    len(urlSet.URLs),
			"filtered_urls": len(filteredURLs),
		})
	}

	result.SubSitemaps = relevantSitemaps
	return result, nil
}

// filterSitemaps filters sub-sitemaps based on patterns
func filterSitemaps(sitemaps []Sitemap, patterns []*regexp.Regexp) []Sitemap {
	if len(patterns) == 0 {
		return sitemaps
	}

	var filtered []Sitemap
	for _, sitemap := range sitemaps {
		for _, pattern := range patterns {
			if pattern.MatchString(sitemap.Loc) {
				filtered = append(filtered, sitemap)
				break
			}
		}
	}
	return filtered
}

// filterURLs filters URLs based on patterns
func filterURLs(urls []URL, patterns []*regexp.Regexp) []URL {
	if len(patterns) == 0 {
		return urls
	}

	var filtered []URL
	for _, url := range urls {
		for _, pattern := range patterns {
			if pattern.MatchString(url.Loc) {
				filtered = append(filtered, url)
				break
			}
		}
	}
	return filtered
}
