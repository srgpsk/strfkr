#!/bin/bash
set -euo pipefail

# Default values
CACHE_FROM=""
NO_CACHE=""
SKIP_GPG=""
SKIP_SSH=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        --cache-from)
            CACHE_FROM="--cache-from $2"
            shift 2
            ;;
        --skip-gpg)
            SKIP_GPG="true"
            shift
            ;;
        --skip-ssh)
            SKIP_SSH="true"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --no-cache          Build without using cache"
            echo "  --cache-from IMAGE  Use specific image for cache"
            echo "  --skip-gpg          Skip copying GPG keys"
            echo "  --skip-ssh          Skip copying SSH keys"
            echo "  --help, -h          Show this help message"
            echo ""
            echo "Default behavior:"
            echo "  - GPG keys are copied if available"
            echo "  - SSH keys are copied if available"
            echo ""
            echo "Examples:"
            echo "  $0                  # Copy all available keys"
            echo "  $0 --no-cache       # Build without cache, copy keys"
            echo "  $0 --skip-gpg       # Copy only SSH keys"
            echo "  $0 --skip-gpg --skip-ssh  # Don't copy any keys"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "Building base image with development tools..."

# Create temporary key directories in project root
cleanup() {
    # Fix permissions before cleanup to avoid permission denied errors
    if [ -d ".build-keys-temp" ]; then
        chmod -R 755 ".build-keys-temp" 2>/dev/null || sudo chmod -R 755 ".build-keys-temp"
        rm -rf ".build-keys-temp" 2>/dev/null || sudo rm -rf ".build-keys-temp"
    fi
}
trap cleanup EXIT

mkdir -p .build-keys-temp

# Default: copy GPG keys if available (unless explicitly skipped)
COPY_GPG=""
if [ "$SKIP_GPG" != "true" ]; then
    if [ -d "$HOME/.gnupg" ] && [ "$(ls -A $HOME/.gnupg 2>/dev/null)" ]; then
        COPY_GPG="--build-arg COPY_GPG=true"
        echo "GPG keys found, copying to build context"

        # Use sudo to copy files with preserved permissions
        sudo cp -r "$HOME/.gnupg" ".build-keys-temp/.gnupg"
        
        # Change ownership to current user for Docker build context
        sudo chown -R $(id -u):$(id -g) ".build-keys-temp/.gnupg"
        
        # Fix permissions for Docker build context (Docker needs to read these files)
        find ".build-keys-temp/.gnupg" -type d -exec chmod 755 {} \;
        find ".build-keys-temp/.gnupg" -type f -exec chmod 644 {} \;
        
        echo "GPG keys copied"
    else
        echo "No GPG keys found, skipping"
        mkdir -p ".build-keys-temp/.gnupg"
    fi
else
    echo "GPG key copying skipped via --skip-gpg"
    mkdir -p ".build-keys-temp/.gnupg"
fi

# Default: copy SSH keys if available (unless explicitly skipped)
COPY_SSH=""
if [ "$SKIP_SSH" != "true" ]; then
    if [ -d "$HOME/.ssh" ] && [ "$(ls -A $HOME/.ssh 2>/dev/null)" ]; then
        COPY_SSH="--build-arg COPY_SSH=true"
        echo "SSH keys found, copying to build context"
        cp -r "$HOME/.ssh" ".build-keys-temp/.ssh"
        echo "SSH keys copied"
    else
        echo "No SSH keys found, skipping"
        mkdir -p ".build-keys-temp/.ssh"
    fi
else
    echo "SSH key copying skipped via --skip-ssh"
    mkdir -p ".build-keys-temp/.ssh"
fi

# Build with all options using project root as context
docker build \
    $NO_CACHE \
    $CACHE_FROM \
    $COPY_GPG \
    $COPY_SSH \
    --build-arg BUILDKIT_INLINE_CACHE=1 \
    --build-arg USER_HOME="$HOME" \
    --tag strfkr-base:latest \
    --file docker/base/Dockerfile.base \
    .

echo "✅ Base image built successfully!"

if [ -n "$COPY_GPG" ]; then
    echo "GPG keys included"
else
    echo "❌  No GPG keys in image"
fi

if [ -n "$COPY_SSH" ]; then
    echo "SSH keys included"
else
    echo "❌  No SSH keys in image"
fi

echo ""
echo "Available images:"
docker images | grep strfkr-base

echo ""
echo "To use this base image, rebuild your development containers:"
echo "   docker compose build"