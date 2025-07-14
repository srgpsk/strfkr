package target

import (
	"context"
	"database/sql"
	"testing"

	"app/internal/scraper/db"
)

// MockQueries for service testing
type MockQueries struct {
	createdTarget db.ScraperTarget
	createError   error
}

func (m *MockQueries) CreateTarget(ctx context.Context, params db.CreateTargetParams) (db.ScraperTarget, error) {
	if m.createError != nil {
		return db.ScraperTarget{}, m.createError
	}

	// Simulate successful creation
	target := db.ScraperTarget{
		ID:         1,
		WebsiteUrl: params.WebsiteUrl,
		SitemapUrl: params.SitemapUrl,
	}

	m.createdTarget = target
	return target, nil
}

func TestCreateTargetWithDefaults(t *testing.T) {
	mockQueries := &MockQueries{}
	service := NewTargetService(mockQueries)

	ctx := context.Background()
	websiteURL := "https://quotes.example.com"
	sitemapURL := "https://quotes.example.com/sitemap.xml"

	err := service.CreateTargetWithDefaults(ctx, websiteURL, sitemapURL)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify target was created with correct values
	if mockQueries.createdTarget.WebsiteUrl != websiteURL {
		t.Errorf("Expected WebsiteUrl %q, got %q", websiteURL, mockQueries.createdTarget.WebsiteUrl)
	}

	if !mockQueries.createdTarget.SitemapUrl.Valid || mockQueries.createdTarget.SitemapUrl.String != sitemapURL {
		t.Error("SitemapUrl not set correctly")
	}
}

func TestCreateTargetError(t *testing.T) {
	mockQueries := &MockQueries{
		createError: sql.ErrConnDone,
	}
	service := NewTargetService(mockQueries)

	ctx := context.Background()

	err := service.CreateTargetWithDefaults(ctx, "https://example.com", "https://example.com/sitemap.xml")
	if err == nil {
		t.Error("Expected error when database operation fails")
	}
}

func TestNewTargetService(t *testing.T) {
	mockQueries := &MockQueries{}
	service := NewTargetService(mockQueries)

	if service == nil {
		t.Error("NewTargetService should return a valid service")
		return
	}

	if service.queries == nil {
		t.Error("Service should have queries set")
	}
}
