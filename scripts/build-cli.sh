#!/bin/bash
set -e
sqlc generate -f ./internal/scraper/db/sqlc.yaml
go build -o ./bin/scraper-cli ./cmd/scraper/cli