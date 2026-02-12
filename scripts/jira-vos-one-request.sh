#!/usr/bin/env bash
# Run one JIRA search request for the VOS KPI and print total + count returned.
# Use this to validate the API and confirm JIRA returns at most 100 per request.
# Requires: .env with JIRA_DOMAIN, JIRA_EMAIL, JIRA_API_TOKEN (from repo root).
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

if [ -f .env ]; then
  set -a
  # shellcheck source=/dev/null
  source <(grep -E '^JIRA_' .env | sed 's/ *=[[:space:]]*/=/' )
  set +a
fi

for v in JIRA_DOMAIN JIRA_EMAIL JIRA_API_TOKEN; do
  [ -n "${!v}" ] || { echo "Missing $v in .env"; exit 1; }
done

# Same JQL as kpi.go (VOS tickets, created in last 1460 days)
JQL="(project in (10525) AND 'issue' in portfolioChildIssuesOf(VBUILD-8121) and assignee in membersOf(\"okta-team-vos_si\") and assignee != 712020:ac68ce71-95a7-4864-8b38-b1f1af717a20) AND created >= -1460d"
ENCODED_JQL=$(python3 -c "import urllib.parse; print(urllib.parse.quote('''$JQL'''))")
URL="https://${JIRA_DOMAIN}.atlassian.net/rest/api/3/search/jql?jql=${ENCODED_JQL}&maxResults=100&startAt=0&fields=created,resolutiondate"
AUTH=$(echo -n "${JIRA_EMAIL}:${JIRA_API_TOKEN}" | base64)

echo "Request: GET search/jql (maxResults=100, startAt=0)"
RESP=$(curl -s -w "\n%{http_code}" -H "Accept: application/json" -H "Content-Type: application/json" -H "Authorization: Basic $AUTH" "$URL")
BODY=$(echo "$RESP" | head -n -1)
CODE=$(echo "$RESP" | tail -n 1)
echo "HTTP status: $CODE"
if [ "$CODE" != "200" ]; then
  echo "$BODY" | head -c 500
  exit 1
fi
TOTAL=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total','?'))" 2>/dev/null || echo "?")
COUNT_ISSUES=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); v=d.get('issues') or d.get('values') or []; print(len(v))" 2>/dev/null || echo "?")
echo "total (from API): $TOTAL"
echo "issues in this response: $COUNT_ISSUES"
if [[ "$TOTAL" =~ ^[0-9]+$ ]]; then
  PAGES=$(( (TOTAL + 99) / 100 ))
  echo "So we need $PAGES request(s) to fetch all (with 2s delay between pages to avoid 429)."
fi
