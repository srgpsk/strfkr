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

# Install sqlc
RUN curl -L https://github.com/sqlc-dev/sqlc/releases/download/v1.27.0/sqlc_1.27.0_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin

# Generate code during build
RUN sqlc generate -f sqlc.spider.yaml

# Expose ports
EXPOSE 8080 8081 2345 2346

# Default command
CMD ["air", "-c", ".air.toml"]

# Production stage (for later)
FROM dependencies AS production
COPY . .
RUN if find . -name "*.templ" -type f | grep -q .; then templ generate; fi
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o webapp ./cmd/webapp
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o scraper ./cmd/scraper
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o cli ./cmd/cli

EXPOSE 8080
CMD ["./webapp"]