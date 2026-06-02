# cxusage

Interactive local Codex usage explorer for the terminal.

`cxusage` scans local Codex session logs and shows token usage, model/effort/mode breakdowns, terminal charts, and estimated cost. It is local-only: no network calls, no telemetry, and no cloud account access.

## Features

- Interactive Bubble Tea TUI by default
- Overview, trends, breakdowns, sessions, and help views
- Dark, light, and auto theme detection on macOS
- Terminal-width-aware rendering with compact fallback
- Multi-line model, effort, and mode cells
- Single terminal chart view with compact axis labels
- JSON output for scripts
- Default timeframe: last 30 days
- Estimated cost column using GPT-5.5 pricing constants

## Install

Requires Go 1.25 or newer.

```sh
git clone https://github.com/hashmil/cxusage.git
cd cxusage
go install ./cmd/cxusage
```

If `~/go/bin` is on your `PATH`, run:

```sh
cxusage
```

If it is not on your `PATH`, run:

```sh
~/go/bin/cxusage
```

Or build a local binary:

```sh
go build -o bin/cxusage ./cmd/cxusage
./bin/cxusage
```

## Usage

Launch the interactive TUI:

```sh
cxusage
```

Show JSON for the last 7 days:

```sh
cxusage --json --last 7d
```

Group by model and effort:

```sh
cxusage --group-by model-effort
```

Use a custom Codex home:

```sh
cxusage --codex-home /path/to/.codex
```

Force a theme:

```sh
cxusage --theme light
cxusage --theme dark
cxusage --theme auto
```

Disable color or animation:

```sh
cxusage --no-color
cxusage --no-animation
```

## Flags

```text
--last 30d           relative range; supports h, d, w, m
--today              today only
--yesterday          yesterday only
--week               current week
--last-week          previous week
--month              current month
--since YYYY-MM-DD   custom start date
--until YYYY-MM-DD   custom end date, inclusive
--group-by VALUE     day, week, month, session, model, effort, mode, model-effort
--json               print JSON instead of launching the TUI
--theme VALUE        auto, dark, or light
--no-color           disable color styling
--no-animation       disable spinner and count-up animation
--codex-home PATH    defaults to ~/.codex
```

## TUI Keys

```text
q / ctrl+c   quit
r            refresh logs
1-5          switch views
left/right   switch views
j/k          move selection
f            cycle timeframe
g            cycle grouping
/            filter rows
a            toggle animation
t            toggle dark/light theme
?            help
```

## What It Reads

`cxusage` reads:

```text
~/.codex/sessions
~/.codex/archived_sessions
```

It looks for `rollout-*.jsonl` files, deduplicates active and archived copies by session id, computes token deltas from cumulative `token_count.total_token_usage` records, and attributes each token delta to the latest preceding `turn_context`.

Model, reasoning effort, and collaboration mode are shown when they are present in the local logs. Service tier is intentionally omitted because historical local logs do not reliably expose it.

## Cost Estimate

The cost number is only an estimate. It uses fixed GPT-5.5 constants:

```text
input:        $5.00 / 1M uncached input tokens
cached input: $0.50 / 1M cached input tokens
output:       $30.00 / 1M output tokens
```

Reasoning output tokens are shown separately, but cost is estimated from the output token total available in the logs.

## Development

```sh
go test ./...
go vet ./...
go build -o bin/cxusage ./cmd/cxusage
```

Run against fixture data:

```sh
cxusage --codex-home ./testdata/codex-home --json
```

## Privacy

`cxusage` is a local CLI. It does not send usage data anywhere. JSON output can include local session-derived metadata, so treat exported files as private unless you have reviewed them.
