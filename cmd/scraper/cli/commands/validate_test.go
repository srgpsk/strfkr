package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/scraper/service"
	"context"
)

func TestParseSitemapURL(t *testing.T) {
	svc := service.NewSitemapService(2 * time.Second)
	ctx := context.Background()
	t.Run("valid sitemap with lastmod", func(t *testing.T) {
		sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/page1</loc>
    <lastmod>2024-01-01</lastmod>
  </url>
  <url>
    <loc>https://example.com/page2</loc>
  </url>
</urlset>`
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, sitemapXML); err != nil {
				t.Fatalf("failed to write sitemapXML: %v", err)
			}
		}))
		defer ts.Close()

		urls, err := svc.ParseSitemapURL(ctx, ts.URL, "TestBot/1.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 2 {
			t.Errorf("expected 2 URLs, got %d", len(urls))
		}
		if urls[0].Loc != "https://example.com/page1" || urls[1].Loc != "https://example.com/page2" {
			t.Errorf("unexpected URLs: %+v", urls)
		}
		if urls[0].LastModTime == nil {
			t.Errorf("expected LastModTime to be set for first URL")
		}
		if urls[1].LastModTime != nil {
			t.Errorf("expected LastModTime to be nil for second URL")
		}
	})

	t.Run("invalid XML", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, "not xml"); err != nil {
				t.Fatalf("failed to write not xml: %v", err)
			}
		}))
		defer ts.Close()
		_, err := svc.ParseSitemapURL(ctx, ts.URL, "TestBot/1.0")
		if err == nil {
			t.Errorf("expected error for invalid XML, got nil")
		}
	})

	t.Run("non-200 response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer ts.Close()
		_, err := svc.ParseSitemapURL(ctx, ts.URL, "TestBot/1.0")
		if err == nil || !strings.Contains(err.Error(), "status 404") {
			t.Errorf("expected HTTP 404 error, got %v", err)
		}
	})
}
