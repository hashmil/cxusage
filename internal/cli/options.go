package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hashmil/cxusage/internal/usage"
)

const (
	DefaultGroupBy = "day"
	DefaultLast    = "30d"
	DefaultTheme   = "auto"
)

type Options struct {
	Timeframe   usage.TimeframeOptions
	CodexHome   string
	GroupBy     string
	JSON        bool
	NoColor     bool
	NoAnimation bool
	Theme       string
	Now         time.Time
}

func ParseArgs(args []string) (Options, error) {
	opts := Options{
		Timeframe: usage.TimeframeOptions{Last: DefaultLast},
		GroupBy:   DefaultGroupBy,
		Theme:     DefaultTheme,
	}
	fs := flag.NewFlagSet("cxusage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.Timeframe.Last, "last", DefaultLast, "relative timeframe such as 24h, 7d, 30d, 2w, or 3m")
	fs.BoolVar(&opts.Timeframe.Today, "today", false, "show usage from today")
	fs.BoolVar(&opts.Timeframe.Yesterday, "yesterday", false, "show usage from yesterday")
	fs.BoolVar(&opts.Timeframe.Week, "week", false, "show usage from the current week")
	fs.BoolVar(&opts.Timeframe.LastWeek, "last-week", false, "show usage from last week")
	fs.BoolVar(&opts.Timeframe.Month, "month", false, "show usage from the current month")
	fs.BoolVar(&opts.Timeframe.CurrentYear, "year", false, "show usage from the current year")
	fs.BoolVar(&opts.Timeframe.CurrentYear, "current-year", false, "show usage from the current year")
	fs.BoolVar(&opts.Timeframe.All, "all", false, "show all available usage")
	fs.BoolVar(&opts.Timeframe.All, "all-time", false, "show all available usage")
	fs.StringVar(&opts.Timeframe.Since, "since", "", "inclusive start date in YYYY-MM-DD")
	fs.StringVar(&opts.Timeframe.Until, "until", "", "inclusive end date in YYYY-MM-DD")
	fs.StringVar(&opts.CodexHome, "codex-home", "", "path to CODEX_HOME; defaults to ~/.codex")
	fs.StringVar(&opts.GroupBy, "group-by", DefaultGroupBy, "group rows by day, week, month, session, model, effort, mode, or model-effort")
	fs.BoolVar(&opts.JSON, "json", false, "print a machine-readable JSON report instead of launching the TUI")
	fs.BoolVar(&opts.NoColor, "no-color", false, "disable color styling")
	fs.BoolVar(&opts.NoAnimation, "no-animation", false, "disable spinner and count-up animation")
	fs.StringVar(&opts.Theme, "theme", DefaultTheme, "theme: auto, dark, or light")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	opts.GroupBy = strings.TrimSpace(opts.GroupBy)
	opts.Theme = strings.ToLower(strings.TrimSpace(opts.Theme))
	switch opts.Theme {
	case "auto", "dark", "light":
	default:
		return opts, fmt.Errorf("theme must be auto, dark, or light")
	}
	if opts.GroupBy == "" {
		opts.GroupBy = DefaultGroupBy
	}
	if hasExplicitTimeframe(opts.Timeframe) {
		opts.Timeframe.Last = ""
	}
	return opts, nil
}

func hasExplicitTimeframe(options usage.TimeframeOptions) bool {
	return options.Today ||
		options.Yesterday ||
		options.Week ||
		options.LastWeek ||
		options.Month ||
		options.CurrentYear ||
		options.All ||
		options.Since != "" ||
		options.Until != ""
}

func BuildReport(opts Options) (usage.Report, error) {
	timeframe := opts.Timeframe
	timeframe.Now = opts.Now
	start, end, label, err := usage.ResolveTimeframe(timeframe)
	if err != nil {
		return usage.Report{}, err
	}
	return usage.BuildReport(
		usage.DefaultRoots(opts.CodexHome),
		start,
		end,
		opts.GroupBy,
		"Codex Usage Report - "+label,
	)
}

func WriteJSON(w io.Writer, report usage.Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func HelpText(command string) string {
	return fmt.Sprintf(`cxusage - interactive Codex usage explorer

Usage:
  %[1]s [flags]
  %[1]s --json [flags]

Timeframe:
  --last 30d           relative range; supports h, d, w, m
  --today              today only
  --yesterday          yesterday only
  --week               current week
  --last-week          previous week
  --month              current month
  --year               current year
  --current-year       current year
  --all                all available local logs
  --all-time           all available local logs
  --since YYYY-MM-DD   custom start date
  --until YYYY-MM-DD   custom end date, inclusive

Output:
  --group-by day|week|month|session|model|effort|mode|model-effort
  --json
  --theme auto|dark|light
  --no-color
  --no-animation
  --codex-home PATH
`, command)
}
