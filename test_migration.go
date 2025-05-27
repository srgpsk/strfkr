package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Test SQLite functionality after Ubuntu/Debian migration
	dbPath := "./data/migration_test.db"
	
	// Remove existing test DB if it exists
	os.Remove(dbPath)
	
	// Create new database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()
	
	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			author TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	
	// Insert test data
	_, err = db.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", 
		"The only way to do great work is to love what you do.", "Goga Joga")
	if err != nil {
		log.Fatal("Failed to insert data:", err)
	}
	
	// Query test data
	var id int
	var text, author string
	err = db.QueryRow("SELECT id, text, author FROM quotes WHERE id = 1").Scan(&id, &text, &author)
	if err != nil {
		log.Fatal("Failed to query data:", err)
	}
	
	fmt.Printf("✅ SQLite migration test successful!\n")
	fmt.Printf("ID: %d\n", id)
	fmt.Printf("Quote: %s\n", text)
	fmt.Printf("Author: %s\n", author)
	
	// Clean up
	os.Remove(dbPath)
	fmt.Printf("✅ Database test completed and cleaned up.\n")
}
