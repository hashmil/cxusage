package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashmil/cxusage/internal/usage"
)

func TestParseArgsDefaultsToThirtyDaysAndAutoTheme(t *testing.T) {
	opts, err := ParseArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Timeframe.Last != "30d" {
		t.Fatalf("last = %q, want 30d", opts.Timeframe.Last)
	}
	if opts.GroupBy != "day" {
		t.Fatalf("group by = %q, want day", opts.GroupBy)
	}
	if opts.Theme != "auto" {
		t.Fatalf("theme = %q, want auto", opts.Theme)
	}
}

func TestParseArgsRejectsUnknownTheme(t *testing.T) {
	if _, err := ParseArgs([]string{"--theme", "solarized"}); err == nil {
		t.Fatal("expected invalid theme error")
	}
}

func TestParseArgsSupportsAllTimeAlias(t *testing.T) {
	opts, err := ParseArgs([]string{"--all-time"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Timeframe.All {
		t.Fatalf("all = false, want true")
	}
	if opts.Timeframe.Last != "" {
		t.Fatalf("last = %q, want empty when all-time is selected", opts.Timeframe.Last)
	}
}

func TestParseArgsSupportsCurrentYearAlias(t *testing.T) {
	opts, err := ParseArgs([]string{"--current-year"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Timeframe.CurrentYear {
		t.Fatalf("current year = false, want true")
	}
	if opts.Timeframe.Last != "" {
		t.Fatalf("last = %q, want empty when current-year is selected", opts.Timeframe.Last)
	}
}

func TestHelpTextDocumentsYearAndAllTime(t *testing.T) {
	help := HelpText("cxusage")
	for _, expected := range []string{"--year", "--current-year", "--all", "--all-time"} {
		if !strings.Contains(help, expected) {
			t.Fatalf("help should include %q, got:\n%s", expected, help)
		}
	}
}

func TestJSONOutputUsesCodexHomeFixture(t *testing.T) {
	root := t.TempDir()
	sessions := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessions, "rollout-2026-06-01T00-00-00-11111111-2222-3333-4444-555555555555.jsonl"), []byte(strings.Join([]string{
		`{"timestamp":"2026-06-01T01:00:00Z","type":"turn_context","payload":{"model":"gpt-5.5","collaboration_mode":{"mode":"default","settings":{"reasoning_effort":"xhigh"}}}}`,
		`{"timestamp":"2026-06-01T01:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":600,"output_tokens":50,"reasoning_output_tokens":12,"total_tokens":1050}}}}`,
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	opts, err := ParseArgs([]string{
		"--json",
		"--codex-home", root,
		"--since", "2026-06-01",
		"--until", "2026-06-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	opts.Now = time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)

	report, err := BuildReport(opts)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := WriteJSON(&out, report); err != nil {
		t.Fatal(err)
	}

	var decoded usage.Report
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Totals.TotalTokens != 1050 {
		t.Fatalf("total tokens = %d, want 1050", decoded.Totals.TotalTokens)
	}
	if decoded.Rows[0].Models[0] != "gpt-5.5" || decoded.Rows[0].Efforts[0] != "xhigh" {
		t.Fatalf("metadata not preserved: %+v", decoded.Rows[0])
	}
}
