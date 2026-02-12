# sds-integration-dashboard

Vehicle Build KPI Dashboard for the SDS team. Tracks build timelines, engineering workload, quality issues, and vehicle reliability using data from JIRA and Fleetio.

Built with a Go backend (Gin) and an embedded React frontend (TypeScript + Tailwind CSS).

## Setup

Install dependencies:

```bash
make deps
```

## Running

Run in development mode (backend and frontend separately):

```bash
make run
```

This will start:
- Go backend on http://localhost:8082
- React frontend on http://localhost:3000

## Building

Build the production binary with embedded frontend:

```bash
make build
```

This creates an `app` binary with the React frontend embedded.

## Before Deploying

Before deploying to Apps Platform, you need to configure the `project.toml` file:

1. **Update the app name** - Change `name` to your desired app name (lowercase, hyphens allowed) -- your app will be hosted at https://your-app-name.experimental.apps.applied.dev
2. **Add your email** - Update `owner` with your Applied email address

Example `project.toml`:

```toml
name = "my-awesome-app"

[metadata]
owner = "yourname@applied.co"
description = "My awesome app"
```

## Deploying

Deploy to Apps Platform:

```bash
make deploy
```

## Structure

- `main.go` - Go backend with Gin web server and embedded frontend
- `frontend/` - React frontend with TypeScript and Tailwind CSS
- `Makefile` - Build and deployment commands
- `project.toml` - Apps Platform metadata
