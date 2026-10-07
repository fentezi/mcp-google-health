# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

An MCP (Model Context Protocol) server exposing Google Health API (`google.golang.org/api/health/v4`) data as MCP tools, written in Go (module `github.com/fentezi/mcp-google-health`). Uses the official `modelcontextprotocol/go-sdk` over Streamable HTTP.

## Commands

```sh
go build ./...
go run ./cmd/main                          # reads .env via godotenv; see example.env
go test ./...
go test ./internal/<pkg> -run TestName     # single test
go vet ./...
gofmt -l .
docker build -t googlehealth-mcp .
```

No tests, Makefile, linter config, or CI exist yet.

## Architecture

`internal/app.Run` runs two servers concurrently (errgroup) for the whole process lifetime:

- **Login** (`internal/server`, `SERVER_HOST:SERVER_PORT`) — `/login?key=<random>` (key logged as `login_url`) and the callback at the path of `OAUTH_REDIRECT_URL`.
- **MCP** (`SERVER_HOST:MCP_PORT` at `MCP_BASE_PATH`) — behind `RequireBearerToken(MCP_AUTH_TOKEN)`.

`internal/auth`: `Client` reads the current token on every request (`validToken`), refreshes and persists it atomically, and on `invalid_grant` deletes the token so the user can log in again without restart. Until a token exists, tools fail with `ErrNotAuthorized`.

`internal/mcpserver`: every Health data type is exposed via the generic `registerDataPointsTool[T]`, which calls `Users.DataTypes.DataPoints.List("users/me/dataTypes/<type>")` and maps each `*health.DataPoint` through an `extract` func into a flat sample struct. To add a data type: define a `XxxSample` struct + `extractXxx`, then register it in `registerDataTypeTools`. `DisableLocalhostProtection` is intentional (see comment in `Handler`).

## Configuration

`internal/config` uses `caarlos0/env/v11`; `example.env` lists every variable. Prefixes: `SERVER_`, `MCP_`, `OAUTH_`. Required: `MCP_AUTH_TOKEN`, `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`.

Tag syntax: `env:"NAME,required"` for required fields and `envPrefix:"PREFIX_"` for nested structs (not `required:"true"` / `env-prefix`, which are `cleanenv` syntax and silently ignored here).
