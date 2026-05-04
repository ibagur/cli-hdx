# hapi

`hapi` is an agent-first Go CLI for HDX HAPI, packaged as `hdx-hapi-cli`.

It is intentionally a standalone shell tool, not an MCP server. Agents can call it directly, receive stable machine-readable output, and later wrap it in a Skill or MCP layer if useful.

## Install

### macOS

Install Go (if not already installed), then:

```bash
go install github.com/ibagur/cli-hdx/cmd/hapi@latest
```

The binary lands in `~/go/bin`. Add it to your PATH if needed:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

### Linux

```bash
go install github.com/ibagur/cli-hdx/cmd/hapi@latest
```

Add `~/go/bin` to your PATH:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

### Windows

**Step 1 — Install Go** from [go.dev/dl](https://go.dev/dl/). Use the `.msi` installer; it sets PATH automatically.

**Step 2 — Install hapi** from PowerShell or Command Prompt:

```powershell
go install github.com/ibagur/cli-hdx/cmd/hapi@latest
```

**Step 3 — Verify the binary is reachable.** Go places binaries in `%USERPROFILE%\go\bin`. Confirm it is on your PATH:

```powershell
hapi --help
```

If the command is not found, add the directory manually:

```powershell
# PowerShell (current session only)
$env:PATH += ";$env:USERPROFILE\go\bin"

# To persist across sessions, set via System Properties → Environment Variables
# and add %USERPROFILE%\go\bin to the User PATH entry
```

**Step 4 — Set your app identifier** (see Configuration below):

```powershell
$env:HAPI_APP_IDENTIFIER = "your-app-identifier"
```

To persist it, add it to your user environment variables via System Properties, or add the line above to your PowerShell profile (`$PROFILE`).

### Build from source

```bash
git clone https://github.com/ibagur/cli-hdx.git
cd cli-hdx
go build -o hapi ./cmd/hapi
```

## Configuration

Generate an app identifier from a name and email:

```bash
hapi auth init --app-name "my-agent" --email "me@example.org"
```

Or set it directly via environment variable:

```bash
# macOS / Linux
export HAPI_APP_IDENTIFIER="..."

# Windows PowerShell
$env:HAPI_APP_IDENTIFIER = "..."
```

Resolution precedence:

1. Explicit CLI flag
2. Environment variable
3. Config file at `~/.config/hapi/config.toml`
4. Built-in default

Supported environment variables:

- `HAPI_APP_IDENTIFIER` (or `HDX_APP_IDENTIFIER`)
- `HAPI_BASE_URL` (or `HDX_BASE_URL`), default `https://hapi.humdata.org/api`
- `HAPI_API_VERSION` (or `HDX_API_VERSION`), default `v2`
- `HAPI_TIMEOUT` (or `HDX_TIMEOUT`), default `30`

Minimal config file (`~/.config/hapi/config.toml`):

```toml
app_identifier = "..."
base_url = "https://hapi.humdata.org/api"
api_version = "v2"
limit = 1000
```

## Installing the Skill for AI Agents

The `hdx-hapi` skill tells an AI agent how to use `hapi` to answer humanitarian data questions. Install `hapi` first, then add the skill to your agent platform.

### Claude Code (with Superpowers plugin)

Place the skill in your global Claude skills directory so it is available in any project:

```bash
mkdir -p ~/.claude/skills/hdx-hapi
curl -fsSL https://raw.githubusercontent.com/ibagur/cli-hdx/main/.agents/skills/hdx-hapi/SKILL.md \
  -o ~/.claude/skills/hdx-hapi/SKILL.md
```

Windows (PowerShell):

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.claude\skills\hdx-hapi"
Invoke-WebRequest `
  -Uri "https://raw.githubusercontent.com/ibagur/cli-hdx/main/.agents/skills/hdx-hapi/SKILL.md" `
  -OutFile "$env:USERPROFILE\.claude\skills\hdx-hapi\SKILL.md"
```

The skill is picked up automatically on the next Claude Code session. No restart needed.

For project-local installation (only available within that project), place the file at `.agents/skills/hdx-hapi/SKILL.md` in the project root instead.

### Codex CLI

Place the skill in your project's agent skills directory:

```bash
mkdir -p .agents/skills/hdx-hapi
curl -fsSL https://raw.githubusercontent.com/ibagur/cli-hdx/main/.agents/skills/hdx-hapi/SKILL.md \
  -o .agents/skills/hdx-hapi/SKILL.md
```

Codex picks up skills from `.agents/skills/` automatically.

### Claude Desktop

Claude Desktop does not load skill files directly. Copy the skill content into your custom system prompt:

1. Open Claude Desktop → Settings → Custom System Prompt
2. Open the skill file: `.agents/skills/hdx-hapi/SKILL.md`
3. Paste the full content (everything after the frontmatter `---` block) into the system prompt field

You can also clone the repo and reference the path locally so you can keep the prompt in sync with updates.

### ChatGPT (Custom GPT or System Prompt)

1. Go to ChatGPT → Explore GPTs → Create (or open an existing custom GPT)
2. Under **Instructions**, paste the content of `SKILL.md` (after the frontmatter block)
3. Ensure `hapi` is available in your environment if using Code Interpreter, or instruct the GPT to emit `hapi` shell commands for you to run locally and paste back the output

For direct ChatGPT sessions without a custom GPT, paste the skill content at the start of a new conversation as a system-level instruction.

## Examples

### CLI usage

Discover available endpoints:

```bash
hapi list-endpoints --format table
```

Look up a country and check what data is available:

```bash
hapi metadata locations --name Sudan --format json
hapi metadata availability --location-code SDN --format json
```

Curated workflows — the fastest path to an answer:

```bash
hapi workflow country-overview --country Sudan --format json
hapi workflow wash-3w --country Nigeria --admin1-name Yobe --format json
hapi workflow funding --country "South Sudan"
hapi workflow food-security --country Mozambique --ipc-phase 3+ --admin-level 1
hapi workflow displacement --country Sudan --type idps --format json
hapi workflow humanitarian-needs --country DRC --sector "Water Sanitation Hygiene" --format json
hapi workflow refugees --country Uganda --format json
hapi workflow population --country Nepal --admin-level 1 --format json
hapi workflow conflict-events --country Sudan --event-type battles --start-date 2026-01-01 --format json
```

Raw endpoint access for domains without a workflow:

```bash
hapi get coordination-context/national-risk --param location_code=SDN --format json
hapi get food-security-nutrition-poverty/food-prices-market-monitor --param location_code=SDN --format json
hapi get climate/hazards-rainfall --param location_code=SDN --format json
```

Limit fields to reduce output size:

```bash
hapi workflow conflict-events --country Sudan --fields location_code,admin1_name,event_type,fatalities --format csv
```

Fetch all pages (use with care on large datasets):

```bash
hapi workflow refugees --country Ethiopia --all-pages --format jsonl
```

### Skill usage (asking an agent)

Once the skill is installed, you can ask your agent natural-language questions. The agent resolves the location, queries HAPI, and returns sourced results.

> "What is the current food security situation in Sudan by district?"

> "How many IDPs are recorded in Ethiopia, broken down by admin1?"

> "Show me WASH operational presence in Yobe state, Nigeria."

> "What funding has been reported for South Sudan this year?"

> "List conflict events in Sudan since January 2026 — battles only."

> "What humanitarian needs data is available for DRC in the WASH sector?"

The agent will cite `resource_hdx_id`, `dataset_hdx_title`, and reference period fields from each response so you can trace the data back to the source dataset on HDX.

JSON is the default output format. Use `--format table` when inspecting data interactively.

## Exit Codes

- `0`: success
- `1`: CLI usage or configuration error
- `2`: HAPI validation or bad request
- `3`: network, timeout, malformed response, or provider unavailable
- `4`: no data returned
- `5`: partial data with one or more page failures

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

## Testing

Unit tests do not require network access. Live smoke testing is gated by `HAPI_APP_IDENTIFIER`:

```bash
go test ./...
HAPI_APP_IDENTIFIER="..." go test ./internal/integration -run Live
```
