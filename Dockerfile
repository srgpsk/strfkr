# Use our pre-built base image
FROM strfkr-base:latest AS base

# Dependencies stage - only rebuilds when go.mod/go.sum changes
FROM base AS dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Development stage - fast rebuilds for code changes
FROM dependencies AS development  
COPY . .

# Generate templates if they exist
RUN if find . -name "*.templ" -type f | grep -q .; then \
        echo "🎨 Generating templates..." && \
        templ generate; \
    else \
        echo "ℹ️  No .templ files found, skipping template generation"; \
    fi

# Expose ports
EXPOSE 8080 8081 2345 2346

# Generate sqlc code at startup, then run air
CMD ["sh", "-c", "mkdir -p internal/scraper/db && sqlc generate -f sqlc.scraper.yaml && air -c .air.toml"]

# Production stage (for later)
FROM dependencies AS production
COPY . .
RUN if find . -name "*.templ" -type f | grep -q .; then templ generate; fi
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o webapp ./cmd/webapp
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o scraper ./cmd/scraper
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o cli ./cmd/cli

EXPOSE 8080
CMD ["./webapp"]