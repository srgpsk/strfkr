package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockRoundTripper struct {
	fn func(req *http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req), nil
}

func newMockClient(fn func(req *http.Request) *http.Response) *http.Client {
	return &http.Client{
		Transport: &mockRoundTripper{fn: fn},
	}
}

func TestAutoDiscoverSitemap(t *testing.T) {
	calls := make(map[string]bool)
	client := newMockClient(func(req *http.Request) *http.Response {
		calls[req.URL.Path] = true
		if req.URL.Path == "/sitemap.xml" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}
	})
	service := &SitemapService{client: client}
	url, err := service.AutoDiscoverSitemap("https://example.com/foo/bar")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasSuffix(url, "/sitemap.xml") {
		t.Errorf("expected /sitemap.xml, got %s", url)
	}
	if !calls["/sitemap.xml"] {
		t.Error("should have called /sitemap.xml")
	}
}

func TestAutoDiscoverSitemap_InvalidURL(t *testing.T) {
	service := &SitemapService{client: newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}
	})}
	_, err := service.AutoDiscoverSitemap(":bad-url")
	if err == nil || !strings.Contains(err.Error(), "invalid URL") {
		t.Error("expected invalid URL error")
	}
}

func TestAutoDiscoverSitemap_AllErrors(t *testing.T) {
	client := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("fail"))}
	})
	service := &SitemapService{client: client}
	_, err := service.AutoDiscoverSitemap("https://example.com")
	if err == nil || !strings.Contains(err.Error(), "no sitemap found") {
		t.Error("expected no sitemap found error")
	}
}

func TestAutoDiscoverSitemap_RobotsTxtOnly(t *testing.T) {
	client := newMockClient(func(req *http.Request) *http.Response {
		if req.URL.Path == "/robots.txt" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("robots"))}
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}
	})
	service := &SitemapService{client: client}
	_, err := service.AutoDiscoverSitemap("https://example.com")
	if err == nil || !strings.Contains(err.Error(), "no sitemap found") {
		t.Error("expected no sitemap found error when only robots.txt is present")
	}
}

func TestValidateSitemap(t *testing.T) {
	client := newMockClient(func(req *http.Request) *http.Response {
		if req.URL.String() == "https://good.com/sitemap.xml" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	if err := service.ValidateSitemap(ctx, "https://good.com/sitemap.xml", ""); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := service.ValidateSitemap(ctx, "https://bad.com/sitemap.xml", ""); err == nil {
		t.Error("expected error for 404, got nil")
	}
}

func TestValidateSitemap_InvalidURL(t *testing.T) {
	service := &SitemapService{client: http.DefaultClient}
	ctx := context.Background()
	err := service.ValidateSitemap(ctx, ":bad-url", "")
	if err == nil || !strings.Contains(err.Error(), "failed to create request") {
		t.Error("expected failed to create request error")
	}
}

func TestValidateSitemap_ClientError(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{fn: func(req *http.Request) *http.Response {
			return nil // Simulate network error
		}},
	}
	service := &SitemapService{client: client}
	ctx := context.Background()
	err := service.ValidateSitemap(ctx, "https://fail.com/sitemap.xml", "")
	if err == nil || !strings.Contains(err.Error(), "failed to fetch sitemap") {
		t.Error("expected failed to fetch sitemap error")
	}
}

func TestValidateSitemap_UserAgent(t *testing.T) {
	var gotUA string
	client := newMockClient(func(req *http.Request) *http.Response {
		gotUA = req.Header.Get("User-Agent")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	_ = service.ValidateSitemap(ctx, "https://good.com/sitemap.xml", "my-agent")
	if gotUA != "my-agent" {
		t.Errorf("expected User-Agent 'my-agent', got '%s'", gotUA)
	}
}

func TestParseSitemapURL(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
    <lastmod>2023-01-01</lastmod>
  </url>
  <url>
    <loc>https://example.com/about</loc>
    <lastmod>2023-01-02T15:04:05Z</lastmod>
  </url>
</urlset>`
	client := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sitemapXML))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	urls, err := service.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 {
		t.Errorf("expected 2 urls, got %d", len(urls))
	}
	if urls[0].Loc != "https://example.com/" {
		t.Errorf("unexpected loc: %s", urls[0].Loc)
	}
	if urls[0].LastModTime == nil || urls[1].LastModTime == nil {
		t.Error("expected LastModTime to be parsed")
	}
}

func TestParseSitemapURL_Errors(t *testing.T) {
	client := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("fail"))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	_, err := service.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "")
	if err == nil {
		t.Error("expected error for 500 status")
	}

	client2 := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte("not xml")))}
	})
	service2 := &SitemapService{client: client2}
	_, err = service2.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "")
	if err == nil {
		t.Error("expected error for invalid xml")
	}
}

func TestParseSitemapURL_EmptySitemap(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`
	client := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sitemapXML))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	urls, err := service.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 urls, got %d", len(urls))
	}
}

func TestParseSitemapURL_MissingLastMod(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/</loc></url></urlset>`
	client := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sitemapXML))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	urls, err := service.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 1 {
		t.Errorf("expected 1 url, got %d", len(urls))
	}
	if urls[0].LastModTime != nil {
		t.Error("expected LastModTime to be nil when missing")
	}
}

func TestParseSitemapURL_BadLastModFormat(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/</loc><lastmod>notadate</lastmod></url></urlset>`
	client := newMockClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(sitemapXML))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	urls, err := service.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 1 {
		t.Errorf("expected 1 url, got %d", len(urls))
	}
	if urls[0].LastModTime != nil {
		t.Error("expected LastModTime to be nil for bad format")
	}
}

func TestParseSitemapURL_UserAgent(t *testing.T) {
	var gotUA string
	client := newMockClient(func(req *http.Request) *http.Response {
		gotUA = req.Header.Get("User-Agent")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("<urlset></urlset>"))}
	})
	service := &SitemapService{client: client}
	ctx := context.Background()
	_, _ = service.ParseSitemapURL(ctx, "https://example.com/sitemap.xml", "test-agent")
	if gotUA != "test-agent" {
		t.Errorf("expected User-Agent 'test-agent', got '%s'", gotUA)
	}
}
