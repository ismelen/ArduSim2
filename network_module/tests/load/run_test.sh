#!/usr/bin/env bash
#
# run_test.sh — Build and launch the network_module broker load test.
#
# Usage:
#   ./run_test.sh              # Runs with default 100 drones, 30s duration
#   ./run_test.sh 50 20        # 50 drones for 20 seconds
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

NUM_DRONES="${1:-100}"
DURATION="${2:-60}"

echo "============================================"
echo "  Network Module Broker — Load Test"
echo "============================================"
echo "  Drones:   $NUM_DRONES"
echo "  Duration: ${DURATION}s"
echo "============================================"
echo ""

# Cleanup on exit
cleanup() {
    echo ""
    echo ">>> Tearing down containers..."
    docker compose down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

# Build images
echo ">>> Building images..."
docker compose build --quiet

# Launch
echo ">>> Starting broker + $NUM_DRONES drones..."
export TOTAL_DRONES="$NUM_DRONES"
docker compose up --detach

# Wait for drones to start
sleep 2

echo ">>> Broker logs (streaming for ${DURATION}s)..."
echo "--------------------------------------------"
timeout "${DURATION}s" docker compose logs --follow broker 2>/dev/null || true
echo "--------------------------------------------"

echo ""
echo ">>> Broker logs:"
echo "--------------------------------------------"
docker compose logs broker 2>/dev/null || true
echo "--------------------------------------------"

# Stats
echo ""
echo ">>> Container stats:"
docker compose ps --format "table {{.Name}}\t{{.Status}}" 2>/dev/null || true

echo ""
echo ">>> Test complete!"
