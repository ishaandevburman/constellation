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

echo "=== P1: Single-Agent Runtime Test ==="
echo ""

echo "1. Starting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" up -d
sleep 2

echo "2. Building binaries..."
go build -o /tmp/constellation-agent "$ROOT_DIR/cmd/agent"
go build -o /tmp/ingress "$ROOT_DIR/cmd/ingress"

echo "3. Starting worker agent..."
/tmp/constellation-agent -model constellation-worker -subject "constellation.event.request" &
AGENT_PID=$!
sleep 3

echo "4. Sending prompt via ingress..."
OUTPUT=$(echo "Say hello in one word" | /tmp/ingress -timeout 30s 2>&1)
INGRESS_EXIT=$?
echo "   Response: $OUTPUT"
echo ""

if [ "$INGRESS_EXIT" -eq 0 ] && [ -n "$OUTPUT" ]; then
  pass "Ingress received non-empty response from worker"
else
  fail "Ingress failed (exit=$INGRESS_EXIT, output='$OUTPUT')"
fi

echo ""
echo "5. Testing agent restart recovery..."
echo "   Killing agent..."
kill "$AGENT_PID"
wait "$AGENT_PID" 2>/dev/null

echo "   Restarting agent..."
/tmp/constellation-agent -model constellation-worker -subject "constellation.event.request" &
AGENT_PID=$!
sleep 3

echo "   Sending prompt after agent restart..."
OUTPUT2=$(echo "Hello again" | /tmp/ingress -timeout 30s 2>&1)
INGRESS_EXIT2=$?
echo "   Response: $OUTPUT2"

if [ "$INGRESS_EXIT2" -eq 0 ] && [ -n "$OUTPUT2" ]; then
  pass "Ingress works after agent restart"
else
  fail "Ingress failed after agent restart (exit=$INGRESS_EXIT2)"
fi

echo ""
echo "6. Testing NATS restart survival..."
echo "   Restarting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" restart nats
sleep 3

echo "   Sending prompt after NATS restart..."
OUTPUT3=$(echo "Hello once more" | /tmp/ingress -timeout 30s 2>&1)
INGRESS_EXIT3=$?
echo "   Response: $OUTPUT3"

if [ "$INGRESS_EXIT3" -eq 0 ] && [ -n "$OUTPUT3" ]; then
  pass "Ingress works after NATS restart"
else
  fail "Ingress failed after NATS restart (exit=$INGRESS_EXIT3)"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

[ "$FAIL" -eq 0 ]
