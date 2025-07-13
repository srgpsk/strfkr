#!/bin/bash
# Local development setup for strfkr project
set -e

echo "Setting up strfkr project for local development..."

# Check if we're in the right directory
if [ ! -f "go.mod" ] || ! grep -q "module app" go.mod; then
    echo "❌ Please run this script from the strfkr project root directory"
    exit 1
fi

# Install Go 1.24.4
echo "📦 Installing Go 1.24.4..."
if ! command -v go &> /dev/null || ! go version | grep -q "go1.24"; then
    echo "Downloading Go 1.24.4..."
    wget -q https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
    rm go1.24.4.linux-amd64.tar.gz
    
    # Add to PATH if not already there
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        echo 'export PATH=$GOPATH/bin:$PATH' >> ~/.bashrc
    fi
    
    export PATH=/usr/local/go/bin:$PATH
    export GOPATH=$HOME/go
    export PATH=$GOPATH/bin:$PATH
fi

echo "✅ Go version: $(go version)"

# Install required Go tools based on your project
echo "🔧 Installing Go development tools..."
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/air-verse/air@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/swaggo/swag/cmd/swag@latest
go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/google/yamlfmt/cmd/yamlfmt@latest
go install github.com/a-h/templ/cmd/templ@latest

# Install golangci-lint
echo "🔍 Installing golangci-lint..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# Install SQLite (your project uses SQLite, not PostgreSQL)
echo "🗄️  Installing SQLite..."
sudo apt update
sudo apt install -y sqlite3 libsqlite3-dev

# Create necessary directories
echo "📁 Creating project directories..."
mkdir -p tmp internal/scraper/db data

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Generate SQLC code
echo "🏗️  Generating SQLC code..."
if [ -f "internal/scraper/db/sqlc.yaml" ]; then
    sqlc generate -f internal/scraper/db/sqlc.yaml
    echo "✅ SQLC code generated"
else
    echo "⚠️  internal/scraper/db/sqlc.yaml not found, skipping SQLC generation"
fi

# Generate templates if they exist
echo "🎨 Generating templates..."
if find . -name "*.templ" -type f | grep -q .; then
    templ generate
    echo "✅ Templates generated"
else
    echo "⚠️  No .templ files found, skipping template generation"
fi

# Set up environment file
echo "🔧 Setting up environment..."
if [ ! -f ".env" ]; then
    cat > .env << 'EOF'
# SQLite database file
DATABASE_URL=data/scraper.db
PORT=8080
ENV=development
EOF
    echo "✅ Created .env file"
else
    echo "⚠️  .env file already exists, skipping"
fi

# Initialize SQLite database
echo "🗄️  Setting up SQLite database..."
mkdir -p data
if [ ! -f "data/scraper.db" ]; then
    sqlite3 data/scraper.db "SELECT 1;" > /dev/null 2>&1
    echo "✅ SQLite database initialized at data/scraper.db"
else
    echo "⚠️  Database already exists at data/scraper.db"
fi

# Set up git hooks (modify for local environment)
echo "🔗 Setting up git hooks..."
if [ -f ".githooks/install.sh" ]; then
    chmod +x .githooks/install.sh
    # Run hooks installer but skip container-specific configuration
    export LOCAL_INSTALL=true
    ./.githooks/install.sh
    echo "✅ Git hooks configured"
else
    echo "⚠️  Git hooks script not found, skipping"
fi

# Test the setup
echo "🧪 Testing setup..."

# Test Go build
echo "Testing Go build..."
if go build -o tmp/test-build ./cmd/scraper > /dev/null 2>&1; then
    echo "✅ Go build successful"
    rm -f tmp/test-build
else
    echo "❌ Go build failed"
    exit 1
fi

# Test SQLite connection
echo "Testing SQLite connection..."
if sqlite3 data/scraper.db "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ SQLite connection successful"
else
    echo "❌ SQLite connection failed"
    exit 1
fi

# Test Air configuration
echo "Testing Air configuration..."
if air -c .air-scraper.toml --help > /dev/null 2>&1; then
    echo "✅ Air configuration valid"
else
    echo "❌ Air configuration invalid"
    exit 1
fi

echo ""
echo "🎉 Setup complete!"
echo ""
echo "📋 Next steps:"
echo "  1. Source your environment: source ~/.bashrc"
echo "  2. Run the scraper with hot reload: air -c .air-scraper.toml"
echo "  3. Or build and run manually: go run ./cmd/scraper"
echo "  4. Run tests: go test ./..."
echo "  5. Run web app: go run ./cmd/webapp"
echo ""
echo "📁 Project structure:"
echo "  - SQLite database: data/scraper.db"
echo "  - Generated code: internal/scraper/db/"
echo "  - Temp files: tmp/"
echo "  - Environment: .env"
echo ""
echo "🔧 Available commands:"
echo "  - air -c .air-scraper.toml  # Run scraper with hot reload"
echo "  - air -c .air.toml          # Run web app with hot reload"
echo "  - go test ./...             # Run tests"
echo "  - sqlc generate -f internal/scraper/db/sqlc.yaml  # Regenerate DB code"
echo "  - templ generate            # Regenerate templates"
echo ""
echo "✅ Ready for development!"