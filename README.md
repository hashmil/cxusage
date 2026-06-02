# cxusage

[![Build](https://github.com/hashmil/cxusage/actions/workflows/build.yml/badge.svg)](https://github.com/hashmil/cxusage/actions/workflows/build.yml)

Interactive local Codex usage explorer for the terminal.

`cxusage` scans local Codex session logs and shows token usage, model/effort/mode breakdowns, terminal charts, and estimated cost. It is local-only: no network calls, no telemetry, and no cloud account access.

This is an unofficial open source tool for enterprise Codex users who can run Codex locally but cannot use cloud Codex because it is not enabled, approved, or available in their environment. It gives teams a local way to understand usage from the logs that already exist on their machine.

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

## Who It Is For

`cxusage` is useful when:

- Your organization permits local Codex usage but has not enabled cloud Codex.
- You need local visibility into sessions, token usage, model/effort/mode mix, and rough cost.
- You cannot rely on a hosted dashboard because of enterprise policy, network limits, or rollout timing.
- You want a private, offline-first usage viewer that reads only local `~/.codex` logs.

## Install

There are three practical ways to install `cxusage`.

### Option 1: Go Install

Use this if you already have Go 1.25 or newer installed.

```sh
go install github.com/hashmil/cxusage/cmd/cxusage@latest
```

Go installs the binary into `$(go env GOPATH)/bin`, which is usually `~/go/bin`.

If `~/go/bin` is already on your `PATH`, run:

```sh
cxusage
```

If it is not on your `PATH`, run it directly:

```sh
~/go/bin/cxusage
```

To make `cxusage` available from any terminal, add Go's bin directory to your shell profile:

```sh
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Then run:

```sh
cxusage
```

### Option 2: Clone And Build From Source

Use this if you want to inspect or modify the source locally.

```sh
git clone https://github.com/hashmil/cxusage.git
cd cxusage
go install ./cmd/cxusage
```

Or build a binary in the repo:

```sh
go build -o bin/cxusage ./cmd/cxusage
./bin/cxusage
```

### Option 3: Download A Prebuilt Binary

Use this if you do not want to install Go.

1. Open the latest successful [Build workflow](https://github.com/hashmil/cxusage/actions/workflows/build.yml).
2. Scroll to **Artifacts**.
3. Download the artifact for your platform:
   - `cxusage-darwin-arm64` for Apple Silicon Macs
   - `cxusage-linux-amd64` for Intel/AMD Linux
   - `cxusage-linux-arm64` for ARM Linux
   - `cxusage-windows-amd64` for Windows
4. Unzip the artifact.

On macOS or Linux, move the binary into a directory on your `PATH`:

```sh
chmod +x cxusage
mkdir -p ~/.local/bin
mv cxusage ~/.local/bin/cxusage
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
cxusage
```

On Windows PowerShell, unzip the artifact and run:

```powershell
.\cxusage.exe
```

For regular Windows use, move `cxusage.exe` into a folder that is already on your `PATH`, or add its folder to your user `PATH`.

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

## GitHub Actions

The repository includes a `Build` workflow that:

- installs the Go version declared in `go.mod`
- runs `go test ./...`
- runs `go vet ./...`
- builds `./cmd/cxusage`
- uploads platform binaries as workflow artifacts

## Privacy

`cxusage` is a local CLI. It does not send usage data anywhere. JSON output can include local session-derived metadata, so treat exported files as private unless you have reviewed them.
