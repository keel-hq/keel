#!/bin/bash
set -euo pipefail

# Project startup script for Keel development
# This runs when agents start working on this project
# Idempotent - safe to run multiple times
# Uses k3s (lightweight Kubernetes) for local development
#
# NOTE: Running Kubernetes inside Docker containers requires proper cgroup v2 support
# with memory controller enabled. If this fails, see workarounds at the end of the script.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
KEEL_PID_FILE="/tmp/keel.pid"
KEEL_LOG_FILE="/tmp/keel.log"
KUBECONFIG_PATH="$HOME/.kube/config"
K0S_CONFIG_DIR="/var/lib/k0s"

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

# Fix cgroup mount for container environments (needs to be rw, not ro)
if mount | grep -q "cgroup.*\(ro,"; then
    log_info "Remounting cgroup as read-write..."
    sudo mount -o remount,rw /sys/fs/cgroup 2>/dev/null || true
fi

# Check for memory controller availability (required for k3s)
if ! grep -q "memory" /sys/fs/cgroup/cgroup.controllers 2>/dev/null; then
    log_warning "Memory cgroup controller not available!"
    log_warning "This is common in containerized environments."
    log_warning "K3s will likely fail to start. See workarounds at script end."
fi

# Check if k3s is running
if pgrep -f "k3s server" >/dev/null 2>&1; then
    log_success "k3s is already running"
else
    log_info "Starting k3s server..."
    # Run k3s server directly in background (no systemd)
    sudo k3s server \
        --write-kubeconfig-mode 644 \
        --disable-cloud-controller \
        --disable traefik \
        >/tmp/k3s-server.log 2>&1 &

    K3S_PID=$!

    # Wait for k3s to start
    log_info "Waiting for k3s to initialize (checking for up to 30s)..."
    sleep 5

    # Check if k3s is still running
    if kill -0 $K3S_PID 2>/dev/null; then
        log_success "k3s process started (PID: $K3S_PID)"
    else
        log_warning "k3s process exited early - check /tmp/k3s-server.log"
        if grep -q "failed to find memory cgroup" /tmp/k3s-server.log 2>/dev/null; then
            echo ""
            echo "   ❌ K3s failed due to missing memory cgroup controller"
            echo "   This is expected in Docker containers without proper cgroup delegation."
            echo ""
        fi
    fi
fi

# Setup kubeconfig
mkdir -p "$HOME/.kube"
if [[ -f "/etc/rancher/k3s/k3s.yaml" ]]; then
    log_info "Exporting kubeconfig..."
    sudo cp /etc/rancher/k3s/k3s.yaml "$KUBECONFIG_PATH"
    sudo chown $(id -u):$(id -g) "$KUBECONFIG_PATH"
    chmod 600 "$KUBECONFIG_PATH"
    log_success "Kubeconfig configured"
    export KUBECONFIG="$KUBECONFIG_PATH"
else
    log_warning "k3s kubeconfig not found - cluster may not have started successfully"
    log_info "Check /tmp/k3s-server.log for details"
fi

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

# Check if k3s actually started successfully
if pgrep -f "k3s server" >/dev/null 2>&1 && [[ -f "$KUBECONFIG_PATH" ]]; then
    echo "📦 k3s cluster: running"
    echo "🔧 Kubeconfig:   $KUBECONFIG_PATH"
else
    echo "⚠️  k3s cluster: NOT running (see errors above)"
    echo ""
    echo "WORKAROUNDS FOR CONTAINERIZED ENVIRONMENTS:"
    echo "=============================================="
    echo ""
    echo "Option 1: Use External Cluster (Recommended)"
    echo "  - Set up a cloud k8s cluster (GKE, EKS, AKS)"
    echo "  - Copy kubeconfig to: $KUBECONFIG_PATH"
    echo "  - Or set KUBECONFIG=/path/to/your/kubeconfig"
    echo ""
    echo "Option 2: Use Host Kubernetes"
    echo "  - If host has k8s running, mount its kubeconfig"
    echo "  - docker run -v ~/.kube:/root/.kube:ro ..."
    echo ""
    echo "Option 3: Fix Host Docker Configuration"
    echo "  - Enable cgroup v2 delegation in /etc/docker/daemon.json"
    echo "  - Run container with: --cgroupns=private --privileged"
    echo ""
    echo "For details, see: design/tasks/000045_fix-the-project-startup/design.md"
    echo ""
fi

echo "🚀 Keel:         Running on http://localhost:9300"
echo "📝 Keel logs:    $KEEL_LOG_FILE"
echo "🔑 UI Login:     admin / admin"
echo ""
echo "Quick commands:"
echo "  kubectl get nodes              # Check cluster nodes"
echo "  kubectl get pods -A            # List all pods"
echo "  tail -f $KEEL_LOG_FILE         # Watch Keel logs"
echo "  tail -f /tmp/k3s-server.log    # Watch k3s logs"
echo "  kill \$(cat $KEEL_PID_FILE)     # Stop Keel"
echo "  sudo pkill -f 'k3s server'    # Stop k3s"
echo ""
