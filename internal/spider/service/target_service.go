package service

import (
	"context"
	"database/sql"

	"strfkr/internal/spider/config"
	"strfkr/internal/spider/db"
)

// TargetQueries defines the interface needed for target service operations
type TargetQueries interface {
	CreateTarget(ctx context.Context, params db.CreateTargetParams) (db.SpiderTarget, error)
}

type TargetService struct {
	queries TargetQueries
}

func NewTargetService(queries TargetQueries) *TargetService {
	return &TargetService{queries: queries}
}

// CreateTargetWithDefaults creates a target with default patterns
func (s *TargetService) CreateTargetWithDefaults(ctx context.Context, websiteURL, sitemapURL string) error {
	domain := config.ExtractDomain(websiteURL)
	patterns := config.DefaultPatterns()

	sitemapPatternsJSON, _ := patterns.ToJSON()
	urlPatternsJSON, _ := config.PatternConfig{URLPatterns: patterns.URLPatterns}.ToJSON()

	_, err := s.queries.CreateTarget(ctx, db.CreateTargetParams{
		WebsiteUrl:            websiteURL,
		SitemapUrl:            sql.NullString{String: sitemapURL, Valid: true},
		FollowSitemap:         sql.NullBool{Bool: true, Valid: true},
		CrawlDelaySeconds:     sql.NullInt64{Int64: 1, Valid: true},
		MaxConcurrentRequests: sql.NullInt64{Int64: 5, Valid: true},
		SitemapPatterns:       sql.NullString{String: sitemapPatternsJSON, Valid: true},
		UrlPatterns:           sql.NullString{String: urlPatternsJSON, Valid: true},
		DomainName:            sql.NullString{String: domain, Valid: true},
	})

	return err
}
