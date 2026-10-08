# Changelog

## [Unreleased]

### Added
- MCP server over Streamable HTTP with bearer-token auth.
- Google OAuth login flow with persisted, auto-refreshed token.
- Tools: heart rate, HRV, steps, distance, floors, weight, height, body fat, oxygen saturation, blood glucose, sleep, exercise.
- Docker image.
- `pageToken` input for fetching further pages.
- Re-authorization without restart when the refresh token is revoked.
- golangci-lint configuration.
- Makefile: local `run`/`build`, Docker run with `.env`, ports and token volume.

### Security
- Login URL requires a one-time key printed in the logs.
- Empty `MCP_AUTH_TOKEN` is rejected at startup.
