# Neuron API Discovery Guide

## Goal
Extract vehicle operation hours data from Neuron dashboard at:
https://neuron.oci.applied.dev/validation_toolset/insights/dashboard/24d446a9-e92d-5e65-a1d9-547d9d4414a1

## Step 1: Discover API Endpoints

### Using Browser DevTools:

1. **Open the Neuron dashboard** in Chrome/Firefox
2. **Open DevTools** (F12 or Right-click → Inspect)
3. **Go to Network tab**
4. **Filter by "Fetch/XHR"** to see API calls
5. **Refresh the page** or interact with the dashboard
6. **Look for API calls** that return data (usually JSON responses)

### What to look for:

Common patterns for internal APIs:
- `/api/v1/...`
- `/api/metrics/...`
- `/api/dashboard/...`
- `/graphql` (if using GraphQL)
- Endpoints containing: `vehicle`, `hours`, `meter`, `telemetry`, `metrics`

### Capture the following for each API call:

- **URL**: Full request URL
- **Method**: GET, POST, etc.
- **Headers**: Look for authentication headers
  - `Authorization: Bearer <token>`
  - `Cookie: session=...`
  - `X-API-Key: ...`
  - Custom auth headers
- **Query parameters**: `?project=...&workspace=...`
- **Request body**: If POST/PUT
- **Response format**: JSON structure

## Step 2: Find Authentication Method

### Check for API tokens:

1. **In DevTools → Application → Cookies**
   - Look for session cookies
   - Copy cookie name and value

2. **In DevTools → Network → Request Headers**
   - Look for `Authorization` header
   - Look for custom auth headers

3. **Check local storage**
   - DevTools → Application → Local Storage
   - Look for tokens or API keys

### Common auth patterns at Applied:

- **OAuth/SSO tokens** (likely Okta-based)
- **API keys** in headers
- **Session cookies** from login
- **Service account tokens** (if available)

## Step 3: Test API Access

### Using curl:

```bash
# With Bearer token
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "https://neuron.oci.applied.dev/api/metrics/vehicle-hours?project=Default"

# With cookies
curl -b "session=YOUR_SESSION_COOKIE" \
     "https://neuron.oci.applied.dev/api/..."

# With API key
curl -H "X-API-Key: YOUR_KEY" \
     "https://neuron.oci.applied.dev/api/..."
```

## Step 4: Document the API

Once you find working endpoints, document:

### Example:

```
Endpoint: GET /api/v1/vehicles/metrics
Base URL: https://neuron.oci.applied.dev
Auth: Bearer token (from Okta SSO)

Query params:
  - project: string (e.g., "Default")
  - workspace: string (e.g., "Adriel's")
  - start_date: ISO 8601 date
  - end_date: ISO 8601 date
  - metric: string (e.g., "operation_hours", "drive_hours")

Response:
{
  "vehicles": [
    {
      "id": "...",
      "name": "...",
      "metrics": {
        "operation_hours": 123.5,
        "date": "2025-02-01"
      }
    }
  ]
}
```

## Step 5: Integrate into App

Once you have the API details, add to the backend:

### Option A: Direct API calls (if you have API token)

1. Add env vars:
   ```
   NEURON_API_URL=https://neuron.oci.applied.dev
   NEURON_API_TOKEN=your_token_here
   ```

2. Create `neuron.go`:
   ```go
   func neuronConfig() (baseURL, token string, ok bool) {
       baseURL = os.Getenv("NEURON_API_URL")
       token = os.Getenv("NEURON_API_TOKEN")
       return baseURL, token, baseURL != "" && token != ""
   }

   func neuronVehicleHours(c *gin.Context) {
       // Similar to fleetioVehicles handler
   }
   ```

### Option B: OAuth flow (if using SSO)

If Neuron uses Okta SSO, you'll need:
1. Service account credentials
2. OAuth client ID/secret
3. Token refresh logic

## Alternative: Check for Internal Documentation

Before reverse-engineering, check if documentation exists:

1. **Internal wiki/docs**: Search for "Neuron API", "Validation Toolset API"
2. **Swagger/OpenAPI**: Try accessing:
   - `https://neuron.oci.applied.dev/swagger`
   - `https://neuron.oci.applied.dev/docs`
   - `https://neuron.oci.applied.dev/api-docs`
3. **Ask team**: Reach out to Neuron/validation toolset team for API access

## Security Considerations

- **Never commit tokens** to git
- Use environment variables for all credentials
- Consider using service accounts instead of personal tokens
- Check if you need permission to access the API programmatically

## Next Steps

1. [ ] Open Neuron dashboard and capture API calls in DevTools
2. [ ] Identify authentication method
3. [ ] Find endpoint that returns vehicle operation hours
4. [ ] Test with curl
5. [ ] Document API response format
6. [ ] Add backend handler in `neuron.go`
7. [ ] Update MTBF KPI to use Neuron data
