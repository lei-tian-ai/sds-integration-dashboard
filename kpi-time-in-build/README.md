# Time in Build KPI – Source and Documentation

This folder contains the source code and documentation for the **Time in Build** KPI (first KPI on the dashboard). Use it to replicate the same metric in another app or share with colleagues.

---

## 1. What This KPI Does

- **Metric**: Average “time in build” (days) per calendar week, for vehicle build epics.
- **Series**:
  - **Rogue**: Epics whose summary contains `"ROG"`. Build time = epic created → epic resolved (approximation; full definition uses first VBUILD child In Progress → last VBUILD child Done).
  - **MachE**: Epics whose summary contains `"MCE"` (excluding D-Max/DMX). Build time = epic created → epic resolved.
  - **Other**: All other closed epics from the filter. Build time = epic created → epic resolved.
- **X-axis**: Calendar week (ISO week, e.g. `2025-W01`).
- **Y-axis**: Average days for that week (per series).
- **Output**: Line chart + table of epics with vehicle name, start/finish, build days, and type (Rogue/MachE/Other).

---

## 2. Logic Flow

1. **Resolve epic set**
   - Either:
     - **Filter path**: Call JIRA `GET /rest/api/3/filter/{filter_id}` to get the saved filter’s JQL (default `filter_id=22515`).
     - Or **Custom JQL**: Use `jql` query param as the base JQL.
   - Strip `ORDER BY` and open-only clauses (e.g. `resolution is empty`, `status not in (Done, Closed)`) so closed epics are included.
   - Restrict to epics and optional time window:
     - Append `AND issuetype = Epic`.
     - If JQL has no `created`, append `AND created >= -730d` (2 years).
   - Optional: `project_keys` query param adds extra epics by project; `include_epic_keys` adds specific epic keys.

2. **Fetch epics**
   - Search JIRA with the final JQL via `GET /rest/api/3/search/jql` (paginated, 100 per page, up to 300 epics).
   - Fields requested: `summary`, `status`, `created`, `updated`, `labels`, `resolutiondate`.

3. **Compute build time (epic-level approximation)**
   - For each epic with both `created` and `resolutiondate` and `resolutiondate > created`:
     - Build days = `(resolutiondate - created)` in days.
     - Week = ISO week of `resolutiondate`.
   - Classify by summary:
     - **Rogue**: summary (uppercase) contains `"ROG"`.
     - **MachE**: summary (uppercase) contains `"MCE"` and does not contain `"D-MAX"` / `"DMAX"` / `"DMX-"`.
     - **Other**: everything else.

4. **Aggregate and respond**
   - Group by week and series; compute average days per week per series.
   - Build epic table rows (epic key, summary, vehicle name, start/finish, build days, week, type).
   - Build per-week vehicle labels for Rogue/MachE/Other for the chart.
   - Return JSON: `weeks`, `rogue`, `machE`, `other`, `epic_rows`, `week_labels_rogue`, `week_labels_mach_e`, `week_labels_other`, `meta` (filter_id, jql_used, epics_seen, rogue_n, machE_n, other_n, epic_keys).

---

## 3. Key Query Items (JQL)

- **Default source**: Saved JIRA filter **22515** (JQL fetched from JIRA).
- **Effective JQL** (after processing):
  - Base: filter JQL (or custom `jql` param) with `ORDER BY` and open-only clauses stripped.
  - Then: `( <base> ) AND issuetype = Epic [ AND created >= -730d ]`.
- **Optional**:
  - `project_keys=VBUILD` (or comma-separated list): also include epics from those projects with `created >= -730d`.
  - `include_epic_keys=KEY1,KEY2`: also include those epic keys (fetched by key).
  - `jql=...`: use this JQL instead of the filter.

---

## 4. APIs Used

