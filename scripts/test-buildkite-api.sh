#!/bin/bash
# Test BuildKite API connectivity
# Usage: ./scripts/test-buildkite-api.sh

set -e

echo "🔧 BuildKite API Test"
echo "====================="
echo ""

# Check if token and org are provided
if [ -z "$BUILDKITE_TOKEN" ]; then
    echo "❌ BUILDKITE_TOKEN not set"
    echo ""
    echo "To use this script:"
    echo "1. Go to https://buildkite.com/user/api-access-tokens"
    echo "2. Click 'New API Token'"
    echo "3. Select scopes: read_builds, read_organizations, read_pipelines"
    echo "4. Copy the token"
    echo "5. Run: export BUILDKITE_TOKEN='your_token_here'"
    echo "6. Run: export BUILDKITE_ORG='your_org_slug'"
    echo "7. Run this script again"
    exit 1
fi

if [ -z "$BUILDKITE_ORG" ]; then
    echo "❌ BUILDKITE_ORG not set"
    echo ""
    echo "Your org slug is in your BuildKite URL:"
    echo "https://buildkite.com/{org_slug}/pipelines"
    echo ""
    echo "Run: export BUILDKITE_ORG='your_org_slug'"
    exit 1
fi

echo "Configuration:"
echo "  Org: $BUILDKITE_ORG"
echo "  Token: ${BUILDKITE_TOKEN:0:20}... (truncated)"
echo ""

# Test 1: Fetch recent builds
echo "Test 1: Fetching recent builds..."
response=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $BUILDKITE_TOKEN" \
    -H "Accept: application/json" \
    "https://api.buildkite.com/v2/organizations/$BUILDKITE_ORG/builds?per_page=5" \
    2>&1 || echo "000")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    echo "  ✅ SUCCESS (200 OK)"
    echo ""
    echo "Sample builds:"
    echo "$body" | jq -r '.[] | "  - [\(.state)] \(.pipeline.name) #\(.number) (\(.branch))"' 2>/dev/null || echo "$body" | head -c 500
    echo ""
else
    echo "  ❌ FAILED (HTTP $http_code)"
    echo "  Response: $body"
    exit 1
fi

# Test 2: Check for deployment pipelines
echo "Test 2: Looking for deployment pipelines..."
deployment_count=$(echo "$body" | jq '[.[] | select(.pipeline.slug | test("deploy"))] | length' 2>/dev/null || echo "0")

if [ "$deployment_count" -gt "0" ]; then
    echo "  ✅ Found $deployment_count builds with 'deploy' in pipeline name"
    echo "$body" | jq -r '.[] | select(.pipeline.slug | test("deploy")) | "  - \(.pipeline.slug) #\(.number) [\(.state)]"' 2>/dev/null
else
    echo "  ⚠️  No builds found with 'deploy' in pipeline name"
    echo ""
    echo "Available pipelines in recent builds:"
    echo "$body" | jq -r '.[] | .pipeline.slug' 2>/dev/null | sort -u | sed 's/^/  - /'
    echo ""
    echo "You may need to customize the isDeploymentPipeline() function in buildkite.go"
    echo "to match your pipeline naming conventions."
fi

echo ""
echo "✅ BuildKite API is working!"
echo ""
echo "Next steps:"
echo "1. Add to your .env file:"
echo "   BUILDKITE_TOKEN=$BUILDKITE_TOKEN"
echo "   BUILDKITE_ORG=$BUILDKITE_ORG"
echo ""
echo "2. Customize buildkite.go line ~130 (isDeploymentPipeline) if needed"
echo "3. Rebuild: make build"
echo "4. Run: make run"
echo "5. Test endpoints:"
echo "   curl http://localhost:8082/api/kpi/buildkite-deployment-time"
echo "   curl http://localhost:8082/api/kpi/buildkite-deployment-failure-rate"
