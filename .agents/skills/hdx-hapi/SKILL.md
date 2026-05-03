---
name: hdx-hapi
description: Use when the user needs current humanitarian operational data from HDX HAPI through the local hapi CLI, including endpoint discovery, data availability, operational presence, funding, food security, displacement, refugees, population, conflict events, humanitarian needs, and source metadata citations.
---

# HDX HAPI

Use this Skill when an agent needs current humanitarian operational data from HDX HAPI through the local `hapi` CLI.

## Protocol

1. Discover endpoints if the requested domain is unclear.

```bash
hapi list-endpoints --format table
```

2. Resolve the requested country or location.

```bash
hapi metadata locations --name "Sudan" --format json
```

3. Check data availability for the resolved ISO-3 location code.

```bash
hapi metadata availability --location-code SDN --format json
```

Use metadata helpers when subnational or sector filters are needed:

```bash
hapi metadata sectors --name "Water" --format json
hapi metadata admin1 --location-code SDN --format json
hapi metadata admin2 --location-code SDN --admin1-code SD01 --format json
```

4. Query the relevant curated workflow or raw endpoint.

```bash
hapi workflow country-overview --country "Sudan" --format json
hapi workflow wash-3w --country "Nigeria" --admin1-name "Yobe" --format json
hapi workflow funding --country "South Sudan" --format json
hapi workflow food-security --country "Mozambique" --ipc-phase 3+ --format json
hapi workflow displacement --country "Sudan" --type idps --format json
hapi workflow humanitarian-needs --country "DRC" --sector "Water Sanitation Hygiene" --status INN --format json
hapi workflow refugees --country "Uganda" --format json
hapi workflow population --country "Nepal" --admin-level 1 --format json
hapi workflow conflict-events --country "Sudan" --event-type battles --start-date 2026-01-01 --format json
```

For domains without a workflow, use raw endpoint access:

```bash
hapi get coordination-context/national-risk --param location_code=SDN --format json
hapi get food-security-nutrition-poverty/food-prices-market-monitor --param location_code=SDN --format json
hapi get climate/hazards-rainfall --param location_code=SDN --format json
```

5. Preserve returned records. Do not aggregate totals inside the Skill unless a separate methodology is explicitly requested and documented. For `conflict-events`, do not compute risk scores, total fatalities, or trend labels unless the user asks for an analysis and the method is explicit.

6. Cite available source fields. Prefer `resource_hdx_id`, `dataset_hdx_stub`, `dataset_hdx_title`, `reference_period_start`, `reference_period_end`, and `hapi_updated_date` when present.

7. Fall back to authoritative web sources only when HAPI is missing, stale, too coarse, or does not cover the requested indicator. Clearly label non-HAPI figures.

## Agent Notes

- JSON is the default contract.
- `--format table` is for human inspection only.
- Use `--fields` to reduce token volume.
- Use `--all-pages` only when the task really needs complete records.
- `workflow refugees --country` treats the country as country of asylum.
- `workflow conflict-events` intentionally does not expose HRP/GHO filters; use raw `hapi get` for those portfolio filters.
- Exit codes: `0` success; `1` usage/config; `2` HAPI validation/bad request; `3` network/upstream/malformed response; `4` no data; `5` partial data.
- If a query returns exit code `4`, report that HAPI returned no data for the resolved query rather than inventing a value.
