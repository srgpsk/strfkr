# strfkr - Famous Quotes Website

## Setup
```bash
make setup
make env-copy  # Edit .env file as needed
```

## Development Commands
```bash
# Start/Stop
make dev              # Start with hot reloading
make dev-detached     # Start in background  
make stop             # Stop all services
make restart          # Restart all services

# Direct execution (in dev container)
go run ./cmd/webapp/main.go    # Port 8080
go run ./cmd/scraper/main.go   # Port 8081

# Building & Testing
make build            # Build all services
make test             # Run tests
make test-coverage    # Run tests with coverage
make lint             # Run linter
make format           # Format code

# Database
make db-reset         # Reset database
make db-migrate       # Run migrations
make db-seed          # Seed with sample data

# Utilities
make logs             # Show all logs
make logs-webapp      # Show webapp logs only
make health-check     # Check service health
```

## URLs
- Main app: http://localhost
- Webapp direct: http://localhost:8080  
- Scraper admin: http://localhost/scraper
- Scraper direct: http://localhost:8081

## Troubleshooting
```bash
# Port conflicts
make stop
sudo netstat -tulpn | grep :80

# Cache issues  
make clean        # Clean containers and volumes
make clean-all    # Clean everything including images

```
