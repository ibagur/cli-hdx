# Provider Adaptation Checklist

## Official Sources

- Official docs:
- OpenAPI/Swagger:
- Official repository:
- API status/changelog:
- Date verified:

## Identity

- Repository name:
- Binary name:
- Provider/source name:
- Default API version:

## Authentication

- Required?
- Environment variable name:
- Header/query parameter format:
- Can credentials be generated?
- Secret handling notes:

## Base URL

- Production base URL:
- Sandbox/test base URL:
- Does the base URL include the API version?
- Example full endpoint URL:

## Endpoint Discovery

List verified endpoint paths here after exploration.

| Purpose | Endpoint | Notes |
|---|---|---|
| Metadata/reference | | |
| Raw records | | |
| Search/list | | |
| Detail lookup | | |

## Pagination

- Pagination style:
- Default limit:
- Max limit:
- Offset/page token parameter:
- How to detect final page:

## Response Shape

Example success response:

```json
{}
```

Where are records stored?

```text
data / results / items / other:
```

## Error Shape

Example error response:

```json
{}
```

Error mapping:

| Provider condition | CLI exit code |
|---|---|
| Usage/config error | 1 |
| Provider validation/bad request | 2 |
| Network/timeout/provider unavailable | 3 |
| No data | 4 |
| Partial data | 5 |

## Reference Metadata

Canonical identifiers:

- Countries/locations:
- Admin areas:
- Sectors/clusters:
- Organizations:
- Datasets/resources:
- Dates/reference periods:

## Candidate Commands

Raw access command:

```bash
<binary> get <endpoint> --param key=value
```

Metadata/reference commands:

```bash
<binary> metadata ...
```

Candidate workflow commands:

```bash
<binary> workflow ...
```

## Output Contract

Use this success envelope unless there is a documented reason to change it:

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

Use this error envelope:

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

## Test Requirements

- Unit tests must use mocks or fake HTTP transports.
- Unit tests must not require network access.
- Integration tests must skip unless required env vars are set.
- Test URL construction.
- Test config precedence.
- Test output envelopes.
- Test exit code mapping.
- Test pagination.
- Test no-data behavior.
- Test malformed provider responses.

## First Exploration Prompt

Use this prompt when starting a new provider repo:

```text
Use AGENTS.md and docs/provider-adaptation-checklist.md.

First, explore the official API documentation for this provider. Do not scaffold code yet.

Fill out or summarize the provider adaptation checklist: official docs, OpenAPI availability, authentication, base URL, endpoint paths, pagination, response shape, error shape, metadata/reference endpoints, and candidate raw/workflow commands.

Before proposing endpoint paths, verify them from official docs, OpenAPI/Swagger, sandbox, or official repository.
```
