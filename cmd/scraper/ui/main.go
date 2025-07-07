package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"app/cmd/scraper/ui/server"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	// Get SCRAPER_PORT - no fallback, must be set
	port := os.Getenv("SCRAPER_PORT")
	if port == "" {
		log.Fatalf("SCRAPER_PORT environment variable is required")
	}

	// Get database path - no fallback, must be set
	dbPath := os.Getenv("SCRAPER_DB_PATH")
	if dbPath == "" {
		log.Fatalf("SCRAPER_DB_PATH environment variable is required")
	}

	// Initialize database
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		err := database.Close()
		if err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	// Create server with all routes and handlers
	srv := server.New(database)

	log.Printf("🕷️  Starting scraper server with admin UI on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv.Handler()))
}
