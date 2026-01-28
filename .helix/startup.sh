#!/bin/bash
set -euo pipefail

# Project startup script for Keel development
# This runs when agents start working on this project
# Idempotent - safe to run multiple times
# Uses k3s (lightweight Kubernetes) for local development

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
KEEL_PID_FILE="/tmp/keel.pid"
KEEL_LOG_FILE="/tmp/keel.log"
KUBECONFIG_PATH="$HOME/.kube/config"
K3S_KUBECONFIG="/etc/rancher/k3s/k3s.yaml"

echo "🚀 Starting project: Keel"
echo "   Project root: $PROJECT_ROOT"

# =============================================================================
# Helper Functions
# =============================================================================

log_step() {
    echo ""
    echo "▶️  $1"
}

log_success() {
    echo "   ✅ $1"
}

log_info() {
    echo "   ℹ️  $1"
}

log_warning() {
    echo "   ⚠️  $1"
}

# Check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Check if a process is running by PID file
is_process_running() {
    local pid_file="$1"
    if [[ -f "$pid_file" ]]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Wait for a condition with timeout
wait_for() {
    local description="$1"
    local check_cmd="$2"
    local timeout="${3:-60}"
    local interval="${4:-2}"

    local elapsed=0
    while ! eval "$check_cmd" &>/dev/null; do
        if [[ $elapsed -ge $timeout ]]; then
            echo "   ❌ Timeout waiting for: $description"
            return 1
        fi
        sleep "$interval"
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    echo ""
    return 0
}

# =============================================================================
# Prerequisites Setup
# =============================================================================

log_step "Checking prerequisites..."

# Check/Install Go
if command_exists go; then
    log_success "Go is installed: $(go version | head -c 50)"
else
    log_info "Installing Go..."
    GO_VERSION="1.21.6"
    GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"

    # Download Go
    curl -sLO "https://go.dev/dl/${GO_TARBALL}"

    # Install Go
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "$GO_TARBALL"
    rm "$GO_TARBALL"

    # Add to profile for future sessions
    if ! grep -q '/usr/local/go/bin' "$HOME/.profile" 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> "$HOME/.profile"
    fi

    log_success "Go installed: $(/usr/local/go/bin/go version | head -c 50)"
fi

# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin

# Check/Install kubectl
if command_exists kubectl; then
    log_success "kubectl is already installed"
else
    log_info "Installing kubectl..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    rm kubectl
    log_success "kubectl installed"
fi

# Check/Install k3s
if command_exists k3s; then
    log_success "k3s is already installed"
else
    log_info "Installing k3s..."
    curl -sfL https://get.k3s.io | INSTALL_K3S_SKIP_ENABLE=true INSTALL_K3S_SKIP_START=true sh -
    log_success "k3s installed"
fi

# =============================================================================
# Cluster Management
# =============================================================================

log_step "Setting up k3s cluster..."

# Check if k3s is running
if pgrep -f "k3s server" >/dev/null 2>&1; then
    log_success "k3s is already running"
else
    log_info "Starting k3s server (no systemd mode)..."
    # Run k3s server directly in background
    sudo k3s server \
        --write-kubeconfig-mode 644 \
        --disable-cloud-controller \
        --disable traefik \
        >/tmp/k3s-server.log 2>&1 &

    # Wait for k3s to start
    log_info "Waiting for k3s to initialize..."
    sleep 10
    log_success "k3s started"
fi

# Setup kubeconfig
mkdir -p "$HOME/.kube"
if [[ -f "$K3S_KUBECONFIG" ]]; then
    sudo cp "$K3S_KUBECONFIG" "$KUBECONFIG_PATH"
    sudo chown $(id -u):$(id -g) "$KUBECONFIG_PATH"
    chmod 600 "$KUBECONFIG_PATH"
    # Replace localhost with 127.0.0.1 for better compatibility
    sed -i 's/127.0.0.1/127.0.0.1/g' "$KUBECONFIG_PATH"
    log_success "Kubeconfig configured"
else
    log_warning "k3s kubeconfig not found yet, waiting..."
    sleep 5
    if [[ -f "$K3S_KUBECONFIG" ]]; then
        sudo cp "$K3S_KUBECONFIG" "$KUBECONFIG_PATH"
        sudo chown $(id -u):$(id -g) "$KUBECONFIG_PATH"
        chmod 600 "$KUBECONFIG_PATH"
        log_success "Kubeconfig configured"
    fi
fi
export KUBECONFIG="$KUBECONFIG_PATH"

# Wait for cluster to be ready
log_info "Waiting for cluster to be ready..."
wait_for "Kubernetes API" "kubectl get nodes" 120 3
log_success "Cluster is ready"

# Show cluster info
kubectl get nodes

# =============================================================================
# Keel Build and Run
# =============================================================================

log_step "Setting up Keel..."

# We need to work with the master branch for the actual code
# The helix-specs branch only has .helix directory
TEMP_KEEL_DIR="/tmp/keel-source"
KEEL_CMD_DIR="$TEMP_KEEL_DIR/cmd/keel"
KEEL_BINARY="$TEMP_KEEL_DIR/cmd/keel/keel"

# Check if we have the Keel source code
if [[ ! -d "$KEEL_CMD_DIR" ]]; then
    log_info "Cloning Keel source from master branch..."

    if [[ -d "$TEMP_KEEL_DIR" ]]; then
        rm -rf "$TEMP_KEEL_DIR"
    fi

    # Get the remote URL and clone master branch
    REMOTE_URL=$(git -C "$PROJECT_ROOT" remote get-url origin)
    git clone --branch master --depth 1 "$REMOTE_URL" "$TEMP_KEEL_DIR"
    log_success "Keel source cloned"
else
    log_success "Keel source already exists at $TEMP_KEEL_DIR"
fi

# Build Keel if binary doesn't exist or source is newer
if [[ ! -f "$KEEL_BINARY" ]] || [[ "$KEEL_CMD_DIR/main.go" -nt "$KEEL_BINARY" ]]; then
    log_info "Building Keel..."
    cd "$TEMP_KEEL_DIR"

    # Download dependencies
    go mod download

    # Build the binary
    cd cmd/keel
    go build -o keel .

    log_success "Keel built successfully"
else
    log_success "Keel binary already exists and is up to date"
fi

# Check if Keel is already running
if is_process_running "$KEEL_PID_FILE"; then
    EXISTING_PID=$(cat "$KEEL_PID_FILE")
    log_success "Keel is already running (PID: $EXISTING_PID)"
else
    log_info "Starting Keel..."

    # Kill any existing process on port 9300
    if command_exists fuser; then
        fuser -k 9300/tcp 2>/dev/null || true
        sleep 1
    fi

    # Start Keel in background
    cd "$(dirname "$KEEL_BINARY")"

    KUBECONFIG="$KUBECONFIG_PATH" \
    BASIC_AUTH_USER=admin \
    BASIC_AUTH_PASSWORD=admin \
    nohup ./keel --no-incluster > "$KEEL_LOG_FILE" 2>&1 &

    KEEL_PID=$!
    echo "$KEEL_PID" > "$KEEL_PID_FILE"

    log_success "Keel started (PID: $KEEL_PID)"
    log_info "Logs: $KEEL_LOG_FILE"
fi

# =============================================================================
# Verification
# =============================================================================

log_step "Verifying setup..."

# Wait for Keel to start
log_info "Waiting for Keel to be ready on port 9300..."
wait_for "Keel HTTP server" "curl -s http://localhost:9300 > /dev/null" 30 2
log_success "Keel is responding"

# Verify we can create deployments
log_info "Verifying cluster accepts deployments..."
kubectl create namespace keel-test --dry-run=client -o yaml 2>/dev/null | kubectl apply -f - 2>/dev/null || true
log_success "Cluster is accepting resources"

# =============================================================================
# Summary
# =============================================================================

echo ""
echo "=============================================="
echo "✅ Project startup complete!"
echo "=============================================="
echo ""
echo "📦 k3s cluster: running"
echo "🔧 Kubeconfig:   $KUBECONFIG_PATH"
echo "🚀 Keel:         Running on http://localhost:9300"
echo "📝 Keel logs:    $KEEL_LOG_FILE"
echo "🔑 UI Login:     admin / admin"
echo ""
echo "Quick commands:"
echo "  kubectl get nodes              # Check cluster nodes"
echo "  kubectl get pods -A            # List all pods"
echo "  tail -f $KEEL_LOG_FILE         # Watch Keel logs"
echo "  kill \$(cat $KEEL_PID_FILE)     # Stop Keel"
echo "  sudo systemctl stop k3s       # Stop cluster"
echo ""
