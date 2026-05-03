# Agent Contract

`hapi` treats stdout as data and stderr as diagnostics. In JSON mode, logs and progress text must never be written to stdout.

## Success Envelope

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

`data` preserves HAPI records as returned, apart from optional `--fields` projection. The CLI does not aggregate humanitarian figures.

## Error Envelope

The executable writes a JSON error envelope before exiting non-zero:

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

## Exit Codes

- `0`: success
- `1`: CLI usage or configuration error
- `2`: HAPI validation or bad request
- `3`: network, timeout, or malformed upstream response
- `4`: no data returned
- `5`: partial data with one or more page failures

## Safe Agent Usage

Agents should:

- Use `hapi list-endpoints` to discover supported raw endpoint paths.
- Resolve a country through `hapi metadata locations`.
- Check `hapi metadata availability` before substantive retrieval.
- Prefer `location_code` over free-text location filters once resolved.
- Prefer curated workflows for supported domains, including `wash-3w`, `funding`, `food-security`, `displacement`, `humanitarian-needs`, `refugees`, `population`, and `conflict-events`.
- Use `--limit`, `--offset`, or `--all-pages` deliberately.
- Set credentials via environment variables such as `HAPI_APP_IDENTIFIER` or the compatibility alias `HDX_APP_IDENTIFIER`; never write identifiers into project repos.
- Cite HAPI source metadata fields such as `resource_hdx_id`, `dataset_hdx_stub`, and reference periods when present.
- Avoid manually aggregating figures unless a separate, explicit analytical method is documented outside this CLI. In particular, `conflict-events` returns ACLED-derived event/fatality records and should not be treated as a risk score or summed across administrative levels without an explicit method.
- Treat table output as human-facing and JSON output as the stable contract.
