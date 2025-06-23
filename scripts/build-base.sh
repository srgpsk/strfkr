#!/bin/bash
set -e

echo "🏗️  Building base development image..."

# Build the base image with just latest tag
docker build \
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