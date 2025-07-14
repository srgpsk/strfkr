package sitemap

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"app/internal/scraper/config"
	"app/internal/scraper/db"
)

// MockQueries for testing parser - implements both interfaces
type MockQueries struct {
	target         db.ScraperTarget
	getTargetError error
	lastLogMessage db.LogMessageParams
	logError       error
}

// Implement db interface for GetTarget
func (m *MockQueries) GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error) {
	return m.target, m.getTargetError
}

// Implement logger interface for LogMessage
func (m *MockQueries) LogMessage(ctx context.Context, params db.LogMessageParams) error {
	if m.logError != nil {
		return m.logError
	}
	m.lastLogMessage = params
	return nil
}

func TestDefaultIfEmpty(t *testing.T) {
	tests := []struct {
		name         string
		slice        []string
		defaultSlice []string
		expected     []string
	}{
		{
			name:         "empty slice returns default",
			slice:        []string{},
			defaultSlice: []string{"default1", "default2"},
			expected:     []string{"default1", "default2"},
		},
		{
			name:         "nil slice returns default",
			slice:        nil,
			defaultSlice: []string{"default1", "default2"},
			expected:     []string{"default1", "default2"},
		},
		{
			name:         "non-empty slice returns original",
			slice:        []string{"original1", "original2"},
			defaultSlice: []string{"default1", "default2"},
			expected:     []string{"original1", "original2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultIfEmpty(tt.slice, tt.defaultSlice)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Expected %q at index %d, got %q", tt.expected[i], i, v)
				}
			}
		})
	}
}

func TestFilterSitemaps(t *testing.T) {
	sitemaps := []Sitemap{
		{Loc: "https://example.com/sitemap-posts.xml"},
		{Loc: "https://example.com/sitemap-pages.xml"},
		{Loc: "https://example.com/sitemap-categories.xml"},
		{Loc: "https://example.com/other-file.xml"},
	}

	pattern, _ := regexp.Compile(`sitemap-posts\.xml|sitemap-pages\.xml`)

	filtered := filterSitemaps(sitemaps, []*regexp.Regexp{pattern})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered sitemaps, got %d", len(filtered))
	}

	// Test with no patterns (should return all)
	allFiltered := filterSitemaps(sitemaps, []*regexp.Regexp{})
	if len(allFiltered) != len(sitemaps) {
		t.Error("With no patterns, should return all sitemaps")
	}
}

func TestFilterURLs(t *testing.T) {
	urls := []URL{
		{Loc: "https://example.com/post/title-1/"},
		{Loc: "https://example.com/post/title-2/"},
		{Loc: "https://example.com/page/about/"},
		{Loc: "https://example.com/admin/login"},
	}

	pattern, _ := regexp.Compile(`/post/[^/]+/$`)

	filtered := filterURLs(urls, []*regexp.Regexp{pattern})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered URLs, got %d", len(filtered))
	}

	// Test with no patterns (should return all)
	allFiltered := filterURLs(urls, []*regexp.Regexp{})
	if len(allFiltered) != len(urls) {
		t.Error("With no patterns, should return all URLs")
	}
}

func TestNewParser(t *testing.T) {
	mockQueries := &MockQueries{}
	parser := NewParser(mockQueries, 30*time.Second)

	if parser == nil {
		t.Error("NewParser should return a valid parser")
		return
	}

	if parser.client == nil {
		t.Error("Parser should have a valid HTTP client")
		return
	}

	if parser.client.Timeout != 30*time.Second {
		t.Error("Parser should use the specified timeout")
	}
}

func TestParseError_NoSitemapURL(t *testing.T) {
	mockQueries := &MockQueries{
		target: db.ScraperTarget{
			ID:         1,
			WebsiteUrl: "https://example.com",
			SitemapUrl: sql.NullString{Valid: false}, // No sitemap URL
		},
	}

	parser := NewParser(mockQueries, 10*time.Second)
	ctx := context.Background()

	_, err := parser.ParseSitemapForTarget(ctx, 1)
	if err == nil {
		t.Error("Expected error when no sitemap URL is configured")
	}

	if !strings.Contains(err.Error(), "no sitemap URL configured") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestParseError_DatabaseError(t *testing.T) {
	mockQueries := &MockQueries{
		getTargetError: sql.ErrConnDone,
	}

	parser := NewParser(mockQueries, 10*time.Second)
	ctx := context.Background()

	_, err := parser.ParseSitemapForTarget(ctx, 1)
	if err == nil {
		t.Error("Expected error when database query fails")
	}
}

func TestParseError_InvalidPatterns(t *testing.T) {
	mockQueries := &MockQueries{
		target: db.ScraperTarget{
			ID:              1,
			WebsiteUrl:      "https://example.com",
			SitemapUrl:      sql.NullString{String: "https://example.com/sitemap.xml", Valid: true},
			SitemapPatterns: sql.NullString{String: `["[invalid regex"]`, Valid: true}, // Invalid regex
		},
	}

	parser := NewParser(mockQueries, 10*time.Second)
	ctx := context.Background()

	_, err := parser.ParseSitemapForTarget(ctx, 1)
	if err == nil {
		t.Error("Expected error when sitemap patterns are invalid")
	}

	if !strings.Contains(err.Error(), "invalid sitemap patterns") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestParseWithHTTPServer(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/post/test-quote/</loc>
    <lastmod>2024-01-01</lastmod>
  </url>
  <url>
    <loc>https://example.com/admin/login</loc>
    <lastmod>2024-01-01</lastmod>
  </url>
</urlset>`))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Setup mock with valid configuration
	patterns := config.DefaultPatterns()
	patternsJSON, _ := patterns.ToJSON()

	mockQueries := &MockQueries{
		target: db.ScraperTarget{
			ID:              1,
			WebsiteUrl:      "https://example.com",
			SitemapUrl:      sql.NullString{String: server.URL + "/sitemap.xml", Valid: true},
			SitemapPatterns: sql.NullString{String: patternsJSON, Valid: true},
			UserAgent:       sql.NullString{String: "TestBot/1.0", Valid: true},
		},
	}

	parser := NewParser(mockQueries, 10*time.Second)
	ctx := context.Background()

	result, err := parser.ParseSitemapForTarget(ctx, 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected valid result")
		return
	}

	// Should have some URLs
	if len(result.URLs) == 0 {
		t.Error("Expected at least one URL in result")
	}
}

func TestHTTPError(t *testing.T) {
	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockQueries := &MockQueries{
		target: db.ScraperTarget{
			ID:         1,
			WebsiteUrl: "https://example.com",
			SitemapUrl: sql.NullString{String: server.URL + "/sitemap.xml", Valid: true},
		},
	}

	parser := NewParser(mockQueries, 10*time.Second)
	ctx := context.Background()

	_, err := parser.ParseSitemapForTarget(ctx, 1)
	if err == nil {
		t.Error("Expected error when HTTP request fails")
	}
}
