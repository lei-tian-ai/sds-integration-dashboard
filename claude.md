# Claude Development Guide for SDS Integration Dashboard

## Project Overview

This is a Vehicle Build KPI Dashboard for the SDS (Self-Driving Systems) team at Applied Intuition. It tracks vehicle integration build timelines, engineering workload, quality issues, and vehicle reliability metrics.

**Tech Stack:**
- Backend: Go with Gin web framework
- Frontend: React with TypeScript and Tailwind CSS
- Data Sources: JIRA (for tickets/epics) and BuildKite (for deployments)
- Charting: Recharts library

## Architecture

The application follows a Go + Embedded React pattern:
- Go backend serves REST APIs at `/api/kpi/*`
- React frontend is embedded into the Go binary for production
- Development mode runs backend (port 8082) and frontend (port 3000) separately

## Key Files

- `main.go` - Entry point, Gin server setup, embedded frontend
- `kpi.go` - Main KPI metrics API handlers
- `buildkite.go` - BuildKite API integration for deployment metrics
- `buildkite_optimized.go` - Optimized version with combined endpoint
- `jira.go` - JIRA API integration for ticket/epic data
- `frontend/src/components/DashboardCompact.tsx` - Main dashboard component with all KPI widgets

## Dashboard Widgets (3-column grid)

1. **Cars in Build** - Table showing active vehicle epics with build timelines
2. **Time in Build** - Line chart of average build days by vehicle type (Rogue, MachE, Other)
3. **VOS Tickets in Build** - Tracks Vehicle OS engineer workload
4. **Bug Tickets in Build** - Issues caught after calibration
5. **MTBF** - Mean Time Between Failure tracking
6. **Avg Deployment Time** - BuildKite deployment duration
7. **Deployment Failure Rate** - BuildKite deployment success metrics

### Target KPIs

Blue dashed reference lines show targets:
- Time in Build: 5 days
- VOS Tickets: <2
- Bug Tickets: <1
- Deployment Failure Rate: <5%

## Development Workflow

### Running Locally

```bash
make run    # Start both backend and frontend in dev mode
```

### Building

```bash
make build  # Build production binary with embedded frontend
```

### Deploying

```bash
make deploy # Deploy to Apps Platform
```

## Environment Variables

Check `.env.example` for required environment variables:
- JIRA credentials and configuration
- BuildKite API token
- Port and service settings

## Code Style Guidelines

- **Compact, high-density UI** - Excel-style tables with minimal padding
- **3-column grid layout** - All widgets same width for consistency
- **Date formats** - Use mm/dd/yy for compact display
- **Chart colors** - Rogue (blue #2563eb), MachE (red #dc2626), Other (green #16a34a)
- **Target lines** - Blue dashed (#3b82f6) with 1px width

## Recent Changes

- Changed dashboard from 2-column to 3-column grid layout
- Made Cars in Build table very compact (Excel-style)
- Added target reference lines to KPI charts
- Removed redundant lines from Deployment Failure Rate chart
- Added data point labels showing totals and failures
- Optimized BuildKite API calls with combined endpoint

## Working with Charts

The dashboard uses Recharts library. Key components:
- `LineChart` - Main chart container
- `Line` - Data series with optional `hide={true}` to hide from chart but keep in tooltip
- `ReferenceLine` - For target lines (e.g., `y={5}` for 5-day target)
- `LabelList` - Custom labels on data points
- `Tooltip` - Shows all data on hover, even hidden lines

## Testing

Before deploying, test locally:
1. Ensure JIRA API credentials are valid
2. Verify BuildKite API token works
3. Check all chart data loads correctly
4. Verify target lines display at correct values

## Useful Commands

```bash
make deps      # Install Go and npm dependencies
make run       # Run in development mode
make build     # Build production binary
make deploy    # Deploy to Apps Platform
make clean     # Clean build artifacts
```

## Troubleshooting

- If charts don't load, check browser console for API errors
- If BuildKite metrics are slow, use the optimized combined endpoint
- Date parsing issues: ensure ISO format from backend
- For authentication issues, check .env file has correct credentials
