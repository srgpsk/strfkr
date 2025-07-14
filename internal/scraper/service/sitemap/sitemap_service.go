package sitemap

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type SitemapService struct {
	client *http.Client
}

func NewSitemapService(timeout time.Duration) *SitemapService {
	return &SitemapService{
		client: &http.Client{Timeout: timeout},
	}
}

// AutoDiscoverSitemap tries common sitemap locations and returns the first valid one
func (s *SitemapService) AutoDiscoverSitemap(websiteURL string) (string, error) {
	commonPaths := []string{
		"/sitemap.xml",
		"/sitemap_index.xml",
		"/sitemap.txt",
		"/robots.txt",
	}

	parsedURL, err := url.Parse(websiteURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	for _, path := range commonPaths {
		testURL := baseURL + path
		resp, err := s.client.Get(testURL)
		if err != nil {
			continue
		}
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
		if resp.StatusCode == 200 {
			if path == "/robots.txt" {
				// Optionally parse robots.txt for sitemap references
				continue
			}
			return testURL, nil
		}
	}
	return "", fmt.Errorf("no sitemap found at common locations")
}

// ValidateSitemap checks if the sitemap at the given URL is accessible and returns its HTTP status
func (s *SitemapService) ValidateSitemap(ctx context.Context, sitemapURL, userAgent string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", sitemapURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}()
	if resp.StatusCode != 200 {
		return fmt.Errorf("sitemap returned status %d", resp.StatusCode)
	}
	return nil
}

// ParseSitemapURL fetches and parses a sitemap.xml from a URL, returning URLs for preview
func (s *SitemapService) ParseSitemapURL(ctx context.Context, sitemapURL, userAgent string) ([]URL, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", sitemapURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("sitemap returned status %d", resp.StatusCode)
	}
	var urlSet URLSet
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(body, &urlSet); err != nil {
		return nil, err
	}
	for i := range urlSet.URLs {
		if urlSet.URLs[i].LastMod != "" {
			t, err := time.Parse("2006-01-02", urlSet.URLs[i].LastMod)
			if err != nil {
				t, err = time.Parse(time.RFC3339, urlSet.URLs[i].LastMod)
			}
			if err == nil {
				urlSet.URLs[i].LastModTime = &t
			}
		}
	}
	return urlSet.URLs, nil
}
