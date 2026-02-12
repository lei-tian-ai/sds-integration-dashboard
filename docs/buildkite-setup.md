# BuildKite API Setup

## Overview

We'll integrate BuildKite to track two deployment KPIs:
1. **Average Deployment Time** - Time from build start to completion
2. **Deployment Failure Rate** - Percentage of failed deployments

## Step 1: Get BuildKite API Token

### Create API Token:

1. Go to https://buildkite.com/user/api-access-tokens
2. Click **New API Token**
3. Set token name: "SDS Integration Dashboard"
4. Select scopes (minimum required):
   - ✅ `read_builds` - Read build data
   - ✅ `read_organizations` - Read org info
   - ✅ `read_pipelines` - Read pipeline info
5. Click **Create Token**
6. **Copy the token** (you won't see it again!)

### Add to .env:

```env
BUILDKITE_TOKEN=your_api_token_here
BUILDKITE_ORG=your_org_slug
```

Your org slug is in your BuildKite URL: `https://buildkite.com/{org_slug}/pipelines`

## Step 2: BuildKite REST API Reference

### Base URL
```
https://api.buildkite.com/v2
```

### Authentication
```bash
Authorization: Bearer YOUR_TOKEN
```

### Key Endpoints

#### 1. List Builds for Organization
```
GET /v2/organizations/{org.slug}/builds
```

**Query Parameters:**
- `created_from` - ISO 8601 date (e.g., "2025-01-01T00:00:00Z")
- `created_to` - ISO 8601 date
- `state[]` - Filter by state: `passed`, `failed`, `canceled`, `running`
- `branch` - Filter by branch (e.g., "main", "production")
- `per_page` - Results per page (default 30, max 100)
- `page` - Page number

**Example Request:**
```bash
curl -H "Authorization: Bearer $BUILDKITE_TOKEN" \
  "https://api.buildkite.com/v2/organizations/applied-intuition/builds?created_from=2025-01-01T00:00:00Z&per_page=100"
```

#### 2. List Builds for Specific Pipeline
```
GET /v2/organizations/{org.slug}/pipelines/{pipeline.slug}/builds
```

**Example:**
```bash
curl -H "Authorization: Bearer $BUILDKITE_TOKEN" \
  "https://api.buildkite.com/v2/organizations/applied-intuition/pipelines/deploy-production/builds?per_page=100"
```

### Build Object Fields

Each build object includes:

```json
{
  "id": "uuid",
  "number": 123,
  "state": "passed",         // passed, failed, canceled, running, scheduled
  "started_at": "2025-02-01T10:00:00.000Z",
  "finished_at": "2025-02-01T10:15:30.000Z",
  "created_at": "2025-02-01T09:59:00.000Z",
  "scheduled_at": "2025-02-01T09:59:30.000Z",
  "pipeline": {
    "slug": "deploy-production",
    "name": "Deploy to Production"
  },
  "branch": "main",
  "commit": "abc123...",
  "message": "Deploy v1.2.3",
  "author": {
    "name": "Jane Doe",
    "email": "jane@applied.co"
  },
  "jobs": [...]
}
```

### Calculate Deployment Time

**Duration = finished_at - started_at**

```javascript
const started = new Date(build.started_at)
const finished = new Date(build.finished_at)
const durationMinutes = (finished - started) / 1000 / 60
```

### Identify Deployment Pipelines

Filter builds by pipeline slug patterns:
- `deploy-*` (e.g., "deploy-production", "deploy-staging")
- Specific pipeline: `deploy-production`
- Branch: `main` or `production`

## Step 3: Rate Limits

BuildKite API rate limits (as of 2024):
- **200 requests per minute** per token
- Rate limit headers in response:
  - `RateLimit-Limit: 200`
  - `RateLimit-Remaining: 195`
  - `RateLimit-Reset: 1612345678`

**Best practice**: Cache results, use pagination, add delay between requests if fetching many pages.

## Step 4: Implementation Plan

### KPI #1: Average Deployment Time

**Formula:**
```
avg_deployment_time = sum(build_durations) / count(builds)
```

**Per week:**
1. Fetch builds for last 3 months with `state[]=passed`
2. Filter by deployment pipeline slugs
3. Calculate duration for each build
4. Group by ISO week (based on `finished_at`)
5. Calculate average per week

### KPI #2: Deployment Failure Rate

**Formula:**
```
failure_rate = failed_builds / total_builds * 100
```

**Per week:**
1. Fetch all builds (passed + failed) for last 3 months
2. Filter by deployment pipeline slugs
3. Group by ISO week (based on `finished_at`)
4. Calculate: `failed / (passed + failed) * 100`

### Backend Implementation

File: `buildkite.go`
- `buildkiteConfig()` - Get token and org from env
- `buildkiteBuilds()` - Fetch builds with pagination
- `kpiBuildkiteDeploymentTime()` - Calculate average deployment time per week
- `kpiBuildkiteDeploymentFailureRate()` - Calculate failure rate per week

### Frontend Implementation

Add two new widgets to `DashboardCompact.tsx`:
1. **Average Deployment Time** - Line chart showing minutes per week
2. **Deployment Failure Rate** - Line chart showing percentage per week

## Step 5: Testing

### Test Authentication:
```bash
export BUILDKITE_TOKEN="your_token"
export BUILDKITE_ORG="your_org_slug"

curl -H "Authorization: Bearer $BUILDKITE_TOKEN" \
  "https://api.buildkite.com/v2/organizations/$BUILDKITE_ORG/builds?per_page=1"
```

Should return JSON with builds array.

### Test with Date Filter:
```bash
curl -H "Authorization: Bearer $BUILDKITE_TOKEN" \
  "https://api.buildkite.com/v2/organizations/$BUILDKITE_ORG/builds?created_from=2025-01-01T00:00:00Z&per_page=5"
```

## Step 6: Deployment Pipeline Detection

You may need to adjust filtering logic based on your setup:

**Option A: Filter by pipeline slug pattern**
```go
strings.HasPrefix(pipeline.Slug, "deploy-")
```

**Option B: Filter by specific pipelines**
```go
deployPipelines := map[string]bool{
    "deploy-production": true,
    "deploy-staging": true,
}
```

**Option C: Filter by branch**
```go
branch == "main" || branch == "production"
```

Ask your team which pipelines/branches represent deployments.

## Resources

- BuildKite REST API: https://buildkite.com/docs/apis/rest-api
- API Token Management: https://buildkite.com/user/api-access-tokens
- Rate Limits: https://buildkite.com/docs/apis/rest-api/limits
