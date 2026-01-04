#!/usr/bin/env bash
set -euo pipefail

LOG_FILE="./schemathesis_output.log"
HAR_FILE="./schemathesis_report.har"

# Remove previous logs
rm -f "$LOG_FILE" "$HAR_FILE"

echo "Getting authentication token..."
# First try to signup (this will fail if user exists, which is ok)
curl -s -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' > /dev/null 2>&1 || true

# Now login
TOKEN_RESPONSE=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}')

TOKEN=$(echo "$TOKEN_RESPONSE" | sed -n 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

if [ -z "$TOKEN" ]; then
    echo "Failed to get token. Response was:"
    echo "$TOKEN_RESPONSE"
    exit 1
fi

echo "✓ Got token: ${TOKEN:0:30}..."
echo ""
echo "Clearing Redis cache..."
docker exec toshokan-cache-1 redis-cli FLUSHALL > /dev/null
echo "✓ Redis cache cleared"
echo ""
echo "Starting Schemathesis tests with authentication..."
echo "Log file: $LOG_FILE"
echo ""

docker run --network="host" --rm \
  -e PYTHONUNBUFFERED=1 \
  -v "$PWD:/app" \
  -w /app \
  schemathesis/schemathesis:latest \
  run /app/openapi.yaml \
  --url=http://localhost:8080 \
  -H "Authorization: Bearer $TOKEN" \
  --max-examples=10 \
  --workers=1 \
  --report-har-path=/app/$HAR_FILE \
  2>&1 | sed "s/\[Filtered\]/Bearer $TOKEN/g" | tee "$LOG_FILE"

echo ""
echo "============================================================"
echo "Schemathesis test run completed."
echo "HAR report: $HAR_FILE"
echo "Debug log: $LOG_FILE"
echo ""
echo "Checking for hook execution in logs..."
if grep -q "\[HOOKS\]" "$LOG_FILE"; then
    echo "✓ Hooks were loaded and executed"
else
    echo "✗ No hooks detected in log"
fi
echo "============================================================"
