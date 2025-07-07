package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/scraper/db"
)

type mockDashboardQueries struct {
	ListActiveTargetsFunc func(context.Context) ([]db.ScraperTarget, error)
	GetRecentLogsFunc     func(context.Context, int64) ([]db.ScraperLog, error)
}

func (m *mockDashboardQueries) ListActiveTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	if m.ListActiveTargetsFunc != nil {
		return m.ListActiveTargetsFunc(ctx)
	}
	return nil, nil
}
func (m *mockDashboardQueries) GetRecentLogs(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
	if m.GetRecentLogsFunc != nil {
		return m.GetRecentLogsFunc(ctx, limit)
	}
	return nil, nil
}

// Satisfy db.Querier interface for tests
func (m *mockDashboardQueries) CompleteQueueItem(ctx context.Context, id int64) error { return nil } // unused in dashboard tests
func (m *mockDashboardQueries) CreateTarget(ctx context.Context, arg db.CreateTargetParams) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
}                                                                                    // unused
func (m *mockDashboardQueries) DeactivateTarget(ctx context.Context, id int64) error { return nil } // unused
func (m *mockDashboardQueries) DequeuePendingURL(ctx context.Context) (db.ScraperQueue, error) {
	return db.ScraperQueue{}, nil
} // unused
func (m *mockDashboardQueries) EnqueueURL(ctx context.Context, arg db.EnqueueURLParams) (db.ScraperQueue, error) {
	return db.ScraperQueue{}, nil
} // unused
func (m *mockDashboardQueries) FailQueueItem(ctx context.Context, arg db.FailQueueItemParams) error {
	return nil
} // unused
func (m *mockDashboardQueries) GetConfig(ctx context.Context, key string) (string, error) {
	return "", nil
} // unused
func (m *mockDashboardQueries) GetLogsByLevel(ctx context.Context, arg db.GetLogsByLevelParams) ([]db.ScraperLog, error) {
	return nil, nil
} // unused
func (m *mockDashboardQueries) GetLogsByTarget(ctx context.Context, arg db.GetLogsByTargetParams) ([]db.ScraperLog, error) {
	return nil, nil
} // unused
func (m *mockDashboardQueries) GetPageByPath(ctx context.Context, arg db.GetPageByPathParams) (db.ScraperPage, error) {
	return db.ScraperPage{}, nil
} // unused
func (m *mockDashboardQueries) GetPageContentHash(ctx context.Context, arg db.GetPageContentHashParams) (sql.NullString, error) {
	return sql.NullString{}, nil
} // unused
func (m *mockDashboardQueries) GetQueueStats(ctx context.Context) (db.GetQueueStatsRow, error) {
	return db.GetQueueStatsRow{}, nil
} // unused
func (m *mockDashboardQueries) GetRecentErrorsCount(ctx context.Context) (int64, error) {
	return 0, nil
} // unused
func (m *mockDashboardQueries) GetTarget(ctx context.Context, id int64) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
} // unused
func (m *mockDashboardQueries) GetTargetByDomain(ctx context.Context, domainName sql.NullString) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
} // unused
func (m *mockDashboardQueries) GetTargetByURL(ctx context.Context, websiteUrl string) (db.ScraperTarget, error) {
	return db.ScraperTarget{}, nil
}                                                                                     // unused
func (m *mockDashboardQueries) GetTargetCount(ctx context.Context) (int64, error)     { return 0, nil } // unused
func (m *mockDashboardQueries) GetTotalPagesCount(ctx context.Context) (int64, error) { return 0, nil } // unused
func (m *mockDashboardQueries) ListAllConfig(ctx context.Context) ([]db.ScraperConfig, error) {
	return nil, nil
} // unused
func (m *mockDashboardQueries) ListAllTargets(ctx context.Context) ([]db.ScraperTarget, error) {
	return nil, nil
} // unused
func (m *mockDashboardQueries) ListPagesByTarget(ctx context.Context, arg db.ListPagesByTargetParams) ([]db.ScraperPage, error) {
	return nil, nil
} // unused
func (m *mockDashboardQueries) LogMessage(ctx context.Context, arg db.LogMessageParams) error {
	return nil
}                                                                                   // unused
func (m *mockDashboardQueries) RetryFailedItem(ctx context.Context, id int64) error { return nil } // unused
func (m *mockDashboardQueries) SavePage(ctx context.Context, arg db.SavePageParams) (db.ScraperPage, error) {
	return db.ScraperPage{}, nil
} // unused
func (m *mockDashboardQueries) SetConfig(ctx context.Context, arg db.SetConfigParams) error {
	return nil
} // unused
func (m *mockDashboardQueries) UpdateTargetLastVisited(ctx context.Context, id int64) error {
	return nil
} // unused
func (m *mockDashboardQueries) UpdateTargetPatterns(ctx context.Context, arg db.UpdateTargetPatternsParams) error {
	return nil
} // unused
func (m *mockDashboardQueries) GetPendingQueueCount(ctx context.Context) (int64, error) {
	return 0, nil
} // unused

func TestDashboardHandler_Dashboard(t *testing.T) {
	h := &DashboardHandler{queries: &mockDashboardQueries{}}
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.Dashboard(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDashboardHandler_HealthAPI(t *testing.T) {
	h := &DashboardHandler{queries: &mockDashboardQueries{}}
	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.HealthAPI(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDashboardHandler_TargetsPage(t *testing.T) {
	h := &DashboardHandler{queries: &mockDashboardQueries{
		ListActiveTargetsFunc: func(ctx context.Context) ([]db.ScraperTarget, error) {
			return []db.ScraperTarget{{ID: 1, WebsiteUrl: "https://a.com", SitemapUrl: sql.NullString{String: "https://a.com/sitemap.xml", Valid: true}, IsActive: sql.NullBool{Bool: true, Valid: true}, CreatedAt: sql.NullTime{Time: time.Now(), Valid: true}}}, nil
		},
	}}
	r := httptest.NewRequest("GET", "/targets", nil)
	w := httptest.NewRecorder()

	h.TargetsPage(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDashboardHandler_TargetsPage_DBError(t *testing.T) {
	h := &DashboardHandler{queries: &mockDashboardQueries{
		ListActiveTargetsFunc: func(ctx context.Context) ([]db.ScraperTarget, error) {
			return nil, sql.ErrConnDone
		},
	}}
	r := httptest.NewRequest("GET", "/targets", nil)
	w := httptest.NewRecorder()

	h.TargetsPage(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestDashboardHandler_LogsPage(t *testing.T) {
	h := &DashboardHandler{queries: &mockDashboardQueries{
		GetRecentLogsFunc: func(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
			return []db.ScraperLog{{LogType: "info", Message: "msg", Url: sql.NullString{String: "u", Valid: true}, Details: sql.NullString{String: "d", Valid: true}, CreatedAt: sql.NullTime{Time: time.Now(), Valid: true}}}, nil
		},
	}}
	r := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()

	h.LogsPage(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDashboardHandler_LogsPage_DBError(t *testing.T) {
	h := &DashboardHandler{queries: &mockDashboardQueries{
		GetRecentLogsFunc: func(ctx context.Context, limit int64) ([]db.ScraperLog, error) {
			return nil, sql.ErrConnDone
		},
	}}
	r := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()

	h.LogsPage(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
