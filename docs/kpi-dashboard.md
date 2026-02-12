# KPI / Metrics Dashboard

The web app includes a **Time in Build** KPI driven by JIRA filter [22515](https://appliedintuition.atlassian.net/issues/?filter=22515) (vehicle build epics).

## Metric: Time in Build (car in → release to fleet)

- **X-axis:** Calendar week (ISO week, e.g. 2025-W06).
- **Y-axis:** Average days.
- **Two lines:**
  - **Rogue:** Epic name contains **ROG**. Days between the **start of the first VBUILD ticket** (first transition to *In Progress*) and the **last ticket** in that epic moved to *Done*.
  - **MachE:** Epic name contains **MCE**. Days between **epic opened** (epic created) and the **release to fleet** ticket closed (transition to *Done*).

## How it works

1. **Backend** (`GET /api/kpi/time-in-build?filter_id=22515`):
   - Fetches the saved JIRA filter’s JQL.
   - Runs a search for epics (bounded by `created >= -365d` if the filter JQL doesn’t already restrict by date).
   - For each epic, fetches child issues (`parent = <epicKey>` or `parentEpic = <epicKey>`).
   - Classifies epics as **Rogue** or **MachE** by **labels** or **summary** (e.g. label `Rogue` / `MachE` or summary containing "rogue" / "mache").
   - For **Rogue:** among children whose summary contains `VBUILD`, uses changelog to get first *In Progress* and last *Done* → computes days.
   - For **MachE:** finds the child whose summary contains `release to fleet`, uses changelog for *Done* → days from epic created.
   - Aggregates by calendar week (average days per week) and returns `weeks`, `rogue`, `machE`, plus optional `meta`.

2. **Frontend:** The **KPI Dashboard** page calls this API and plots a line chart (Recharts) with two series.

## Configuration

- **JIRA:** Same as [JIRA setup](jira-setup.md) (`JIRA_DOMAIN`, `JIRA_EMAIL`, `JIRA_API_TOKEN` in `.env`).
- **Filter:** Default filter ID is `22515`. Override with `?filter_id=...` on `/api/kpi/time-in-build`.
- **Limits:** Backend caps at 25 epics and 30 children per epic to avoid timeouts; adjust `kpiMaxEpics` / `kpiMaxChildren` in `kpi.go` if needed.

## Customizing Rogue / MachE and ticket types

Detection is heuristic:

- **Rogue / MachE:** Epic **name (summary)** containing **ROG** = Rogue build, **MCE** = MachE build (case-insensitive).
- **VBUILD:** Child issue **summary** containing "vbuild".
- **Release to fleet:** Child issue **summary** containing "release to fleet".
- **Status names:** Changelog is checked for status *In Progress* and *Done* (exact match). If your workflow uses different names (e.g. "In Progress" vs "In progress"), update `statusTransitionFromChangelog` calls in `kpi.go`.

You can extend `kpi.go` to use custom fields (e.g. a "Vehicle line" select) instead of labels/summary if your JIRA project uses them.

## Adding more KPIs

The same pattern can be reused for 7–8 metrics:

1. Add a new backend handler (e.g. in `kpi.go`) that fetches the right JIRA (or other) data and returns time-series or single values.
2. Expose it as e.g. `GET /api/kpi/<metric-name>`.
3. Add a new chart or card on the Dashboard (or a new dashboard tab) that fetches that endpoint and visualizes the data.
