#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

cleanup() {
  echo ""
  echo "Cleaning up..."
  kill "$AGENT_PID" 2>/dev/null || true
  docker compose -f "$ROOT_DIR/docker-compose.yml" down 2>/dev/null || true
}
trap cleanup EXIT

echo "=== P0: Messaging Pipeline Test ==="
echo ""

echo "1. Starting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" up -d
sleep 2

echo "2. Building agent service..."
go build -o /tmp/constellation-agent "$ROOT_DIR/cmd/agent"

echo "3. Starting agent (constellation-router)..."
/tmp/constellation-agent -model constellation-router -subject "constellation.event.>" &
AGENT_PID=$!
sleep 2

echo "4. Publishing test event via NATS..."
RESPONSE=$(go run "$ROOT_DIR/cmd/agent" -model constellation-router <<'EOF' 2>&1 &
constellation.event.>
EOF
)

# Use nats CLI or go run a tool to publish and receive
EVENT_ID=$(uuidgen 2>/dev/null || echo "test-$(date +%s)")

# Publish a request event and wait for reply using nats request
if command -v nats &>/dev/null; then
  echo "   Publishing request via nats CLI..."
  REPLY=$(echo '{"id":"test","type":"request","source":"p0-test","data":{"prompt":"Say hello in one word"}}' | \
    nats request "constellation.event.request" 2>&1)
  echo "   Reply: $REPLY"
  if echo "$REPLY" | grep -qi "hello"; then
    pass "Agent responded to request"
  else
    fail "Agent did not respond properly"
  fi
else
  echo "   nats CLI not found, skipping publish test"
  fail "nats CLI required for test"
fi

echo ""
echo "5. Testing JetStream durability..."
echo "   Restarting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" restart nats
sleep 3

# Check event survived via stream info
if command -v nats &>/dev/null; then
  STREAM_INFO=$(nats stream info events 2>&1 || true)
  MSG_COUNT=$(echo "$STREAM_INFO" | grep -i "messages" | head -1 | grep -oP '\d+' || echo "0")
  if [ "$MSG_COUNT" -gt 0 ]; then
    pass "Events survived NATS restart ($MSG_COUNT messages)"
  else
    fail "No messages found after restart"
  fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
