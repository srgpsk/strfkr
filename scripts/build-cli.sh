#!/bin/bash
set -e

sqlc generate -f ./internal/scraper/db/sqlc.yaml

DB_PATH="./data/scraper.db"
MIGRATIONS_PATH="./internal/scraper/db/migrations"

# Create DB file if missing
if [ ! -f "$DB_PATH" ]; then
    mkdir -p ./data
fi

# Apply all migrations using golang-migrate
if command -v migrate >/dev/null 2>&1; then
    migrate -database "sqlite3://$DB_PATH" -path "$MIGRATIONS_PATH" up
else
    echo "Please install golang-migrate: go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

go build -o ./bin/scraper-cli ./cmd/scraper/cli