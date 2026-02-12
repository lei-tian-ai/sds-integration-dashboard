# Fleetio integration

The app can read data from [Fleetio](https://www.fleetio.com/) (fleet management) via the [Fleetio API](https://developer.fleetio.com/). You need an **Account Token** and an **API Key**.

## 1. Get your Fleetio credentials

1. Log in at [Fleetio](https://secure.fleetio.com/) (your dashboard URL is like `https://secure.fleetio.com/<account_id>/dashboard/...`).
2. Open **Settings** (sidebar dropdown) → **Manage API Keys**.
3. **Account Token** – at the bottom of the “Manage API Keys” page. Use this as `FLEETIO_ACCOUNT_TOKEN`.
4. **API Key** – click **+ Add API Key**, add a label, choose API version, then copy the key. Use it as `FLEETIO_API_KEY`.

Details: [Fleetio Quick Start](https://developer.fleetio.com/docs/overview/quick-start).

## 2. Configure the app

### Using `.env` (recommended for local)

Add to your `.env` (same file as JIRA):

```env
FLEETIO_ACCOUNT_TOKEN=your_account_token
FLEETIO_API_KEY=your_api_key
```

Restart the backend so it reloads `.env`.

### Deployed (Apps Platform)

- Set **FLEETIO_ACCOUNT_TOKEN** and **FLEETIO_API_KEY** in the app’s environment (e.g. secrets or env vars in the platform).
- Do not commit real keys; use a secret store for the API key.

## 3. API endpoints

Base path: **`/api/fleetio`**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/fleetio/me` | Current user (useful to verify auth). |
| GET | `/api/fleetio/vehicles` | List vehicles (paginated). |

### Query params for `/api/fleetio/vehicles`

| Param | Default | Description |
|-------|---------|-------------|
| `per_page` | `25` | Results per page. |
| `page` | `1` | Page number. |

Response includes `vehicles`, `total_count`, `total_pages`, `current_page` (from Fleetio’s pagination headers when present).

## 4. Example requests

```bash
# Check Fleetio auth (current user)
curl -s "http://localhost:8082/api/fleetio/me"

# List first page of vehicles
curl -s "http://localhost:8082/api/fleetio/vehicles"

# Second page, 10 per page
curl -s "http://localhost:8082/api/fleetio/vehicles?page=2&per_page=10"
```

If Fleetio isn’t configured, these return **503** with a `missing` list of required env vars.

## 5. More Fleetio data

The [Fleetio API Reference](https://developer.fleetio.com/docs/category/api) includes many resources (meter entries, fuel entries, issues, etc.). You can add more backend routes that call `https://secure.fleetio.com/api/v1/<resource>` with the same `Authorization: Token <key>` and `Account-Token: <account_token>` headers.
