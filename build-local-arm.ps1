# Local ARM image build script for testing on ARM clusters (PowerShell version)

param(
    [string]$Platform = "linux/arm64",
    [string]$Tag = "keel:local-arm",
    [string]$Registry = ""
)

$ErrorActionPreference = "Stop"

Write-Host "Building Keel for platform: $Platform" -ForegroundColor Blue
Write-Host "Tag: $Tag" -ForegroundColor Blue

# Check if docker buildx is available
try {
    docker buildx version | Out-Null
} catch {
    Write-Host "Docker buildx is not available. Please install it first." -ForegroundColor Red
    exit 1
}

# Create a new builder instance if it doesn't exist
$builderExists = docker buildx inspect multiarch 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Creating new buildx builder..." -ForegroundColor Blue
    docker buildx create --name multiarch --driver docker-container --use
    docker buildx inspect --bootstrap
} else {
    docker buildx use multiarch
}

# Build the image
Write-Host "Building image..." -ForegroundColor Blue
if ($Registry) {
    # Build and push to registry
    docker buildx build `
        --platform "$Platform" `
        --tag "$Registry/$Tag" `
        --push `
        .
    Write-Host "Image pushed to: $Registry/$Tag" -ForegroundColor Green
} else {
    # Build for local use (single platform only)
    docker buildx build `
        --platform "$Platform" `
        --tag "$Tag" `
        --load `
        .
    Write-Host "Image built: $Tag" -ForegroundColor Green
    Write-Host "To save the image for transfer:" -ForegroundColor Blue
    Write-Host "  docker save $Tag | gzip > keel-arm.tar.gz"
    Write-Host "To load on your ARM cluster:" -ForegroundColor Blue
    Write-Host "  gunzip -c keel-arm.tar.gz | docker load"
}

Write-Host "Build complete!" -ForegroundColor Green
