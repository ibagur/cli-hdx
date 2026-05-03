# AGENTS.md

This file gives coding-agent instructions for this repository. It applies to the whole repo unless a more specific `AGENTS.md` exists in a subdirectory.

## Project Purpose

`hapi` is an agent-first Go CLI for HDX HAPI, packaged as `hdx-hapi-cli`.

Keep it a standalone shell CLI. Do not implement an MCP server unless the user explicitly asks for one. Agents should be able to call the executable directly and consume stable machine-readable output.

## Core Commands

```bash
# Build and run
go run ./cmd/hapi --help
go build -o hapi ./cmd/hapi

# Unit tests; must not require network access
go test ./...

# Single package
go test ./internal/cli/...

# Live integration; requires credentials and must stay gated
HAPI_APP_IDENTIFIER="..." go test ./internal/integration -run Live
```

Use `gofmt` on edited Go files before finishing.

## Architecture

Entry point:

- `cmd/hapi/main.go` bootstraps `cli.NewRootCommand` and writes a JSON error envelope to stdout on failure.

Layer flow:

```text
cli (cobra commands)
  -> config.Resolve()       # flags > env > ~/.config/hapi/config.toml > defaults
  -> workflows.Service      # composite retrieval workflows
  -> client.Client          # HTTP, pagination, format negotiation
  -> output                 # json/jsonl/csv/table
```

Key packages:

- `internal/cli`: Cobra commands. The `state` struct carries `Options{Stdout, Stderr, HTTPClient, Env, ConfigPath}` so tests and callers can redirect dependencies.
- `internal/config`: Configuration resolution. `HAPI_*` env vars are primary; `HDX_*` aliases are supported for compatibility.
- `internal/registry`: Static endpoint key to versioned path map. Use `Lookup`/`MustPath` instead of scattering endpoint strings through commands.
- `internal/client`: HAPI HTTP client. Handles URL construction, pagination, JSON fetches, and CSV fetches.
- `internal/workflows`: High-level retrieval discipline such as resolving locations, checking availability, and then fetching substantive records. Depends on the `Queryer` interface so tests can mock upstream calls.
- `internal/output`: Stable output envelopes and format writers.
- `internal/integration`: Live smoke tests. These must remain skipped unless required environment variables are present.

## Agent Contract

stdout is data. stderr is diagnostics.

Never mix logs, progress text, debug traces, or human commentary into JSON stdout. JSON output is the stable machine-readable contract.

Success responses should use `output.Envelope`:

```json
{
  "ok": true,
  "source": "HDX/HAPI",
  "api_version": "v2",
  "endpoint": "metadata/data-availability",
  "query": {},
  "count": 12,
  "data": [],
  "meta": {
    "limit": 1000,
    "offset": 0,
    "all_pages": false,
    "warnings": []
  }
}
```

Errors should use `output.ErrorEnvelope`:

```json
{
  "ok": false,
  "error": {
    "code": "missing_app_identifier",
    "message": "Set HAPI_APP_IDENTIFIER or pass --app-identifier.",
    "retryable": false
  }
}
```

Exit codes:

- `0`: success
- `1`: CLI usage or configuration error
- `2`: HAPI validation or bad request
- `3`: network, timeout, malformed response, or provider unavailable
- `4`: no data returned
- `5`: partial data with one or more page failures

## HAPI Retrieval Rules

When adding or changing HAPI behavior:

- Verify endpoint paths against official HAPI documentation, OpenAPI/Swagger, the HAPI sandbox, or the official repository before hard-coding paths.
- Preserve the v2 registry behavior unless there is an explicit reason to support another API version.
- Prefer endpoint registry keys over literal paths in workflow code.
- Resolve countries through `metadata/location`.
- Check `metadata/data-availability` before substantive workflow retrieval.
- Prefer `location_code` over free-text filters once a location is resolved.
- Preserve HAPI records as returned, except for explicit `--fields` projection.
- Do not aggregate humanitarian figures unless an explicit analytical method is designed, documented, and tested.
- Treat ACLED-derived `conflict-events` rows as source records. Do not compute risk scores, total fatalities, or trend classifications in the CLI.
- Include source metadata fields such as `resource_hdx_id`, `dataset_hdx_stub`, and reference periods when HAPI provides them.

The registry was checked in this repo against the live official OpenAPI document at `https://hapi.humdata.org/openapi.json` on 2026-05-03, and the expanded v2 endpoint names were checked against official HAPI docs and changelog on 2026-05-04. Re-check the official source before making endpoint changes.

## Configuration Rules

Configuration precedence is:

1. CLI flag
2. Environment variable
3. Config file at `~/.config/hapi/config.toml`
4. Built-in default

Supported environment variables include:

- `HAPI_APP_IDENTIFIER`
- `HDX_APP_IDENTIFIER` compatibility alias
- `HAPI_BASE_URL`
- `HDX_BASE_URL` compatibility alias
- `HAPI_API_VERSION`
- `HDX_API_VERSION` compatibility alias
- `HAPI_TIMEOUT`
- `HDX_TIMEOUT` compatibility alias

Never commit credentials, generated identifiers, local config files, or secrets.

## Implementation Guidelines

- Inspect the current repo state before editing.
- Keep changes scoped to the requested behavior.
- Follow existing package boundaries and naming conventions.
- Keep unit tests offline by using fake HTTP transports, mocks, or local test servers.
- Gate all live tests behind environment variables and skip when credentials are missing.
- Add workflow commands only when they encode useful retrieval discipline, not just another spelling of a raw endpoint.
- Keep raw endpoint access available through `hapi get <endpoint> --param key=value`.
- Treat table output as human-facing. Treat JSON output as the stable contract.
- Run the relevant tests before claiming completion, and run `go test ./...` when the change touches shared behavior.

## Documentation

When behavior changes, update the relevant docs:

- `README.md` for user-facing commands, configuration, examples, and installation notes.
- `docs/agent-contract.md` for output envelopes, exit codes, and agent usage expectations.
- `templates/AGENTS.md` only when the transferable template for other API CLI repos should change.
- Skill or report files only when the user specifically asks or the change directly affects them.
