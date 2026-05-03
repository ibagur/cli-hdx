# hapi

`hapi` is an agent-first Go CLI for HDX HAPI, packaged as `hdx-hapi-cli`.

It is intentionally a standalone shell tool, not an MCP server. Agents can call it directly, receive stable machine-readable output, and later wrap it in a Skill or MCP layer if useful.

## Install

From source:

```bash
go install github.com/ibagur/cli-hdx/cmd/hapi@latest
```

Local development:

```bash
go test ./...
go run ./cmd/hapi --help
```

## Configuration

Set your HAPI app identifier outside project repositories:

```bash
export HAPI_APP_IDENTIFIER="..."
```

You can generate one locally from an app name and email:

```bash
hapi auth init --app-name "my-agent" --email "me@example.org"
```

Resolution precedence is:

1. Explicit CLI flag
2. Environment variable
3. Config file at `~/.config/hapi/config.toml`
4. Built-in default

Supported environment variables:

- `HAPI_APP_IDENTIFIER`
- `HDX_APP_IDENTIFIER` as a compatibility alias
- `HAPI_BASE_URL`, default `https://hapi.humdata.org/api`
- `HDX_BASE_URL` as a compatibility alias; either `https://hapi.humdata.org` or `https://hapi.humdata.org/api` is accepted
- `HAPI_API_VERSION`, default `v2`
- `HDX_API_VERSION` as a compatibility alias
- `HAPI_TIMEOUT` or `HDX_TIMEOUT`, default `30`

Minimal config file:

```toml
app_identifier = "..."
base_url = "https://hapi.humdata.org/api"
api_version = "v2"
limit = 1000
```

## Endpoint Verification

The v2 registry was checked against the live official OpenAPI document at `https://hapi.humdata.org/openapi.json` on 2026-05-03. Current HAPI docs and changelog were also reviewed on 2026-05-04 for the expanded v2 registry. The known paths include:

- `metadata/location`
- `metadata/data-availability`
- `coordination-context/operational-presence`
- `coordination-context/funding`
- `coordination-context/conflict-events`
- `food-security-nutrition-poverty/food-security`
- `food-security-nutrition-poverty/food-prices-market-monitor`
- `food-security-nutrition-poverty/poverty-rate`
- `affected-people/idps`
- `affected-people/refugees-persons-of-concern`
- `affected-people/returnees`
- `affected-people/humanitarian-needs`
- `geography-infrastructure/baseline-population`
- `coordination-context/national-risk`
- `climate/hazards-rainfall`

The official changelog also notes the 2025-02-18 endpoint rename and that the renamed endpoints are released under API v2.

## Examples

```bash
hapi metadata locations --name Sudan --format json
hapi metadata availability --location-code SDN --format json
hapi list-endpoints --format table
hapi get metadata/sector --param name=Water --format json
hapi get coordination-context/conflict-events --param location_code=SDN --format json
hapi workflow wash-3w --country Nigeria --admin1-name Yobe --format json
hapi workflow funding --country "South Sudan"
hapi workflow food-security --country Mozambique --ipc-phase 3+ --admin-level 1
hapi workflow refugees --country Uganda --format json
hapi workflow population --country Nepal --admin-level 1 --format json
hapi workflow conflict-events --country Sudan --event-type battles --start-date 2026-01-01 --format json
```

JSON is the default. Use `--format jsonl`, `--format csv`, or `--format table` when needed.

Global flags include `--limit`, `--offset`, `--all-pages`, `--fields`, `--output`, `--api-version`, `--base-url`, `--app-identifier`, `--timeout`, `--quiet`, and `--debug`.

## Exit Codes

- `0`: success
- `1`: CLI usage or configuration error
- `2`: HAPI validation or bad request
- `3`: network, timeout, malformed response, or provider unavailable
- `4`: no data returned
- `5`: partial data with one or more page failures

## Testing

Unit tests do not require network access. Live smoke testing is gated by `HAPI_APP_IDENTIFIER`:

```bash
go test ./...
HAPI_APP_IDENTIFIER="..." go test ./internal/integration -run Live
```
