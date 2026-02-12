# JIRA integration

The app can search JIRA issues via the [JIRA Cloud REST API v3](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/) (enhanced JQL search endpoint `/rest/api/3/search/jql`) using your Atlassian email and API token.

## 1. Get your JIRA site domain

Your JIRA base URL is `https://<domain>.atlassian.net`. The **domain** is the subdomain (e.g. if the URL is `https://mycompany.atlassian.net`, use `mycompany`).

## 2. Create an Atlassian API token

You already created one. If you need another:

1. Go to [Atlassian API tokens](https://id.atlassian.com/manage-profile/security/api-tokens).
2. Create a token and copy it (you won’t see it again).

## 3. Configure the app

### Deployed (Apps Platform)

- **Secrets:** Store your API token as a secret and map it to the env var `JIRA_API_TOKEN` (e.g. via Apps Platform dashboard or CLI).
- **Env vars:** Set `JIRA_DOMAIN` and `JIRA_EMAIL` (the email for your Atlassian account) in the app’s environment (e.g. in `project.toml` under `[cloudrun.env_vars]` or in the platform UI).

Example in `project.toml`:

```toml
[cloudrun.env_vars]
JIRA_DOMAIN = "mycompany"
JIRA_EMAIL = "you@company.com"
```

Then add a secret named so it’s exposed as `JIRA_API_TOKEN` (see Apps Platform docs for mapping secrets to env vars).

### Local development

**Option A – use a `.env` file (recommended):**

1. Copy the template: `cp .env.example .env`
2. Edit `.env` and set your values:
   ```
   JIRA_DOMAIN=mycompany
   JIRA_EMAIL=you@company.com
   JIRA_API_TOKEN=your-api-token
   ```
3. Run the app; the backend loads `.env` automatically. Do not commit `.env` (it’s in `.gitignore`).

**Option B – export in the shell:**

```bash
export JIRA_DOMAIN="mycompany"
export JIRA_EMAIL="you@company.com"
export JIRA_API_TOKEN="your-api-token"
```

Then run the app as usual (e.g. `make backend` or `go run .`).

## 4. Use the API

**GET** `/api/jira/search` — search issues with JQL.

| Query param   | Default                                | Description                                                                 |
|---------------|----------------------------------------|-----------------------------------------------------------------------------|
| `jql`         | `created >= -180d order by created DESC` | JQL query; must include a restriction (e.g. project, date) – unbounded queries return 400 |
| `maxResults`  | `50`                   | Max issues to return (1–100)         |

Examples:

```bash
# Recent issues from last 180 days (default)
curl "http://localhost:8082/api/jira/search"

# By project (project is a valid restriction)
curl "http://localhost:8082/api/jira/search?jql=project%3DMYPROJ%20order%20by%20created%20DESC"

# By text in summary (combine with date or project to keep query bounded)
curl "http://localhost:8082/api/jira/search?jql=summary%20~%20\"search%20term\"%20AND%20created%20%3E%3D%20-90d"
```

Response shape:

```json
{
  "total": 42,
  "issues": [
    {
      "key": "PROJ-123",
      "summary": "Issue title",
      "status": "In Progress",
      "created": "2025-02-01T10:00:00.000+0000",
      "updated": "2025-02-08T14:30:00.000+0000"
    }
  ]
}
```

If JIRA isn’t configured, the endpoint returns 503 with a message to set the env vars/secret.
