# Future Skill Draft

Use this Skill when an agent needs current humanitarian operational data from HDX HAPI through the local `hapi` CLI.

## Protocol

1. Resolve the requested country or location.

```bash
hapi metadata locations --name "Sudan" --format json
```

2. Check data availability for the resolved ISO-3 location code.

```bash
hapi metadata availability --location-code SDN --format json
```

3. Query the relevant curated workflow or raw endpoint.

```bash
hapi workflow wash-3w --country "Nigeria" --admin1-name "Yobe" --format json
hapi workflow funding --country "South Sudan" --format json
hapi workflow food-security --country "Mozambique" --ipc-phase 3+ --format json
hapi workflow humanitarian-needs --country "DRC" --sector "Water Sanitation Hygiene" --status INN --format json
```

4. Preserve returned records. Do not aggregate totals inside the Skill unless a separate methodology is explicitly requested and documented.

5. Cite available source fields. Prefer `resource_hdx_id`, `dataset_hdx_stub`, `dataset_hdx_title`, `reference_period_start`, `reference_period_end`, and `hapi_updated_date` when present.

6. Fall back to authoritative web sources only when HAPI is missing, stale, too coarse, or does not cover the requested indicator. Clearly label non-HAPI figures.

## Agent Notes

- JSON is the default contract.
- `--format table` is for human inspection only.
- Use `--fields` to reduce token volume.
- Use `--all-pages` only when the task really needs complete records.
- If a query returns exit code `4`, report that HAPI returned no data for the resolved query rather than inventing a value.
