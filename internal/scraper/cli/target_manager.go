package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"app/internal/scraper/db"
	"app/internal/scraper/service"
	"app/internal/scraper/sitemap"

	_ "github.com/mattn/go-sqlite3"
)

type TargetManager struct {
	db            *sql.DB
	queries       *db.Queries
	targetService *service.TargetService
	parser        *sitemap.Parser
}

type TargetInfo struct {
	ID          int64      `json:"id"`
	WebsiteURL  string     `json:"website_url"`
	SitemapURL  string     `json:"sitemap_url"`
	Active      bool       `json:"active"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastVisited *time.Time `json:"last_visited,omitempty"`
}

func NewTargetManager() (*TargetManager, error) {
	// Initialize database connection
	database, err := sql.Open("sqlite3", "data/scraper.db?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := db.New(database)
	targetService := service.NewTargetService(queries)

	// Create sitemap parser with database access
	parser := sitemap.NewParser(queries, 30*time.Second)

	return &TargetManager{
		db:            database,
		queries:       queries,
		targetService: targetService,
		parser:        parser,
	}, nil
}

func (tm *TargetManager) Close() error {
	return tm.db.Close()
}

func (tm *TargetManager) AddTarget(websiteURL, sitemapURL, userAgent string, autoDiscover, validate bool) error {
	ctx := context.Background()
	ss := service.NewSitemapService(30 * time.Second)

	fmt.Printf("Adding target: %s\n", websiteURL)

	// Auto-discover sitemap if needed
	if sitemapURL == "" && autoDiscover {
		discovered, err := ss.AutoDiscoverSitemap(websiteURL)
		if err != nil {
			return fmt.Errorf("failed to discover sitemap: %w", err)
		}
		sitemapURL = discovered
		fmt.Printf("✅ Auto-discovered sitemap: %s\n", sitemapURL)
	}

	// Validate sitemap if requested
	if validate && sitemapURL != "" {
		if err := ss.ValidateSitemap(ctx, sitemapURL, userAgent); err != nil {
			return fmt.Errorf("sitemap validation failed: %w", err)
		}
		fmt.Printf("✅ Sitemap validation passed\n")
	}

	// Create target in database
	params := db.CreateTargetParams{
		WebsiteUrl:            websiteURL,
		SitemapUrl:            sql.NullString{String: sitemapURL, Valid: sitemapURL != ""},
		FollowSitemap:         sql.NullBool{Bool: sitemapURL != "", Valid: true},
		UserAgent:             sql.NullString{String: userAgent, Valid: userAgent != ""},
		CrawlDelaySeconds:     sql.NullInt64{Int64: 1, Valid: true},
		MaxConcurrentRequests: sql.NullInt64{Int64: 3, Valid: true},
	}

	target, err := tm.queries.CreateTarget(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}

	fmt.Printf("✅ Target created successfully with ID: %d\n", target.ID)
	return nil
}

func (tm *TargetManager) ListTargets(activeOnly bool) ([]TargetInfo, error) {
	ctx := context.Background()

	var targets []db.ScraperTarget
	var err error

	if activeOnly {
		targets, err = tm.queries.ListActiveTargets(ctx)
	} else {
		targets, err = tm.queries.ListAllTargets(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}

	result := make([]TargetInfo, len(targets))
	for i, target := range targets {
		result[i] = TargetInfo{
			ID:         target.ID,
			WebsiteURL: target.WebsiteUrl,
			SitemapURL: target.SitemapUrl.String,
			Active:     target.IsActive.Bool,
			UpdatedAt:  target.UpdatedAt.Time,
		}
		if target.LastVisitedAt.Valid {
			result[i].LastVisited = &target.LastVisitedAt.Time
		}
	}

	return result, nil
}

func (tm *TargetManager) PrintTargetsJSON(targets []TargetInfo) error {
	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func (tm *TargetManager) ShowTarget(targetID int64) error {
	ctx := context.Background()

	target, err := tm.queries.GetTarget(ctx, targetID)
	if err != nil {
		return fmt.Errorf("failed to get target: %w", err)
	}

	fmt.Printf("Target ID: %d\n", target.ID)
	fmt.Printf("Website URL: %s\n", target.WebsiteUrl)
	fmt.Printf("Sitemap URL: %s\n", target.SitemapUrl.String)
	fmt.Printf("Active: %t\n", target.IsActive.Bool)
	fmt.Printf("Created: %s\n", target.CreatedAt.Time.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", target.UpdatedAt.Time.Format(time.RFC3339))

	if target.LastVisitedAt.Valid {
		fmt.Printf("Last Visited: %s\n", target.LastVisitedAt.Time.Format(time.RFC3339))
	}

	if target.UserAgent.Valid {
		fmt.Printf("User Agent: %s\n", target.UserAgent.String)
	}

	return nil
}

func (tm *TargetManager) RemoveTarget(targetID int64, force bool) error {
	ctx := context.Background()

	// Check if target exists
	target, err := tm.queries.GetTarget(ctx, targetID)
	if err != nil {
		return fmt.Errorf("failed to get target: %w", err)
	}

	// Confirm deletion unless forced
	if !force {
		fmt.Printf("Are you sure you want to remove target %d (%s)? [y/N]: ", targetID, target.WebsiteUrl)
		var response string
		n, err := fmt.Scanln(&response)
		if err != nil && n == 0 {
			fmt.Printf("failed to read input: %v\n", err)
		}
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Removal cancelled.")
			return nil
		}
	}

	// Deactivate target (soft delete)
	err = tm.queries.DeactivateTarget(ctx, targetID)
	if err != nil {
		return fmt.Errorf("failed to deactivate target: %w", err)
	}

	fmt.Printf("✅ Target %d (%s) has been deactivated\n", targetID, target.WebsiteUrl)
	return nil
}
