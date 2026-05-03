# Agent Instructions

This repository is for building an agent-first standalone CLI for an external API.

Do not implement an MCP server unless explicitly requested. The goal is a shell CLI that agents can call directly, receive stable machine-readable output from, and later wrap in a Skill or MCP layer only if useful.

## Core Design Rules

- JSON output by default.
- stdout is data only.
- stderr is diagnostics only.
- Never mix logs or progress text into JSON stdout.
- Use stable success and error envelopes.
- Use non-zero exit codes consistently.
- Do not put secrets in the repo.
- Unit tests must not require network access.
- Live integration tests must be gated by environment variables and skipped when credentials are missing.
- Verify current endpoint paths from official docs, OpenAPI/Swagger, API docs, sandbox, or the official repository before hard-coding paths.
- Provide direct raw endpoint access, such as `tool get <endpoint> --param key=value`.
- Add workflow commands only when they encode useful retrieval discipline.
- Preserve provider records as returned unless normalization is explicitly designed, documented, and tested.
- Do not aggregate, summarize, or reinterpret provider figures unless an explicit analytical method is designed and documented.

## Expected CLI Layers

1. CLI parser
2. Config resolver
3. Endpoint registry
4. HTTP client
5. Raw endpoint command
6. Optional workflow commands
7. Output adapters
8. Error and exit-code mapper
9. Documentation
10. Unit and gated integration tests

## API Exploration Questions

When exploring the API, first answer:

- What is the official documentation source?
- Is there OpenAPI/Swagger?
- What is the base URL?
- How does authentication work?
- What are the pagination rules?
- What is the response shape?
- What is the error shape?
- What reference metadata exists?
- Which endpoints should be exposed directly?
- Which workflows, if any, should be added?
- What source, citation, provenance, rate-limit, or freshness metadata should be preserved?

## Implementation Expectations

- Start by inspecting the repo state.
- Verify endpoint paths before implementation.
- Write tests before or alongside implementation.
- Do not require network access for unit tests.
- Use fake HTTP transports, mocks, or local test servers for unit tests.
- Gate live tests behind environment variables.
- Keep credentials in flags, environment variables, or user config outside the repo.
- Run the full test suite before claiming completion.

## Output Contract Preference

Prefer this success envelope unless there is a provider-specific reason to change it. Replace `Provider/API`, `v1`, and endpoint examples with the actual API details:

```json
{
  "ok": true,
  "source": "Provider/API",
  "api_version": "v1",
  "endpoint": "example/path",
  "query": {},
  "count": 0,
  "data": [],
  "meta": {
    "limit": 1000,
    "offset": 0,
    "all_pages": false,
    "warnings": []
  }
}
```

Prefer this error envelope:

```json
{
  "ok": false,
  "error": {
    "code": "error_code",
    "message": "Human-readable message.",
    "retryable": false
  }
}
```

## Exit Code Preference

- `0`: success
- `1`: CLI usage or configuration error
- `2`: provider validation or bad request
- `3`: network, timeout, malformed response, or provider unavailable
- `4`: no data returned
- `5`: partial data with one or more page failures
