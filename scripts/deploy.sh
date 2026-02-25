#!/bin/bash
set -euo pipefail

# Deploy ATLinks to Synology NAS
# Usage: ./scripts/deploy.sh [options]
#
# Options:
#   --skip-build    Skip Docker build (redeploy existing image)
#   --logs          Tail logs after deploy
#   --stop          Stop services and exit
#   --status        Show container status and exit
#
# Prerequisites:
#   - SSH access to Synology NAS (key-based auth recommended)
#   - Docker installed on NAS
#   - .env.prod file at deploy path on NAS with DATABASE_URL, JWT_SECRET, TUNNEL_TOKEN
#   - PostgreSQL running on NAS with atlinks database created

# --- Config ---
SYNOLOGY_HOST="192.168.23.44"
SSH_PORT="2222"
DEPLOY_PATH="/volume1/docker/atlinks"
REGISTRY="192.168.23.44:5050"
IMAGE_NAME="$REGISTRY/atlinks:latest"
DOCKER="sudo /usr/local/bin/docker"

SSH_CMD="ssh -p $SSH_PORT brady@$SYNOLOGY_HOST"
SCP_CMD="scp -O -P $SSH_PORT"

# --- Parse args ---
SKIP_BUILD=false
TAIL_LOGS=false
ACTION="deploy"

for arg in "$@"; do
    case $arg in
        --skip-build) SKIP_BUILD=true ;;
        --logs)       TAIL_LOGS=true ;;
        --stop)       ACTION="stop" ;;
        --status)     ACTION="status" ;;
    esac
done

# --- Commands ---
remote() {
    $SSH_CMD "$@"
}

remote_docker() {
    $SSH_CMD "cd $DEPLOY_PATH && $DOCKER $*"
}

remote_compose() {
    $SSH_CMD "cd $DEPLOY_PATH && $DOCKER compose --env-file .env.prod $*"
}

# --- Actions ---
case $ACTION in
    stop)
        echo "==> Stopping services..."
        remote_compose "down"
        echo "Services stopped."
        exit 0
        ;;
    status)
        echo "==> Container status:"
        remote "$DOCKER ps --filter name=atlinks --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
        echo ""
        echo "==> Recent logs:"
        remote_compose "logs --tail 10"
        exit 0
        ;;
esac

# --- Deploy ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

if [ "$SKIP_BUILD" = false ]; then
    BUILD_VERSION=$(git -C "$PROJECT_DIR" rev-parse --short HEAD 2>/dev/null || date +%s)
    echo "==> Building Docker image (version: $BUILD_VERSION)..."
    docker build --build-arg "BUILD_VERSION=$BUILD_VERSION" -t "$IMAGE_NAME" "$PROJECT_DIR"

    echo "==> Pushing image to registry..."
    docker push "$IMAGE_NAME"
else
    echo "==> Skipping build (--skip-build)"
fi

echo "==> Uploading compose file..."
$SCP_CMD "$PROJECT_DIR/docker-compose.prod.yml" "brady@$SYNOLOGY_HOST:$DEPLOY_PATH/docker-compose.yml"

echo "==> Pulling latest image on NAS..."
remote "$DOCKER pull $IMAGE_NAME"

echo "==> Restarting services..."
remote_compose "down"
remote_compose "up -d"

echo ""
echo "==> Container status:"
remote "$DOCKER ps --filter name=atlinks --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

echo ""
echo "==> Startup logs:"
sleep 3
remote_compose "logs --tail 20"

if [ "$TAIL_LOGS" = true ]; then
    echo ""
    echo "==> Tailing logs (Ctrl+C to stop)..."
    remote_compose "logs -f"
fi

echo ""
echo "Deploy complete."
