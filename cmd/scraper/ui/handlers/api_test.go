package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/scraper/db"
)

type mockQueries struct {
	GetTargetCountFunc       func(context.Context) (int64, error)
	GetPendingQueueCountFunc func(context.Context) (int64, error)
	GetTotalPagesCountFunc   func(context.Context) (int64, error)
	GetRecentErrorsCountFunc func(context.Context) (int64, error)
	ListActiveTargetsFunc    func(context.Context) ([]db.ScraperTarget, error)

	// Add these fields for the rest of the API handler tests
	GetRecentLogsFunc func(context.Context, int64) ([]db.ScraperLog, error)
	LogMessageFunc    func(context.Context, db.LogMessageParams) error
}

func (m *mockQueries) GetTargetCount(ctx context.Context) (int64, error) {
	return m.GetTargetCountFunc(ctx)
}
func (m *mockQueries) GetPendingQueueCount(ctx context.Context) (int64, error) {
	return m.GetPendingQueueCountFunc(ctx)
}
func (m *mockQueries) GetTotalPagesCount(ctx context.Context) (int64, error) {
	return m.GetTotalPagesCountFunc(ctx)
}
func (m *mockQueries) GetRecentErrorsCount(ctx context.Context) (int64, error) {
	return m.GetRecentErrorsCountFunc(ctx)
}
func (m *mockQueries) ListActiveTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	return m.ListActiveTargetsFunc(ctx)
}

// Add these methods for the rest of the API handler tests
func (m *mockQueries) GetRecentLogs(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
	if m.GetRecentLogsFunc != nil {
		return m.GetRecentLogsFunc(ctx, limit)
	}
	return nil, nil
}
func (m *mockQueries) LogMessage(ctx context.Context, arg db.LogMessageParams) error {
	if m.LogMessageFunc != nil {
		return m.LogMessageFunc(ctx, arg)
	}
	return nil
}

// Satisfy db.Querier interface for tests
func (m *mockQueries) CompleteQueueItem(ctx context.Context, id int64) error {
	panic("not implemented")
}
func (m *mockQueries) CreateTarget(ctx context.Context, arg db.CreateTargetParams) (db.ScraperTarget, error) {
	panic("not implemented")
}
func (m *mockQueries) DeactivateTarget(ctx context.Context, id int64) error { panic("not implemented") }
func (m *mockQueries) DequeuePendingURL(ctx context.Context) (db.ScraperQueue, error) {
	panic("not implemented")
}
func (m *mockQueries) EnqueueURL(ctx context.Context, arg db.EnqueueURLParams) (db.ScraperQueue, error) {
	panic("not implemented")
}
func (m *mockQueries) FailQueueItem(ctx context.Context, arg db.FailQueueItemParams) error {
	panic("not implemented")
}
func (m *mockQueries) GetConfig(ctx context.Context, key string) (string, error) {
	panic("not implemented")
}
func (m *mockQueries) GetLogsByLevel(ctx context.Context, arg db.GetLogsByLevelParams) ([]db.ScraperLog, error) {
	panic("not implemented")
}
func (m *mockQueries) GetLogsByTarget(ctx context.Context, arg db.GetLogsByTargetParams) ([]db.ScraperLog, error) {
	panic("not implemented")
}
func (m *mockQueries) GetPageByPath(ctx context.Context, arg db.GetPageByPathParams) (db.ScraperPage, error) {
	panic("not implemented")
}
func (m *mockQueries) GetPageContentHash(ctx context.Context, arg db.GetPageContentHashParams) (sql.NullString, error) {
	panic("not implemented")
}
func (m *mockQueries) GetQueueStats(ctx context.Context) (db.GetQueueStatsRow, error) {
	panic("not implemented")
}
func (m *mockQueries) GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error) {
	panic("not implemented")
}
func (m *mockQueries) GetTargetByDomain(ctx context.Context, domainName sql.NullString) (db.ScraperTarget, error) {
	panic("not implemented")
}
func (m *mockQueries) GetTargetByURL(ctx context.Context, websiteUrl string) (db.ScraperTarget, error) {
	panic("not implemented")
}
func (m *mockQueries) ListAllConfig(ctx context.Context) ([]db.ScraperConfig, error) {
	panic("not implemented")
}
func (m *mockQueries) ListAllTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	panic("not implemented")
}
func (m *mockQueries) ListPagesByTarget(ctx context.Context, arg db.ListPagesByTargetParams) ([]db.ScraperPage, error) {
	panic("not implemented")
}
func (m *mockQueries) RetryFailedItem(ctx context.Context, id int64) error { panic("not implemented") }
func (m *mockQueries) SavePage(ctx context.Context, arg db.SavePageParams) (db.ScraperPage, error) {
	panic("not implemented")
}
func (m *mockQueries) SetConfig(ctx context.Context, arg db.SetConfigParams) error {
	panic("not implemented")
}
func (m *mockQueries) UpdateTargetLastVisited(ctx context.Context, id int64) error {
	panic("not implemented")
}
func (m *mockQueries) UpdateTargetPatterns(ctx context.Context, arg db.UpdateTargetPatternsParams) error {
	panic("not implemented")
}
func (m *mockQueries) GetPageClassifier(ctx context.Context, arg db.GetPageClassifierParams) (db.GetPageClassifierRow, error) {
	return db.GetPageClassifierRow{}, nil
}

