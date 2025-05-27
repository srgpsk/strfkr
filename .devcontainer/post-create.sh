#!/bin/bash

# Post-create script for dev container setup
set -e

echo "🚀 Setting up strfkr development environment..."

# Update Go tools
echo "📦 Installing/updating Go development tools..."
go install golang.org/x/tools/gopls@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/go-delve/delve/cmd/dlv@latest
# Air and Templ are already installed in the Docker image - skip reinstalling to avoid conflicts
echo "ℹ️  Air and Templ are pre-installed in the container"

# Create necessary directories
echo "📁 Creating project directories..."
# Use current directory if not in container, /app if in container
WORKDIR="${PWD}"
if [ -d "/app" ] && [ -w "/app" ]; then
    WORKDIR="/app"
fi

mkdir -p "${WORKDIR}/data/scraper"
mkdir -p "${WORKDIR}/logs"
mkdir -p "${WORKDIR}/tmp"
mkdir -p "${WORKDIR}/web/static/css"
mkdir -p "${WORKDIR}/web/static/js"
mkdir -p "${WORKDIR}/web/static/images"
mkdir -p "${WORKDIR}/web/templates"

# Set up Git safe directory (important for mounted volumes)
echo "🔧 Configuring Git..."
git config --global --add safe.directory "${WORKDIR}"
git config --global --add safe.directory '*'

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
cd "${WORKDIR}"

# Test internet connectivity first
echo "🌐 Testing internet connectivity for Copilot and Go modules..."
if curl -s --connect-timeout 5 https://github.com > /dev/null; then
    echo "✅ GitHub connection successful - Copilot should work"
else
    echo "⚠️ GitHub connection failed - Copilot may not work properly"
fi

if curl -s --connect-timeout 5 https://proxy.golang.org > /dev/null; then
    echo "✅ Go module proxy accessible"
else
    echo "⚠️ Go module proxy access failed"
fi

go mod download
go mod tidy

# Generate templ templates if any exist
echo "🎨 Generating Templ templates..."
if find . -name "*.templ" -type f | grep -q .; then
    templ generate
    echo "✅ Templ templates generated"
else
    echo "ℹ️  No .templ files found"
fi

# Set up pre-commit hooks or other development tools
echo "🔨 Setting up development tools..."

# Create .env file if it doesn't exist
if [ ! -f "${WORKDIR}/.env" ]; then
    echo "📝 Creating .env file from template..."
    if [ -f "${WORKDIR}/.env.example" ]; then
        cp "${WORKDIR}/.env.example" "${WORKDIR}/.env"
        echo "✅ .env file created. Please edit it with your configuration."
    else
        echo "ℹ️  No .env.example found, skipping .env creation"
    fi
fi

# Test that Go tools are working
echo "🧪 Testing Go environment..."
go version
gopls version
goimports --help > /dev/null && echo "✅ goimports working"
golangci-lint --version
dlv version
# Test Air and Templ (pre-installed in container)
if command -v air > /dev/null; then
    air -v && echo "✅ air working"
else
    echo "⚠️  air not found"
fi
if command -v templ > /dev/null; then
    templ --help > /dev/null && echo "✅ templ working"
else
    echo "⚠️  templ not found"
fi

# Make sure we can build the project
echo "🏗️  Testing project build..."
if go build -o /tmp/test-build ./cmd/webapp; then
    echo "✅ Project builds successfully"
    rm -f /tmp/test-build
else
    echo "⚠️  Project build failed - you may need to fix dependencies"
fi

echo ""
echo "🎉 Development environment setup complete!"
echo ""
echo "📝 You're in a Dev Container - services are managed externally."
echo "🛠️  Available commands:"
echo "  make help          - Show all available make targets"
echo "  go run ./cmd/webapp/main.go - Run webapp directly"
echo "  go run ./cmd/scraper/main.go - Run scraper directly"
echo "  make test          - Run tests"
echo "  make lint          - Run linter"
echo "  make format        - Format code"
echo ""
echo "🌐 Services (if running):"
echo "  http://localhost:8080 - Webapp (when started)"
echo "  http://localhost:8081 - Scraper (when started)"
echo ""
echo "🐛 Debugging:"
echo "  F5                 - Start debugging (use 'Debug Webapp' configuration)"
echo "  Ctrl+Shift+P       - Command palette"
echo ""
echo "📁 Important directories:"
echo "  ${WORKDIR}/data          - Database and data files"
echo "  ${WORKDIR}/web           - Web templates and static assets"
echo "  ${WORKDIR}/cmd           - Application entry points"
echo ""
