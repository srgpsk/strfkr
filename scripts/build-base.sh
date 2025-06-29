#!/bin/bash
set -e

echo "🏗️  Building base development image..."

# Parse command line arguments
BUILD_ARGS=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-cache)
            BUILD_ARGS="$BUILD_ARGS --no-cache"
            echo "🔄 Building with --no-cache flag"
            shift
            ;;
        --pull)
            BUILD_ARGS="$BUILD_ARGS --pull"
            echo "🔄 Building with --pull flag"
            shift
            ;;
        *)
            echo "Unknown argument: $1"
            echo "Usage: $0 [--no-cache] [--pull]"
            exit 1
            ;;
    esac
done

# Build the base image with collected arguments
docker build \
    $BUILD_ARGS \
    -f docker/base/Dockerfile.base \
    -t strfkr-base:latest \
    .

echo "✅ Base image built successfully!"
echo ""
echo "📦 Available images:"
docker images | grep strfkr-base

echo ""
echo "💡 To use this base image, rebuild your development containers:"
echo "   docker compose build"