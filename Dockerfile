# Multi-stage Dockerfile for Go development
FROM golang:1.24-bullseye AS development

# Install development tools
RUN go install github.com/air-verse/air@latest
RUN go install github.com/a-h/templ/cmd/templ@latest

# Install build dependencies and common tools
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    wget \
    curl \
    sqlite3 \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Create directory structure
RUN mkdir -p /app/cmd/webapp \
    /app/cmd/scraper \
    /app/cmd/cli \
    /app/internal \
    /app/pkg \
    /app/web/templates \
    /app/web/static \
    /app/data

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application
COPY . .

# Expose port
EXPOSE 8080

# Default command for development (will be overridden by docker-compose)
CMD ["air"]

# Debug stage - includes Delve debugger
FROM golang:1.24-bullseye AS debug

# Install debug tools and dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    wget \
    curl \
    sqlite3 \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install github.com/cosmtrek/air@latest
RUN go install github.com/a-h/templ/cmd/templ@latest

# Set working directory
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with debug symbols (no optimization, include debug info)
RUN CGO_ENABLED=1 GOOS=linux go build -gcflags="all=-N -l" -o webapp ./cmd/webapp
RUN CGO_ENABLED=1 GOOS=linux go build -gcflags="all=-N -l" -o scraper ./cmd/scraper
RUN CGO_ENABLED=1 GOOS=linux go build -gcflags="all=-N -l" -o cli ./cmd/cli

# Expose application and debugger ports
EXPOSE 8080 8081 2345 2346

# Default command runs webapp with delve
CMD ["dlv", "exec", "./webapp", "--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "--continue"]

# Builder stage for production
FROM golang:1.24-bullseye AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    sqlite3 \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build optimized binaries for production
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o webapp ./cmd/webapp
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o scraper ./cmd/scraper
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o cli ./cmd/cli

# Final production stage
FROM ubuntu:22.04 AS production

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    sqlite3 \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Copy binaries from builder stage
COPY --from=builder /app/webapp .
COPY --from=builder /app/scraper .
COPY --from=builder /app/cli .

# Copy web assets
COPY --from=builder /app/web ./web

# Create data directory
RUN mkdir -p /root/data

# Add healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Default command
CMD ["./webapp"]
