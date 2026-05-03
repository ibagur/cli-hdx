# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build / run
go run ./cmd/hapi --help
go build -o hapi ./cmd/hapi

# Test (no network required)
go test ./...

# Single package
go test ./internal/cli/...

# Live integration (requires HAPI_APP_IDENTIFIER)
HAPI_APP_IDENTIFIER="..." go test ./internal/integration -run Live
```

## Architecture

Entry point: `cmd/hapi/main.go` — bootstraps a `cobra.Command` via `cli.NewRootCommand` and writes a JSON error envelope to stdout on failure.

**Layer flow:**

```
cli (cobra commands)
  → config.Resolve()       # flags > env > ~/.config/hapi/config.toml > defaults
  → workflows.Service      # composite queries (country-overview, wash-3w, funding, etc.)
  → client.Client          # HTTP, pagination, format negotiation
  → output (json/jsonl/csv/table)
```

**Key packages:**

- `internal/cli` — all cobra commands; `state` struct carries `Options{Stdout, Stderr}` so callers can redirect output.
- `internal/config` — `Resolve(flags, env, path)` merges config; `HDX_*` env vars are aliases for `HAPI_*`.
- `internal/workflows` — high-level operations that may call multiple HAPI endpoints (e.g. resolve location → check availability → fetch). Takes a `Queryer` interface — easy to mock in tests.
- `internal/client` — `Fetch` (JSON, paginated) and `FetchCSV`; `BuildURL` handles versioned paths.
- `internal/registry` — static map of endpoint keys → versioned URL paths; use `Lookup(version, key)` to resolve.
- `internal/output` — `Envelope`/`ErrorEnvelope` structs; `WriteJSON`, `WriteJSONL`, `WriteCSV`, `WriteTable`.

## Agent Contract

stdout = data, stderr = diagnostics. JSON output is the stable machine-readable contract.

All responses wrapped in `Envelope{ok, source, api_version, endpoint, query, count, data, meta}`. Errors produce `ErrorEnvelope{ok:false, error:{code, message, retryable}}`.

Exit codes: `0` success · `1` config/usage · `2` bad request · `3` network · `4` no data · `5` partial.

## Configuration Precedence

1. CLI flag
2. Env var (`HAPI_APP_IDENTIFIER` / `HDX_APP_IDENTIFIER`, `HAPI_BASE_URL` / `HDX_BASE_URL`, etc.)
3. `~/.config/hapi/config.toml`
4. Built-in default

`HDX_BASE_URL` accepts both `https://hapi.humdata.org` and `https://hapi.humdata.org/api`.

## HAPI API Notes

Registry verified against live OpenAPI (`https://hapi.humdata.org/openapi.json`) on 2026-05-03, version `0.9.13`, API v2. Endpoint rename from 2025-02-18 is reflected in the registry.

Workflow commands should resolve location via `metadata/location` first, then check `metadata/data-availability` before substantive retrieval. Use `location_code` (e.g. `SDN`) over free-text filters once resolved.
