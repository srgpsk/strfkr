# Test: Single file with identical structure
FROM golang:1.24-bullseye AS base

# Install system dependencies (from Dockerfile.dev-base)
RUN apt-get update && apt-get install -y \
    git ca-certificates tzdata wget curl sqlite3 libsqlite3-dev \
    build-essential pkg-config \
    htop tree jq vim nano \
    net-tools iputils-ping telnet \
    yamllint \
    gnupg2 \
    && rm -rf /var/lib/apt/lists/* && apt-get clean

# Install Go development tools (from Dockerfile.dev-base)
RUN go install github.com/air-verse/air@latest && \
    go install github.com/a-h/templ/cmd/templ@latest && \
    go install golang.org/x/tools/gopls@latest && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Set up environment and workspace (from Dockerfile.dev-base)
WORKDIR /app
RUN mkdir -p cmd internal pkg web/templates web/static data
ENV CGO_ENABLED=1
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

FROM base AS dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

FROM dependencies AS development  
COPY . .
RUN if find . -name "*.templ" -type f | grep -q .; then templ generate; fi
EXPOSE 8080 8081 2345 2346
CMD ["air", "-c", ".air.toml"]