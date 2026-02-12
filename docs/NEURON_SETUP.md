# Quick Start: Getting Neuron Data for MTBF

## Your Dashboard URL
```
https://neuron.oci.applied.dev/validation_toolset/insights/dashboard/24d446a9-e92d-5e65-a1d9-547d9d4414a1?project=Default&workspace=Adriel%27s
```

## Step-by-Step: Find the API

### 1. Open DevTools and Capture API Calls (5 minutes)

1. Open the Neuron dashboard in Chrome
2. Press **F12** to open DevTools
3. Click the **Network** tab
4. Click **Fetch/XHR** filter button
5. **Refresh the page** (Cmd+R / Ctrl+R)
6. Look at all the requests that appear

### 2. Identify the Data Request

Look for requests that:
- Return JSON data (click on them and check the "Response" tab)
- Contain vehicle data or metrics
- Have URLs like:
  - `*/api/*`
  - `*/metrics/*`
  - `*/vehicles/*`
  - `*/telemetry/*`
  - `*/graphql*` (if using GraphQL)

### 3. Copy Authentication Details

For the request that looks like it returns vehicle data:

1. Click on the request in DevTools
2. Go to the **Headers** tab
3. Scroll to **Request Headers** section
4. Look for and copy:
   - `Authorization: Bearer eyJ...` (copy the token after "Bearer ")
   - OR `Cookie: session=...` (copy the cookie value)
   - OR `X-API-Key: ...` (copy the key)

### 4. Copy the Full Request

In the same Headers tab:
- **Request URL**: Copy the full URL
- **Request Method**: Note if it's GET, POST, etc.
- **Query String Parameters**: Note any parameters

### 5. Test with curl

Replace `YOUR_TOKEN` with what you copied:

```bash
# If using Bearer token:
curl -H "Authorization: Bearer YOUR_TOKEN" \
     -H "Accept: application/json" \
     "https://neuron.oci.applied.dev/THE_API_PATH_YOU_FOUND"

# If using Cookie:
curl -H "Cookie: session=YOUR_SESSION_COOKIE" \
     -H "Accept: application/json" \
     "https://neuron.oci.applied.dev/THE_API_PATH_YOU_FOUND"
```

If successful, you'll see JSON data. Copy this output.

## Step 6: Update the Code

Once you have working API details:

### A. Update `neuron.go`

```go
// Line 11: Update the base URL if different
const neuronBaseURL = "https://neuron.oci.applied.dev"

// Line 37: Replace with actual API path
path := "/api/v1/metrics/vehicle-hours" // Replace this

// Line 67: Update auth header if needed
req.Header.Set("Authorization", "Bearer "+token) // Or use Cookie, X-API-Key, etc.
```

### B. Add to `main.go`

```go
// After line 43 in main.go, add:
api.GET("/neuron/vehicle-hours", neuronVehicleHours)
```

### C. Add to `.env`

```env
NEURON_API_TOKEN=your_token_here
```

### D. Test the endpoint

```bash
# Rebuild
make build

# Run
make run

# Test in another terminal
curl http://localhost:8082/api/neuron/vehicle-hours?project=Default
```

## Alternative: Ask Internal Team

Before spending too much time reverse-engineering, try:

1. **Slack/email** the Neuron or Validation Toolset team
2. Ask: "Is there API documentation for the Insights dashboard? I need to fetch vehicle operation hours programmatically for KPI tracking."
3. They may have:
   - API docs
   - Service account credentials
   - Better endpoints for bulk data access

## What Data We Need

For MTBF metric, we need:
- **Vehicle operation hours** (or drive hours) per vehicle
- **Date range**: last 3 months
- **Aggregation**: By calendar week
- **Format**: Total hours per week across all vehicles

Example ideal response:
```json
{
  "weeks": [
    {
      "week": "2025-W05",
      "total_operation_hours": 1234.5,
      "vehicles": [
        {"id": "MCE-203", "hours": 456.2},
        {"id": "ROG-131", "hours": 778.3}
      ]
    }
  ]
}
```

## Next Steps After Getting Data

1. Update `kpiMTBF` in `kpi.go` to fetch Neuron hours
2. Match hours with failure counts by week
3. Calculate: `mtbf = hours / failures`
4. Update frontend Dashboard.tsx to display MTBF ratio instead of just failures
5. Update About page to mark MTBF as complete

## Need Help?

See `docs/neuron-api-discovery.md` for detailed guidance.
