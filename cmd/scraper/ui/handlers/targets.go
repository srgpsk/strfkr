package handlers

import (
	"net/http"
	"strconv"

	"app/cmd/scraper/ui/templates/components"
	"app/internal/scraper/db"
	"database/sql"
)

type TargetsHandler struct {
	queries db.Querier
}

func NewTargetsHandler(queries db.Querier) *TargetsHandler {
	return &TargetsHandler{queries: queries}
}

// NewForm returns the new target form for HTMX modal
func (h *TargetsHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	component := components.TargetForm()

	w.Header().Set("Content-Type", "text/html")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render form", http.StatusInternalServerError)
	}
}

// Create handles target creation via HTMX form submission
func (h *TargetsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		component := components.FormError("Invalid form data")
		if renderErr := component.Render(r.Context(), w); renderErr != nil {
			http.Error(w, renderErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	websiteURL := r.FormValue("website_url")
	sitemapURL := r.FormValue("sitemap_url")

	if websiteURL == "" {
		component := components.FormError("Website URL is required")
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	params := db.CreateTargetParams{
		WebsiteUrl:            websiteURL,
		SitemapUrl:            sql.NullString{String: sitemapURL, Valid: sitemapURL != ""},
		FollowSitemap:         sql.NullBool{Valid: false}, // default for now
		CrawlDelaySeconds:     sql.NullInt64{Valid: false},
		MaxConcurrentRequests: sql.NullInt64{Valid: false},
		UserAgent:             sql.NullString{Valid: false},
		CustomHeaders:         sql.NullString{Valid: false},
		Notes:                 sql.NullString{Valid: false},
		SitemapPatterns:       sql.NullString{Valid: false},
		UrlPatterns:           sql.NullString{Valid: false},
		DomainName:            sql.NullString{Valid: false},
	}

	_, err := h.queries.CreateTarget(r.Context(), params)
	if err != nil {
		component := components.FormError("Failed to create target: " + err.Error())
		if renderErr := component.Render(r.Context(), w); renderErr != nil {
			http.Error(w, renderErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	component := components.FormSuccess("Target created successfully")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Delete handles target deletion via HTMX
func (h *TargetsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Target ID required", http.StatusBadRequest)
		return
	}

	targetID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, "Invalid target ID", http.StatusBadRequest)
		return
	}

	err = h.queries.DeactivateTarget(r.Context(), targetID)
	if err != nil {
		http.Error(w, "Failed to delete target: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
