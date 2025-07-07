package handlers

import (
	"net/http"
	"time"

	"app/cmd/scraper/ui/models"
	"app/cmd/scraper/ui/templates/pages"
	"app/internal/scraper/db"
)

type DashboardHandler struct {
	queries db.Querier
}

func NewDashboardHandler(queries db.Querier) *DashboardHandler {
	return &DashboardHandler{queries: queries}
}

// Dashboard handles the main dashboard page
func (h *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	component := pages.Dashboard()
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HealthPage handles the health check page
func (h *DashboardHandler) HealthPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	data := models.HealthData{
		Status:    "ok",
		Service:   "scraper",
		Timestamp: time.Now(),
		Uptime:    "24h",
	}
	component := pages.Health(data)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render health page", http.StatusInternalServerError)
		return
	}
}

// TargetsPage handles the targets management page
func (h *DashboardHandler) TargetsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	dbTargets, err := h.queries.ListActiveTargets(r.Context())
	if err != nil {
		http.Error(w, "Failed to load targets: "+err.Error(), http.StatusInternalServerError)
		return
	}
	targets := make([]models.TargetData, 0, len(dbTargets))
	for _, t := range dbTargets {
		targets = append(targets, models.TargetData{
			ID:         t.ID,
			WebsiteURL: t.WebsiteUrl,
			SitemapURL: t.SitemapUrl.String,
			Status:     "active", // Only active targets listed
			CreatedAt:  t.CreatedAt.Time,
		})
	}
	component := pages.Targets(targets)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render targets page", http.StatusInternalServerError)
		return
	}
}

// LogsPage handles the logs page
func (h *DashboardHandler) LogsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	dbLogs, err := h.queries.GetRecentLogs(r.Context(), 50)
	if err != nil {
		http.Error(w, "Failed to load logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	logs := make([]models.LogEntry, 0, len(dbLogs))
	for _, l := range dbLogs {
		logs = append(logs, models.LogEntry{
			Timestamp: l.CreatedAt.Time,
			Level:     l.LogType,
			Message:   l.Message,
			URL:       l.Url.String,
			Details:   l.Details.String,
		})
	}
	component := pages.Logs(logs)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render logs page", http.StatusInternalServerError)
		return
	}
}

// HealthAPI handles health check API endpoint
func (h *DashboardHandler) HealthAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"ok","service":"scraper","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Health handles the health check with detailed data (API version)
func (h *DashboardHandler) Health(w http.ResponseWriter, r *http.Request) {
	data := models.HealthData{
		Status:    "ok",
		Service:   "scraper",
		Timestamp: time.Now(),
		Uptime:    "24h",
	}
	component := pages.Health(data)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Targets handles the targets management with real DB data (API version)
func (h *DashboardHandler) Targets(w http.ResponseWriter, r *http.Request) {
	dbTargets, err := h.queries.ListActiveTargets(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	targets := make([]models.TargetData, 0, len(dbTargets))
	for _, t := range dbTargets {
		targets = append(targets, models.TargetData{
			ID:         t.ID,
			WebsiteURL: t.WebsiteUrl,
			SitemapURL: t.SitemapUrl.String,
			Status:     "active",
			CreatedAt:  t.CreatedAt.Time,
		})
	}
	component := pages.Targets(targets)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Logs handles the logs display with real DB data (API version)
func (h *DashboardHandler) Logs(w http.ResponseWriter, r *http.Request) {
	dbLogs, err := h.queries.GetRecentLogs(r.Context(), 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logs := make([]models.LogEntry, 0, len(dbLogs))
	for _, l := range dbLogs {
		logs = append(logs, models.LogEntry{
			Timestamp: l.CreatedAt.Time,
			Level:     l.LogType,
			Message:   l.Message,
			URL:       l.Url.String,
			Details:   l.Details.String,
		})
	}
	component := pages.Logs(logs)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
