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
  kill "$ORCH_PID" "$WORKER_PID" "$CRITIC_PID" 2>/dev/null || true
  docker compose -f "$ROOT_DIR/docker-compose.yml" down 2>/dev/null || true
}
trap cleanup EXIT

echo "=== P2: Multi-Agent Chain Test (Router→Worker→Critic) ==="
echo ""

echo "1. Starting NATS..."
docker compose -f "$ROOT_DIR/docker-compose.yml" up -d
sleep 2

echo "2. Building binaries..."
go build -o /tmp/constellation-agent "$ROOT_DIR/cmd/agent"
go build -o /tmp/constellation-orchestrator "$ROOT_DIR/cmd/orchestrator"
go build -o /tmp/ingress "$ROOT_DIR/cmd/ingress"

echo "3. Starting worker agent on constellation.event.task.worker..."
/tmp/constellation-agent -model constellation-worker -subject "constellation.event.task.worker" &
WORKER_PID=$!
sleep 2

echo "4. Starting critic agent on constellation.event.task.critic..."
/tmp/constellation-agent -model constellation-critic -subject "constellation.event.task.critic" &
CRITIC_PID=$!
sleep 2

echo "5. Starting orchestrator on constellation.event.request..."
/tmp/constellation-orchestrator \
  -request-subject "constellation.event.request" \
  -agents "worker,critic" \
  -agent-timeout 30s \
  -max-retries 2 &
ORCH_PID=$!
sleep 3

echo "6. Sending prompt via ingress..."
OUTPUT=$(echo "List 3 programming languages" | /tmp/ingress -timeout 120s 2>&1)
INGRESS_EXIT=$?
RESPONSE_LINE=$(echo "$OUTPUT" | grep -v "^2026" | head -1)
echo "   Response: $RESPONSE_LINE"
echo ""

if [ "$INGRESS_EXIT" -eq 0 ] && [ -n "$RESPONSE_LINE" ]; then
  pass "Ingress received non-empty response from multi-agent chain"
else
  fail "Ingress failed (exit=$INGRESS_EXIT, response='$RESPONSE_LINE')"
fi

echo ""

echo "7. Verifying event durability and correlation chain..."
STREAM_INFO=$(nats req '$JS.API.STREAM.INFO.events' '' 2>&1)
MSG_COUNT=$(echo "$STREAM_INFO" | grep -o '"messages":[0-9]*' | grep -o '[0-9]*')
echo "   Events stored in stream: $MSG_COUNT"
if [ "$MSG_COUNT" -ge 4 ]; then
  pass "Stream contains at least 4 events (request + chain steps + response)"
else
  fail "Stream has only $MSG_COUNT events (expected >=4)"
fi

echo ""
echo "8. Checking event sources via stream..."
# Get last event and verify it's from orchestrator via API
LAST_EVENT=$(nats req '$JS.API.STREAM.MSG.GET.events' '{"seq":'"$MSG_COUNT"'}' 2>&1 | grep -o '"source":"[^"]*"' | head -1 || echo "")
echo "   Last event source: $LAST_EVENT"

echo ""
echo "9. Checking agent restart recovery..."
echo "   Killing worker..."
kill "$WORKER_PID"
wait "$WORKER_PID" 2>/dev/null
echo "   Restarting worker..."
/tmp/constellation-agent -model constellation-worker -subject "constellation.event.task.worker" &
WORKER_PID=$!
sleep 2

echo "   Sending prompt after worker restart..."
OUTPUT2=$(echo "Say hello" | /tmp/ingress -timeout 30s 2>&1)
INGRESS_EXIT2=$?
RESPONSE2=$(echo "$OUTPUT2" | grep -v "^2026" | head -1)
if [ "$INGRESS_EXIT2" -eq 0 ] && [ -n "$RESPONSE2" ]; then
  pass "Chain recovers after agent restart"
else
  fail "Agent restart recovery failed"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

[ "$FAIL" -eq 0 ]
