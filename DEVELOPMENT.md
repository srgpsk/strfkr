# Development Guide

## Quick Start

1. **Initial Setup**
   ```bash
   make setup
   make env-copy
   # Edit .env file as needed
   ```

2. **Start Development Environment**
   ```bash
   make dev
   ```

3. **Access the application**
   - Main app: http://localhost
   - Webapp direct: http://localhost:8080
   - Scraper admin: http://localhost/scraper
   - Scraper direct: http://localhost:8081

## Development Workflow

### Hot Reloading
The development environment uses `air` for hot reloading:
- Go files: Automatically rebuild and restart
- Templates (Templ): Automatically regenerate
- Static files: Automatically sync
- Nginx config: Automatically reload

### Available Commands

```bash
# Development
make dev              # Start with hot reloading
make dev-detached     # Start in background
make dev-clean        # Clean start
make stop             # Stop all services
make restart          # Restart all services

# Building
make build            # Build all services
make build-prod       # Build production images

# Testing
make test             # Run tests
make test-coverage    # Run tests with coverage
make benchmark        # Run benchmarks

# Database
make db-reset         # Reset database
make db-migrate       # Run migrations
make db-seed          # Seed with sample data

# Code Quality
make lint             # Run linter
make format           # Format code

# Utilities
make logs             # Show all logs
make logs-webapp      # Show webapp logs only
make health-check     # Check service health
```

## VSCode Integration

The project includes VSCode configuration for:
- Go debugging with Delve
- Task definitions for common operations
- Settings optimized for Go development
- Templ template support

### Debugging
1. Start the development environment: `make dev`
2. Use "Attach to Webapp Container" launch configuration
3. Set breakpoints and debug

## File Watching & Performance

### Docker Compose Watch
The setup uses Docker Compose's `--watch` feature for optimal performance:
- **sync+restart**: Go source code changes trigger rebuild and restart
- **sync**: Templates and static files sync without restart
- **Cached volumes**: Go mod cache and build cache persist between restarts

### Volume Mounts
- `:cached` - Optimized for host writes, container reads
- Named volumes for Go cache - Persist between container recreations
- Separate tmp volumes - Avoid conflicts between services

## Troubleshooting

### Port Conflicts
If you get port binding errors:
```bash
make stop
# Check what's using the ports
sudo netstat -tulpn | grep :80
sudo netstat -tulpn | grep :8080
```

### Cache Issues
Clear Docker caches:
```bash
make clean        # Clean containers and volumes
make clean-all    # Clean everything including images
```

### Database Issues
Reset database:
```bash
make db-reset
```

### File Permission Issues (Linux/macOS)
Ensure proper ownership:
```bash
sudo chown -R $USER:$USER .
```

## Performance Optimization

### Go Module Cache
- Cached in named volume `go-mod-cache`
- Shared between all Go containers
- Persists between container recreations

### Build Cache
- Cached in named volume `go-build-cache`
- Speeds up subsequent builds
- Shared between webapp and scraper

### Air Configuration
- Optimized exclude patterns
- Separate config for webapp and scraper
- Minimal rebuild scope

## VS Code Extensions Recommended

- Go (Google)
- Docker (Microsoft)
- HTMX Attributes (otovo-oss)
- SQLite Viewer (qwtel)
- Thunder Client (for API testing)

## Environment Variables

Copy `.env.example` to `.env` and customize:
- Database paths
- Ports
- Debug settings
- Rate limiting configuration
- Security settings (for development)

## Next Steps

After the development environment is running:
1. Implement the basic quote models in `internal/models/`
2. Set up database schema and migrations
3. Create basic HTTP handlers
4. Implement Templ templates
5. Add HTMX interactions
6. Implement the scraper logic
