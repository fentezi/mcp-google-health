# googlehealth-mcp

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants read-only access to your [Google Health API](https://developers.google.com/health) data: heart rate, sleep, steps, weight, and more.

## Tools

| Tool | Data |
|---|---|
| `get_heart_rate` | Heart-rate samples |
| `get_heart_rate_variability` | HRV (RMSSD, SDNN) |
| `get_steps` | Step-count intervals |
| `get_distance` | Distance intervals |
| `get_floors` | Floors-climbed intervals |
| `get_weight` | Weight samples |
| `get_height` | Height samples |
| `get_body_fat` | Body-fat percentage |
| `get_oxygen_saturation` | Blood-oxygen saturation |
| `get_blood_glucose` | Blood glucose |
| `get_sleep` | Sleep sessions |
| `get_exercise` | Exercise sessions |
| `ping` | Health check |

Every data tool accepts optional `filter` ([AIP-160](https://google.aip.dev/160) expression, e.g. `steps.interval.start_time >= "2026-01-01T00:00:00Z"`) and `pageSize`.

## Setup

### 1. Google Cloud

1. Create a project in [Google Cloud Console](https://console.cloud.google.com) and enable the **Google Health API**.
2. Configure the OAuth consent screen and add yourself as a test user.
3. Create an OAuth client of type **Web application** with the authorized redirect URI `http://localhost:8080/oauth/callback`.

### 2. Configuration

```sh
cp example.env .env
```

Fill in `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`, and set `MCP_AUTH_TOKEN` to a long random string (e.g. `openssl rand -hex 32`). See `example.env` for all options.

### 3. Run

With Go 1.26+:

```sh
go run ./cmd/main
```

Or with Docker:

```sh
docker build -t googlehealth-mcp .
docker run --env-file .env -p 127.0.0.1:8080:8080 -p 127.0.0.1:8060:8060 -v googlehealth-data:/data googlehealth-mcp
```

The image binds to `0.0.0.0` and stores the token in the `/data` volume.

### 4. Authorize

On start without a token, the server logs a one-time `login_url` (`http://localhost:8080/login?key=...`). Open it and grant access. The token is saved to `OAUTH_TOKEN_FILE` and refreshed automatically.

Until then, MCP tools return a "not authorized" error. If Google later revokes the refresh token (apps in testing mode expire it after 7 days), the server drops it and logs a new login URL; no restart needed.

The login server listens on `SERVER_PORT` and serves the callback at the path of `OAUTH_REDIRECT_URL`, so keep their ports in sync.

## Connecting a client

The endpoint is `http://localhost:8060/mcp/health` and requires `Authorization: Bearer <MCP_AUTH_TOKEN>`.

Claude Code:

```sh
claude mcp add --transport http googlehealth http://localhost:8060/mcp/health \
  --header "Authorization: Bearer <MCP_AUTH_TOKEN>"
```

Other clients:

```json
{
  "mcpServers": {
    "googlehealth": {
      "type": "http",
      "url": "http://localhost:8060/mcp/health",
      "headers": { "Authorization": "Bearer <MCP_AUTH_TOKEN>" }
    }
  }
}
```

## Security

This server exposes personal health data. Keep `MCP_AUTH_TOKEN` secret, never commit `.env` or `token.json`, and put the server behind TLS if it is reachable beyond localhost.

## License

[MIT](LICENSE)
