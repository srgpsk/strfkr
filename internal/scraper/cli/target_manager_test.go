package cli

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"app/internal/scraper/db"
)

// Add minimal tests for TargetManager methods
func TestTargetManager_PrintTargetsJSON(t *testing.T) {
	tm := &TargetManager{}
	targets := []TargetInfo{{ID: 1, WebsiteURL: "https://a.com", SitemapURL: "https://a.com/sitemap.xml", Active: true, UpdatedAt: time.Now()}}
	if err := tm.PrintTargetsJSON(targets); err != nil {
		t.Errorf("PrintTargetsJSON failed: %v", err)
	}
}

func TestTargetManager_ShowTarget_Panic(t *testing.T) {
	tm := &TargetManager{queries: nil}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil queries, but did not panic")
		}
	}()
	err := tm.ShowTarget(123)
	_ = err // ignore error, test is for panic
}

func TestTargetManager_RemoveTarget_Force_Panic(t *testing.T) {
	tm := &TargetManager{queries: nil}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil queries, but did not panic")
		}
	}()
	err := tm.RemoveTarget(123, true)
	_ = err // ignore error, test is for panic
}

func TestTargetManager_AddAndShowTarget_InMemory(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer func() {
		cerr := dbConn.Close()
		if cerr != nil {
			t.Errorf("failed to close db: %v", cerr)
		}
	}()

	// Debug: print working directory and attempted schema path
	cwd, _ := os.Getwd()
	println("[DEBUG] CWD:", cwd)
	println("[DEBUG] Attempting to read schema at ../db/migrations/001_initial_schema.sql")

	// Run all migrations needed for test DB
	migrationFiles := []string{
		"../db/migrations/001_initial_schema.sql",
		"../db/migrations/002_rate_limit_per_target.sql",
	}
	for _, mf := range migrationFiles {
		println("[DEBUG] Applying migration:", mf)
		schema, err := os.ReadFile(mf)
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", mf, err)
		}
		_, err = dbConn.Exec(string(schema))
		if err != nil {
			t.Fatalf("failed to exec migration %s: %v", mf, err)
		}
	}

	queries := db.New(dbConn)
	tm := &TargetManager{db: dbConn, queries: queries}

	website := "https://test.com"
	sitemap := "https://test.com/sitemap.xml"
	_, err = tm.queries.CreateTarget(context.Background(), db.CreateTargetParams{
		WebsiteUrl:            website,
		SitemapUrl:            sql.NullString{String: sitemap, Valid: true},
		FollowSitemap:         sql.NullBool{Bool: true, Valid: true},
		CrawlDelaySeconds:     sql.NullInt64{Int64: 1, Valid: true},
		MaxConcurrentRequests: sql.NullInt64{Int64: 3, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Should not panic
	err = tm.ShowTarget(1)
	if err != nil {
		t.Errorf("ShowTarget failed: %v", err)
	}
}
