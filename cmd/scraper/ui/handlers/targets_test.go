package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/scraper/db"
)

type mockTargetsQueries struct {
	CreateTargetFunc     func(context.Context, db.CreateTargetParams) (db.ScraperTarget, error)
	DeactivateTargetFunc func(context.Context, int64) error
}

func (m *mockTargetsQueries) CreateTarget(ctx context.Context, arg db.CreateTargetParams) (db.ScraperTarget, error) {
	if m.CreateTargetFunc != nil {
		return m.CreateTargetFunc(ctx, arg)
	}
	return db.ScraperTarget{}, nil
}
func (m *mockTargetsQueries) DeactivateTarget(ctx context.Context, id int64) error {
	if m.DeactivateTargetFunc != nil {
		return m.DeactivateTargetFunc(ctx, id)
	}
	return nil
}

// Satisfy db.Querier interface for tests
func (m *mockTargetsQueries) ListActiveTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	return nil, nil
}
func (m *mockTargetsQueries) GetRecentLogs(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
	return nil, nil
}
func (m *mockTargetsQueries) CompleteQueueItem(ctx context.Context, id int64) error { return nil }
func (m *mockTargetsQueries) DequeuePendingURL(ctx context.Context) (db.ScraperQueue, error) {
	return db.ScraperQueue{}, nil
}
func (m *mockTargetsQueries) EnqueueURL(ctx context.Context, arg db.EnqueueURLParams) (db.ScraperQueue, error) {
	return db.ScraperQueue{}, nil
}
func (m *mockTargetsQueries) FailQueueItem(ctx context.Context, arg db.FailQueueItemParams) error {
	return nil
}
func (m *mockTargetsQueries) GetConfig(ctx context.Context, key string) (string, error) {
	return "", nil
}
func (m *mockTargetsQueries) GetLogsByLevel(ctx context.Context, arg db.GetLogsByLevelParams) ([]db.ScraperLog, error) {
	return nil, nil
}
func (m *mockTargetsQueries) GetLogsByTarget(ctx context.Context, arg db.GetLogsByTargetParams) ([]db.ScraperLog, error) {
	return nil, nil
}
func (m *mockTargetsQueries) GetPageByPath(ctx context.Context, arg db.GetPageByPathParams) (db.ScraperPage, error) {
	return db.ScraperPage{}, nil
}
func (m *mockTargetsQueries) GetPageContentHash(ctx context.Context, arg db.GetPageContentHashParams) (sql.NullString, error) {
	return sql.NullString{}, nil
}
func (m *mockTargetsQueries) GetQueueStats(ctx context.Context) (db.GetQueueStatsRow, error) {
	return db.GetQueueStatsRow{}, nil
}
func (m *mockTargetsQueries) GetRecentErrorsCount(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockTargetsQueries) GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
}
func (m *mockTargetsQueries) GetTargetByDomain(ctx context.Context, domainName sql.NullString) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
}
func (m *mockTargetsQueries) GetTargetByURL(ctx context.Context, websiteUrl string) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
}
func (m *mockTargetsQueries) GetTargetCount(ctx context.Context) (int64, error)     { return 0, nil }
func (m *mockTargetsQueries) GetTotalPagesCount(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockTargetsQueries) ListAllConfig(ctx context.Context) ([]db.ScraperConfig, error) {
	return nil, nil
}
func (m *mockTargetsQueries) ListAllTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	return nil, nil
}
func (m *mockTargetsQueries) ListPagesByTarget(ctx context.Context, arg db.ListPagesByTargetParams) ([]db.ScraperPage, error) {
	return nil, nil
}
func (m *mockTargetsQueries) LogMessage(ctx context.Context, arg db.LogMessageParams) error {
	return nil
}
func (m *mockTargetsQueries) RetryFailedItem(ctx context.Context, id int64) error { return nil }
func (m *mockTargetsQueries) SavePage(ctx context.Context, arg db.SavePageParams) (db.ScraperPage, error) {
	return db.ScraperPage{}, nil
}
func (m *mockTargetsQueries) SetConfig(ctx context.Context, arg db.SetConfigParams) error { return nil }
func (m *mockTargetsQueries) UpdateTargetLastVisited(ctx context.Context, id int64) error { return nil }
func (m *mockTargetsQueries) UpdateTargetPatterns(ctx context.Context, arg db.UpdateTargetPatternsParams) error {
	return nil
}
func (m *mockTargetsQueries) GetPendingQueueCount(ctx context.Context) (int64, error) { return 0, nil }

func TestTargetsHandler_NewForm(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{}}
	r := httptest.NewRequest("GET", "/targets/new", nil)
	w := httptest.NewRecorder()

	h.NewForm(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTargetsHandler_Create_Success(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{
		CreateTargetFunc: func(ctx context.Context, arg db.CreateTargetParams) (db.ScraperTarget, error) {
			return db.ScraperTarget{ID: 1, WebsiteUrl: arg.WebsiteUrl}, nil
		},
	}}
	form := "website_url=https://a.com&sitemap_url=https://a.com/sitemap.xml"
	r := httptest.NewRequest("POST", "/targets", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Create(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTargetsHandler_Create_MissingWebsiteURL(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{}}
	form := "sitemap_url=https://a.com/sitemap.xml"
	r := httptest.NewRequest("POST", "/targets", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Create(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTargetsHandler_Create_DBError(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{
		CreateTargetFunc: func(ctx context.Context, arg db.CreateTargetParams) (db.ScraperTarget, error) {
			return db.ScraperTarget{}, errors.New("db error")
		},
	}}
	form := "website_url=https://a.com&sitemap_url=https://a.com/sitemap.xml"
	r := httptest.NewRequest("POST", "/targets", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Create(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTargetsHandler_Delete_Success(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{
		DeactivateTargetFunc: func(ctx context.Context, id int64) error {
			return nil
		},
	}}
	r := httptest.NewRequest("DELETE", "/targets/1", nil)
	r.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.Delete(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTargetsHandler_Delete_BadID(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{}}
	r := httptest.NewRequest("DELETE", "/targets/abc", nil)
	r.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	h.Delete(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestTargetsHandler_Delete_DBError(t *testing.T) {
	h := &TargetsHandler{queries: &mockTargetsQueries{
		DeactivateTargetFunc: func(ctx context.Context, id int64) error {
			return errors.New("db error")
		},
	}}
	r := httptest.NewRequest("DELETE", "/targets/1", nil)
	r.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.Delete(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
