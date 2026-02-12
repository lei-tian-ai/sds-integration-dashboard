#!/bin/bash
# Test Neuron API connectivity and response format
# Usage: ./scripts/test-neuron-api.sh

set -e

echo "🔍 Neuron API Discovery Helper"
echo "================================"
echo ""

# Check if token is provided
if [ -z "$NEURON_API_TOKEN" ]; then
    echo "❌ NEURON_API_TOKEN not set"
    echo ""
    echo "To use this script:"
    echo "1. Open https://neuron.oci.applied.dev/validation_toolset/insights/dashboard/..."
    echo "2. Open DevTools (F12) → Network tab → Fetch/XHR"
    echo "3. Refresh the page and find API requests"
    echo "4. Copy the Authorization token from Request Headers"
    echo "5. Run: export NEURON_API_TOKEN='your_token_here'"
    echo "6. Run this script again"
    exit 1
fi

# Default values
BASE_URL="${NEURON_API_URL:-https://neuron.oci.applied.dev}"
PROJECT="${NEURON_PROJECT:-Default}"
WORKSPACE="${NEURON_WORKSPACE:-}"

echo "Configuration:"
echo "  Base URL: $BASE_URL"
echo "  Project: $PROJECT"
echo "  Workspace: $WORKSPACE"
echo "  Token: ${NEURON_API_TOKEN:0:20}... (truncated)"
echo ""

# Common API paths to try
API_PATHS=(
    "/api/v1/metrics"
    "/api/v1/vehicles"
    "/api/v1/vehicle-metrics"
    "/api/metrics/vehicle-hours"
    "/api/vehicles/operation-hours"
    "/api/telemetry/metrics"
    "/validation_toolset/api/metrics"
    "/validation_toolset/api/vehicles"
    "/api/v1/insights/metrics"
    "/graphql"
)

echo "Testing common API paths..."
echo ""

for path in "${API_PATHS[@]}"; do
    url="$BASE_URL$path?project=$PROJECT"

    echo "Testing: $url"

    # Try Bearer token
    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $NEURON_API_TOKEN" \
        -H "Accept: application/json" \
        "$url" 2>&1 || echo "000")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$http_code" = "200" ]; then
        echo "  ✅ SUCCESS (200 OK)"
        echo "  Response preview:"
        echo "$body" | head -c 500
        echo ""
        echo "  Full response saved to: neuron-response-${path//\//-}.json"
        echo "$body" | jq '.' > "neuron-response-${path//\//-}.json" 2>/dev/null || echo "$body" > "neuron-response-${path//\//-}.json"
        echo ""
        echo "🎉 Found working endpoint: $url"
        echo ""
        echo "Next steps:"
        echo "1. Check the response file: neuron-response-${path//\//-}.json"
        echo "2. Update neuron.go line 37 with: path := \"$path\""
        echo "3. Update the response parsing code to match the JSON structure"
        exit 0
    elif [ "$http_code" = "401" ] || [ "$http_code" = "403" ]; then
        echo "  ❌ Auth failed ($http_code) - token may be invalid"
    elif [ "$http_code" = "404" ]; then
        echo "  ⚠️  Not found ($http_code)"
    else
        echo "  ⚠️  Status: $http_code"
    fi
    echo ""
done

echo "❌ No working endpoints found from common paths."
echo ""
echo "Manual discovery required:"
echo "1. Open Neuron dashboard in browser"
echo "2. DevTools (F12) → Network tab → Fetch/XHR"
echo "3. Refresh and find the request that loads vehicle data"
echo "4. Copy the request URL and try:"
echo "   curl -H \"Authorization: Bearer \$NEURON_API_TOKEN\" \"THE_URL_YOU_FOUND\""
echo ""
echo "Or ask the Neuron team for API documentation."
