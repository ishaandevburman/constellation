#!/usr/bin/env bash
set -uo pipefail

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

# ---- 1. Setup ----
echo "1. Starting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" up -d
sleep 2

echo "2. Building binaries..."
go build -o /tmp/constellation-agent "$ROOT_DIR/cmd/agent"
go build -o /tmp/p0test "$ROOT_DIR/cmd/p0test"

# ---- 2. Pub/Sub Flow ----
echo ""
echo "--- Pub/Sub Flow ---"

/tmp/constellation-agent -model constellation-router -subject "constellation.event.request" &
AGENT_PID=$!
sleep 3

OUTPUT=$(/tmp/p0test -prompt "Say hello in one word" 2>&1)
echo "$OUTPUT"
echo ""

if echo "$OUTPUT" | grep -q "Result:"; then
  pass "Agent responded to request"
else
  fail "Agent did not respond properly"
fi

if echo "$OUTPUT" | grep -q "CorrelationID:"; then
  pass "Correlation tracking works"
else
  fail "Missing correlation ID"
fi

# ---- 3. JetStream Durability ----
echo ""
echo "--- JetStream Durability ---"

echo "Creating test stream..."
nats stream add test-events --subjects "constellation.durability.>" --storage file --replicas 1 --defaults 2>&1 || true

echo "Publishing durability test message..."
nats pub "constellation.durability.test" '{"msg":"hello"}' 2>&1
sleep 1

BEFORE=$(nats stream info test-events 2>&1 | grep -oP 'Messages:\s+\K\d+' || echo "0")
echo "Messages before restart: $BEFORE"

echo "Restarting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" restart nats
sleep 3

AFTER=$(nats stream info test-events 2>&1 | grep -oP 'Messages:\s+\K\d+' || echo "0")
echo "Messages after restart: $AFTER"

if [ "${AFTER:-0}" = "${BEFORE:-0}" ] && [ "${AFTER:-0}" -gt 0 ]; then
  pass "Events survived NATS restart ($AFTER messages)"
else
  fail "Messages lost after restart (before=${BEFORE:-0}, after=${AFTER:-0})"
fi

nats stream rm test-events --force 2>&1 || true

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

[ "$FAIL" -eq 0 ]
