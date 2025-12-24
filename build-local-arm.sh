#!/bin/bash
# Local ARM image build script for testing on ARM clusters

set -e

# Default values
PLATFORM="${PLATFORM:-linux/arm64}"
TAG="${TAG:-keel:local-arm}"
REGISTRY="${REGISTRY:-}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Building Keel for platform: ${PLATFORM}${NC}"
echo -e "${BLUE}Tag: ${TAG}${NC}"

# Check if docker buildx is available
if ! docker buildx version &> /dev/null; then
    echo "Docker buildx is not available. Please install it first."
    exit 1
fi

# Create a new builder instance if it doesn't exist
if ! docker buildx inspect multiarch &> /dev/null; then
    echo -e "${BLUE}Creating new buildx builder...${NC}"
    docker buildx create --name multiarch --driver docker-container --use
    docker buildx inspect --bootstrap
else
    docker buildx use multiarch
fi

# Build the image
echo -e "${BLUE}Building image...${NC}"
if [ -n "$REGISTRY" ]; then
    # Build and push to registry
    docker buildx build \
        --platform "$PLATFORM" \
        --tag "$REGISTRY/$TAG" \
        --push \
        .
    echo -e "${GREEN}Image pushed to: ${REGISTRY}/${TAG}${NC}"
else
    # Build for local use (single platform only)
    docker buildx build \
        --platform "$PLATFORM" \
        --tag "$TAG" \
        --load \
        .
    echo -e "${GREEN}Image built: ${TAG}${NC}"
    echo -e "${BLUE}To save the image for transfer:${NC}"
    echo "  docker save $TAG | gzip > keel-arm.tar.gz"
    echo -e "${BLUE}To load on your ARM cluster:${NC}"
    echo "  gunzip -c keel-arm.tar.gz | docker load"
fi

echo -e "${GREEN}Build complete!${NC}"
