package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"app/cmd/scraper/ui/models"
	"app/cmd/scraper/ui/templates/components"
	"app/internal/scraper/db"
)

type APIHandler struct {
	queries *db.Queries
}

func NewAPIHandler(queries *db.Queries) *APIHandler {
	return &APIHandler{queries: queries}
}

// Stats returns stats widget for HTMX
func (h *APIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	ctx := r.Context()

	// Get target count
	targetCount := 0
	if count, err := h.queries.GetTargetCount(ctx); err == nil {
		targetCount = int(count)
	}

	// Get pending queue count
	pendingCount := 0
	if count, err := h.queries.GetPendingQueueCount(ctx); err == nil {
		pendingCount = int(count)
	}

	// Get total pages count
	totalPages := 0
	if count, err := h.queries.GetTotalPagesCount(ctx); err == nil {
		totalPages = int(count)
	}

	// Get recent errors count
	recentErrors := 0
	if count, err := h.queries.GetRecentErrorsCount(ctx); err == nil {
		recentErrors = int(count)
	}

	stats := models.StatsData{
		Targets:      targetCount,
		PendingQueue: pendingCount,
		TotalPages:   totalPages,
		RecentErrors: recentErrors,
	}

	component := components.StatsCards(stats)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TargetsList returns targets list for HTMX
func (h *APIHandler) TargetsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	ctx := r.Context()

	dbTargets, err := h.queries.ListActiveTargets(ctx)
	if err != nil {
		log.Printf("Error getting targets: %v", err)
		component := components.TargetsList([]models.TargetData{})
		if renderErr := component.Render(r.Context(), w); renderErr != nil {
			http.Error(w, renderErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if len(dbTargets) > limit {
		dbTargets = dbTargets[:limit]
	}

	targets := make([]models.TargetData, len(dbTargets))
	for i, dbTarget := range dbTargets {
		status := "inactive"
		if dbTarget.IsActive.Valid && dbTarget.IsActive.Bool {
			status = "active"
		}
		targets[i] = models.TargetData{
			ID:         dbTarget.ID,
			WebsiteURL: dbTarget.WebsiteUrl,
			SitemapURL: dbTarget.SitemapUrl.String,
			Status:     status,
			CreatedAt:  dbTarget.CreatedAt.Time,
		}
	}

	component := components.TargetsList(targets)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Logs returns recent logs for HTMX
func (h *APIHandler) Logs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	ctx := r.Context()

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	dbLogs, err := h.queries.GetRecentLogs(ctx, int64(limit))
	if err != nil {
		log.Printf("Error getting logs: %v", err)
		logs := []models.LogEntry{}
		component := components.LogsList(logs)
		if renderErr := component.Render(r.Context(), w); renderErr != nil {
			http.Error(w, renderErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	logs := make([]models.LogEntry, len(dbLogs))
	for i, dbLog := range dbLogs {
		logs[i] = models.LogEntry{
			Timestamp: dbLog.CreatedAt.Time,
			Level:     dbLog.LogType,
			Message:   dbLog.Message,
			URL:       dbLog.Url.String,
			Details:   dbLog.Details.String,
		}
	}

	component := components.LogsList(logs)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StartCrawling logs the action
func (h *APIHandler) StartCrawling(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	ctx := r.Context()

	err := h.queries.LogMessage(ctx, db.LogMessageParams{
		LogType:  "info",
		TargetID: sql.NullInt64{Valid: false},
		Url:      sql.NullString{Valid: false},
		Message:  "Crawling started via admin interface",
		Details:  sql.NullString{String: "Started by user", Valid: true},
	})
	if err != nil {
		log.Printf("Error adding log: %v", err)
	}

	component := components.StatusMessage("success", "Crawling started successfully")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// RefreshSitemaps logs the action
func (h *APIHandler) RefreshSitemaps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	ctx := r.Context()

	err := h.queries.LogMessage(ctx, db.LogMessageParams{
		LogType:  "info",
		TargetID: sql.NullInt64{Valid: false},
		Url:      sql.NullString{Valid: false},
		Message:  "Sitemap refresh initiated via admin interface",
		Details:  sql.NullString{String: "Initiated by user", Valid: true},
	})
	if err != nil {
		log.Printf("Error adding log: %v", err)
	}

	component := components.StatusMessage("info", "Sitemaps refresh initiated")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
