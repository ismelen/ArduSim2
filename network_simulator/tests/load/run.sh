#!/bin/bash
set -e

# Move to the directory where docker-compose.yaml is located
cd "$(dirname "$0")"

echo "Building Docker images..."
docker compose build

echo "Starting broker..."
docker compose up -d broker

echo "Waiting for broker to initialize (3s)..."
sleep 3

echo "Starting 50 UAV simulations (2 msg each = 100 broadcasts)..."
docker compose up -d --scale uav=50 uav

echo "Waiting for UAVs to finish emitting (approx 7 seconds)..."
sleep 7

echo "Stopping broker..."
docker compose kill -s SIGINT broker
sleep 2

echo ""
echo "==== BROKER LOG SUMMARY ===="
LOGS=$(docker compose logs broker)
echo "$LOGS" | tail -n 35

echo ""
echo "==== TEST VERIFICATION ===="
TOTAL_RCVD=$(echo "$LOGS" | grep -oE 'Total broadcasts received:[ \t]+[0-9]+' | awk '{print $NF}' | tail -n 1)
ERRORS=$(echo "$LOGS" | grep -oE 'UDP send errors:[ \t]+[0-9]+' | awk '{print $NF}' | tail -n 1)

FAILED=0

if [ -z "$TOTAL_RCVD" ]; then
    echo "❌ ERROR: Could not parse 'Total broadcasts received' from log."
    FAILED=1
elif [ "$TOTAL_RCVD" -lt 100 ]; then
    echo "❌ ERROR: Total received ($TOTAL_RCVD) is < expected (100)."
    FAILED=1
else
    echo "✅ Total received broadcasts OK: $TOTAL_RCVD >= 100"
fi

if [ -z "$ERRORS" ]; then
    echo "❌ ERROR: Could not parse 'UDP send errors' from log."
    FAILED=1
elif [ "$ERRORS" -ne 0 ]; then
    echo "❌ ERROR: Expected 0 UDP send errors, but got $ERRORS."
    FAILED=1
else
    echo "✅ No UDP send errors."
fi

echo ""
echo "Cleaning up containers..."
docker compose down

if [ $FAILED -eq 1 ]; then
    echo "🔥 TEST STATUS: FAILED 🔥"
    exit 1
else
    echo "🎉 TEST STATUS: PASSED 🎉"
    exit 0
fi
