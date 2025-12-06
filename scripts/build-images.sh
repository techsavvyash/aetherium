#!/bin/bash
# =============================================================================
# Aetherium Docker Image Build Script
# =============================================================================
# Usage:
#   ./scripts/build-images.sh [tag] [push]
#
# Examples:
#   ./scripts/build-images.sh                    # Build with 'latest' tag
#   ./scripts/build-images.sh v1.0.0             # Build with specific tag
#   ./scripts/build-images.sh v1.0.0 push        # Build and push to registry
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REGISTRY="${REGISTRY:-ghcr.io/techsavvyash}"
TAG="${1:-latest}"
PUSH="${2:-}"

# Images to build
IMAGES=(
    "api-gateway:docker/Dockerfile.api-gateway"
    "worker:docker/Dockerfile.worker"
    "fc-agent:docker/Dockerfile.fc-agent"
    "cli:docker/Dockerfile.cli"
)

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if docker is available
if ! command -v docker &> /dev/null; then
    error "Docker is not installed"
fi

log "Building Aetherium images with tag: $TAG"
log "Registry: $REGISTRY"
echo ""

# Build each image
for IMAGE_DEF in "${IMAGES[@]}"; do
    IMAGE_NAME="${IMAGE_DEF%%:*}"
    DOCKERFILE="${IMAGE_DEF##*:}"

    FULL_IMAGE="$REGISTRY/aetherium/$IMAGE_NAME:$TAG"

    log "Building $FULL_IMAGE..."

    docker build \
        -t "$FULL_IMAGE" \
        -f "$DOCKERFILE" \
        --build-arg VERSION="$TAG" \
        --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        .

    success "Built $FULL_IMAGE"

    # Also tag as latest if not already
    if [ "$TAG" != "latest" ]; then
        docker tag "$FULL_IMAGE" "$REGISTRY/aetherium/$IMAGE_NAME:latest"
    fi
done

# Push if requested
if [ "$PUSH" = "push" ]; then
    log "Pushing images to $REGISTRY..."

    for IMAGE_DEF in "${IMAGES[@]}"; do
        IMAGE_NAME="${IMAGE_DEF%%:*}"
        FULL_IMAGE="$REGISTRY/aetherium/$IMAGE_NAME:$TAG"

        log "Pushing $FULL_IMAGE..."
        docker push "$FULL_IMAGE"

        if [ "$TAG" != "latest" ]; then
            docker push "$REGISTRY/aetherium/$IMAGE_NAME:latest"
        fi

        success "Pushed $FULL_IMAGE"
    done
fi

echo ""
success "All images built successfully!"
echo ""
echo "Images:"
for IMAGE_DEF in "${IMAGES[@]}"; do
    IMAGE_NAME="${IMAGE_DEF%%:*}"
    echo "  - $REGISTRY/aetherium/$IMAGE_NAME:$TAG"
done
