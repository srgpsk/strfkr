#!/bin/bash
set -e

echo "🧪 Running tests with coverage..."

# Generate coverage for all packages
go test -coverprofile=coverage.out ./...

# Remove excluded files from coverage
if [ -f coverage.ignore ]; then
    echo "Excluding files from coverage calculation..."
    while IFS= read -r pattern; do
        # Skip empty lines and comments
        [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
        
        # Remove matching lines from coverage file
        grep -v "$pattern" coverage.out > coverage.tmp && mv coverage.tmp coverage.out || true
    done < coverage.ignore
fi

# Calculate final coverage
COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
echo "📊 Final coverage: $COVERAGE%"

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
echo "📄 HTML report generated: coverage.html"

echo $COVERAGE