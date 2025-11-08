#!/bin/bash

# Test script for SSE-based log streaming

BASE_URL="http://localhost:8087"
API_URL="$BASE_URL/api"

echo "=== Testing SSE Log Streaming ==="
echo

# Step 1: Login to get session cookie
echo "1. Logging in..."
LOGIN_RESPONSE=$(curl -s -c /tmp/chariot-cookies.txt -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}')
echo "Login response: $LOGIN_RESPONSE"
echo

# Step 2: Start async execution
echo "2. Starting async execution..."
EXEC_RESPONSE=$(curl -s -b /tmp/chariot-cookies.txt -X POST "$API_URL/execute-async" \
  -H "Content-Type: application/json" \
  -d '{
    "program": "logPrint(\"Starting test\"); let x = 1 + 2; logPrint(\"Result is: \" + x); logPrint(\"Test complete\");"
  }')
echo "Execution response: $EXEC_RESPONSE"

# Extract execution ID
EXEC_ID=$(echo $EXEC_RESPONSE | grep -o '"execution_id":"[^"]*"' | cut -d'"' -f4)
echo "Execution ID: $EXEC_ID"
echo

if [ -z "$EXEC_ID" ]; then
  echo "ERROR: Failed to get execution ID"
  exit 1
fi

# Step 3: Stream logs via SSE
echo "3. Streaming logs (SSE)..."
echo "Press Ctrl+C to stop streaming"
echo "---"
curl -N -b /tmp/chariot-cookies.txt "$API_URL/logs/$EXEC_ID" &
STREAM_PID=$!

# Wait a bit for streaming to complete
sleep 5
kill $STREAM_PID 2>/dev/null

echo
echo "---"

# Step 4: Get final result
echo "4. Getting final result..."
RESULT=$(curl -s -b /tmp/chariot-cookies.txt "$API_URL/result/$EXEC_ID")
echo "Result: $RESULT"
echo

# Cleanup
rm -f /tmp/chariot-cookies.txt

echo "=== Test Complete ==="
