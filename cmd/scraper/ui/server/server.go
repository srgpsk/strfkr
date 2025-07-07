package server

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"app/cmd/scraper/ui/handlers"
	"app/internal/scraper/db"
)

// Server holds the application dependencies and routes
type Server struct {
	queries *db.Queries
	db      *sql.DB
	mux     *http.ServeMux

	// Handler instances
	dashboardHandler *handlers.DashboardHandler
	apiHandler       *handlers.APIHandler
	targetsHandler   *handlers.TargetsHandler
}

// New creates a new server instance
func New(database *sql.DB) *Server {
	queries := db.New(database)

	s := &Server{
		queries: queries,
		db:      database,
		mux:     http.NewServeMux(),
	}

	// Initialize handlers
	s.dashboardHandler = handlers.NewDashboardHandler(queries)
	s.apiHandler = handlers.NewAPIHandler(queries)
	s.targetsHandler = handlers.NewTargetsHandler(queries)

	// Setup routes
	s.setupRoutes()

	return s
}

// setupRoutes configures all application routes
func (s *Server) setupRoutes() {
	// Static files for admin UI
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Main admin routes
	s.mux.Handle("GET /", withMiddleware(s.dashboardHandler.Dashboard))
	s.mux.Handle("GET /health", withMiddleware(s.dashboardHandler.HealthAPI))

	// HTMX API routes for admin functionality
	s.mux.Handle("GET /api/stats", withMiddleware(s.apiHandler.Stats))
	s.mux.Handle("GET /api/targets", withMiddleware(s.apiHandler.TargetsList))
	s.mux.Handle("GET /api/logs", withMiddleware(s.apiHandler.Logs))

	// Target management routes
	s.mux.Handle("GET /targets/new", withMiddleware(s.targetsHandler.NewForm))
	s.mux.Handle("POST /api/targets", withMiddleware(s.targetsHandler.Create))
	s.mux.Handle("DELETE /api/targets/{id}", withMiddleware(s.targetsHandler.Delete))

	// Crawling control routes - ADD THESE
	s.mux.Handle("POST /api/crawl/start", withMiddleware(s.apiHandler.StartCrawling))
	s.mux.Handle("POST /api/sitemap/refresh-all", withMiddleware(s.apiHandler.RefreshSitemaps))
}

// Handler returns the main HTTP handler
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Middleware wrapper for logging
func withLogging(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

// Middleware wrapper for recovery
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain multiple middleware
func withMiddleware(handler http.HandlerFunc) http.Handler {
	return withRecovery(withLogging(handler))
}