// --- Add missing method to satisfy db.Querier interface ---
func (m *mockQueries) SavePageClassifier(ctx context.Context, arg db.SavePageClassifierParams) error {
	return nil
}

func TestAPIHandler_Stats(t *testing.T) {
	mock := &mockQueries{
		GetTargetCountFunc:       func(ctx context.Context) (int64, error) { return 2, nil },
		GetPendingQueueCountFunc: func(ctx context.Context) (int64, error) { return 3, nil },
		GetTotalPagesCountFunc:   func(ctx context.Context) (int64, error) { return 4, nil },
		GetRecentErrorsCountFunc: func(ctx context.Context) (int64, error) { return 1, nil },
	}
	h := &APIHandler{queries: mock}
	r := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()

	h.Stats(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %s", ct)
	}
}

func TestAPIHandler_Stats_DBError(t *testing.T) {
	mock := &mockQueries{
		GetTargetCountFunc:       func(ctx context.Context) (int64, error) { return 0, errors.New("fail") },
		GetPendingQueueCountFunc: func(ctx context.Context) (int64, error) { return 0, errors.New("fail") },
		GetTotalPagesCountFunc:   func(ctx context.Context) (int64, error) { return 0, errors.New("fail") },
		GetRecentErrorsCountFunc: func(ctx context.Context) (int64, error) { return 0, errors.New("fail") },
	}
	h := &APIHandler{queries: mock}
	r := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()

	h.Stats(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIHandler_TargetsList(t *testing.T) {
	mock := &mockQueries{
		ListActiveTargetsFunc: func(ctx context.Context) ([]db.ScraperTarget, error) {
			return []db.ScraperTarget{{ID: 1, WebsiteUrl: "https://a.com", SitemapUrl: sql.NullString{String: "https://a.com/sitemap.xml", Valid: true}, IsActive: sql.NullBool{Bool: true, Valid: true}, CreatedAt: sql.NullTime{Time: time.Now(), Valid: true}}}, nil
		},
	}
	h := &APIHandler{queries: mock}
	r := httptest.NewRequest("GET", "/api/targets", nil)
	w := httptest.NewRecorder()

	h.TargetsList(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIHandler_TargetsList_DBError(t *testing.T) {
	mock := &mockQueries{
		ListActiveTargetsFunc: func(ctx context.Context) ([]db.ScraperTarget, error) {
			return nil, errors.New("fail")
		},
	}
	h := &APIHandler{queries: mock}
	r := httptest.NewRequest("GET", "/api/targets", nil)
	w := httptest.NewRecorder()

	h.TargetsList(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIHandler_Logs(t *testing.T) {
	mock := &mockQueries{
		GetRecentLogsFunc: func(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
			return []db.ScraperLog{{LogType: "info", Message: "msg", Url: sql.NullString{String: "u", Valid: true}, Details: sql.NullString{String: "d", Valid: true}, CreatedAt: sql.NullTime{Time: time.Now(), Valid: true}}}, nil
		},
	}
	h := NewAPIHandler(mock)
	r := httptest.NewRequest("GET", "/api/logs", nil)
	w := httptest.NewRecorder()

	h.Logs(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIHandler_Logs_DBError(t *testing.T) {
	mock := &mockQueries{
		GetRecentLogsFunc: func(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
			return nil, errors.New("fail")
		},
	}
	h := NewAPIHandler(mock)
	r := httptest.NewRequest("GET", "/api/logs", nil)
	w := httptest.NewRecorder()

	h.Logs(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIHandler_StartCrawling(t *testing.T) {
	mock := &mockQueries{
		LogMessageFunc: func(ctx context.Context, arg db.LogMessageParams) error { return nil },
	}
	h := NewAPIHandler(mock)
	r := httptest.NewRequest("POST", "/api/crawl/start", nil)
	w := httptest.NewRecorder()

	h.StartCrawling(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIHandler_RefreshSitemaps(t *testing.T) {
	mock := &mockQueries{
		LogMessageFunc: func(ctx context.Context, arg db.LogMessageParams) error { return nil },
	}
	h := NewAPIHandler(mock)
	r := httptest.NewRequest("POST", "/api/sitemap/refresh-all", nil)
	w := httptest.NewRecorder()

	h.RefreshSitemaps(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