| API | Method | Purpose |
|-----|--------|--------|
| JIRA: Get filter | `GET /rest/api/3/filter/{id}` | Get JQL for saved filter (e.g. 22515). |
| JIRA: Search (JQL) | `GET /rest/api/3/search/jql` | Search issues by JQL (epics). Params: `jql`, `maxResults`, `startAt`, `fields`. |
| JIRA: Get issue | `GET /rest/api/3/issue/{key}` | Optional: fetch single epic by key for `include_epic_keys`. |

**Authentication**: Basic auth with JIRA email + API token (e.g. `JIRA_EMAIL`, `JIRA_API_TOKEN`). Base URL: `https://{JIRA_DOMAIN}.atlassian.net`.

---

## 5. Dependencies

### Backend (Go)
- **github.com/gin-gonic/gin** – HTTP router and context.
- **Standard library**: `encoding/base64`, `encoding/json`, `fmt`, `io`, `math`, `net/http`, `net/url`, `sort`, `strings`, `time`, `os`.

### Frontend (React)
- **react** (useState, useEffect).
- **recharts**: `LineChart`, `Line`, `XAxis`, `YAxis`, `CartesianGrid`, `Tooltip`, `Legend`, `ResponsiveContainer`, `LabelList`.

---

## 6. Environment / Config

- **JIRA_DOMAIN** – JIRA Cloud domain (e.g. `appliedintuition`).
- **JIRA_EMAIL** – Atlassian account email.
- **JIRA_API_TOKEN** – API token (Atlassian account security).

---

## 7. Files in This Package

| Path | Description |
|------|-------------|
| `README.md` | This file: logic, queries, APIs, dependencies. |
| `backend/time_in_build.go` | Go handler and all helpers (JIRA config, get filter, search, field helpers, Rogue/MachE/Other logic). Copy into your Go service. |
| `frontend/TimeInBuildSection.tsx` | React component: fetch, chart, table, types. Copy into your app and mount where the KPI should appear. |

To share as a zip from the repo root:  
`zip -r kpi-time-in-build.zip kpi-time-in-build`

---

## 8. How to Integrate

1. **Backend**
   - Register route: `GET /api/kpi/time-in-build` → `kpiTimeInBuild`.
   - Ensure `jiraConfig` / `jiraAPIReq` (or equivalent) use your JIRA env vars.
   - Paste in the contents of `backend/time_in_build.go` (adjust package if needed).

2. **Frontend**
   - Install `recharts` if not already.
   - Add `frontend/TimeInBuildSection.tsx` and render it (e.g. in your dashboard).
   - The component expects `GET /api/kpi/time-in-build?filter_id=22515` (or your backend URL) and renders the chart + table + meta.

---

## 9. Response Shape (Backend → Frontend)

```json
{
  "weeks": ["2024-W01", "2024-W02", ...],
  "rogue": [12.5, 0, 8.2, ...],
  "machE": [0, 15.0, 0, ...],
  "other": [0, 0, 5.1, ...],
  "epic_rows": [
    {
      "epic_key": "VBUILD-123",
      "summary": "ROG-131 - ...",
      "vehicle_name": "ROG-131",
      "start_time": "2024-01-01T00:00:00Z",
      "finish_time": "2024-01-15T00:00:00Z",
      "build_days": 14.0,
      "week": "2024-W02",
      "type": "Rogue"
    }
  ],
  "week_labels_rogue": { "2024-W02": ["ROG-131"] },
  "week_labels_mach_e": {},
  "week_labels_other": {},
  "meta": {
    "filter_id": "22515",
    "jql_used": "( ... ) AND issuetype = Epic AND created >= -730d",
    "epic_keys": ["VBUILD-123", ...],
    "epics_seen": 42,
    "rogue_n": 10,
    "machE_n": 8,
    "other_n": 24
  }
}
```

---

## 10. References

- JIRA Cloud REST API v3 – Issue search: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/
- Filter API: `GET /rest/api/3/filter/{id}` returns `{ "jql": "..." }`.
