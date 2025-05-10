#!/bin/bash

# Debug script to check API responses

PORT=${1:-8177}
HOST="localhost"

echo "=== BTC Force API Debug ==="
echo "Testing endpoints at http://$HOST:$PORT"
echo ""

echo "1. Testing /health endpoint:"
curl -s "http://$HOST:$PORT/health" | jq . || echo "Failed to get /health"
echo ""

echo "2. Testing /runtime endpoint:"
response=$(curl -s "http://$HOST:$PORT/runtime")
echo "Raw response: $response"
echo "Parsed:"
echo "$response" | jq . || echo "Failed to parse JSON"
echo ""

echo "3. Testing /stats endpoint:"
curl -s "http://$HOST:$PORT/stats" | jq . || echo "Failed to get /stats"
echo ""

echo "4. Testing /workers endpoint:"
curl -s "http://$HOST:$PORT/workers" | jq . || echo "Failed to get /workers"
echo ""

echo "5. Checking if server is running:"
if curl -s --head "http://$HOST:$PORT/health" | head -n 1 | grep "HTTP/1.[01] [23].." > /dev/null; then
    echo "✓ Server is responding"
else
    echo "✗ Server is not responding or not running"
    echo "Make sure btcforce is running on port $PORT"
fi