#!/bin/bash
set -e
 
# Minimal Go coverage script for pre-push hook, with pattern exclusion

go test -coverprofile=coverage.out ./...

# Exclude patterns from coverage.out if coverage.ignore exists
if [ -f coverage.ignore ]; then
    echo "Excluding patterns from coverage.out..."
    cp coverage.out coverage.tmp
    while IFS= read -r pattern; do
        [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
        grep -v "$pattern" coverage.tmp > coverage.tmp2
        mv coverage.tmp2 coverage.tmp
        echo "Excluding pattern: $pattern"
    done < coverage.ignore
    mv coverage.tmp coverage.out
fi

COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')

echo $COVERAGE