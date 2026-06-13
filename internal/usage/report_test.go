package usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tokenEvent(t *testing.T, timestamp string, total Usage, last Usage) string {
	t.Helper()
	payload := map[string]any{
		"timestamp": timestamp,
		"type":      "event_msg",
		"payload": map[string]any{
			"type": "token_count",
			"info": map[string]any{
				"total_token_usage": total,
				"last_token_usage":  last,
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func turnContext(t *testing.T, timestamp, model, effort, mode string) string {
	t.Helper()
	payload := map[string]any{
		"timestamp": timestamp,
		"type":      "turn_context",
		"payload": map[string]any{
			"model":  model,
			"effort": effort,
			"collaboration_mode": map[string]any{
				"mode": mode,
				"settings": map[string]any{
					"model":            model,
					"reasoning_effort": effort,
				},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeRollout(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(joinLines(lines...)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func joinLines(lines ...string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}

func TestBuildReportUsesCumulativeDeltaWithinWindow(t *testing.T) {
	dir := t.TempDir()
	writeRollout(t, dir, "rollout-2026-06-01T00-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl",
		tokenEvent(t, "2026-05-24T23:00:00Z",
			Usage{InputTokens: 100, CachedInputTokens: 40, OutputTokens: 10, ReasoningOutputTokens: 3, TotalTokens: 110},
			Usage{InputTokens: 100, CachedInputTokens: 40, OutputTokens: 10, ReasoningOutputTokens: 3, TotalTokens: 110},
		),
		tokenEvent(t, "2026-05-25T12:00:00Z",
			Usage{InputTokens: 250, CachedInputTokens: 160, OutputTokens: 30, ReasoningOutputTokens: 8, TotalTokens: 280},
			Usage{InputTokens: 150, CachedInputTokens: 120, OutputTokens: 20, ReasoningOutputTokens: 5, TotalTokens: 170},
		),
	)

	report, err := BuildReport([]string{dir}, mustTime("2026-05-25T00:00:00Z"), mustTime("2026-05-26T00:00:00Z"), "day", "")
	if err != nil {
		t.Fatal(err)
	}

	if report.Totals.TotalTokens != 170 {
		t.Fatalf("total tokens = %d, want 170", report.Totals.TotalTokens)
	}
	if report.Totals.InputTokens != 150 || report.Totals.CachedInputTokens != 120 || report.Totals.OutputTokens != 20 || report.Totals.ReasoningOutputTokens != 5 {
		t.Fatalf("unexpected totals: %+v", report.Totals)
	}
	if len(report.Rows) != 1 || report.Rows[0].Label != "2026-05-25" {
		t.Fatalf("unexpected rows: %+v", report.Rows)
	}
}

func TestBuildReportDeduplicatesArchivedCopyBySessionID(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, "sessions")
	archived := filepath.Join(root, "archived_sessions")
	if err := os.MkdirAll(active, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archived, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "rollout-2026-06-01T00-00-00-11111111-2222-3333-4444-555555555555.jsonl"
	line := tokenEvent(t, "2026-06-01T01:00:00Z",
		Usage{InputTokens: 1000, CachedInputTokens: 800, OutputTokens: 50, ReasoningOutputTokens: 10, TotalTokens: 1050},
		Usage{InputTokens: 1000, CachedInputTokens: 800, OutputTokens: 50, ReasoningOutputTokens: 10, TotalTokens: 1050},
	)
	writeRollout(t, active, name, line)
	writeRollout(t, archived, name, line)

	report, err := BuildReport([]string{active, archived}, mustTime("2026-06-01T00:00:00Z"), mustTime("2026-06-02T00:00:00Z"), "day", "")
	if err != nil {
		t.Fatal(err)
	}

	if report.Totals.TotalTokens != 1050 {
		t.Fatalf("total tokens = %d, want 1050", report.Totals.TotalTokens)
	}
	if report.SessionsCounted != 1 {
		t.Fatalf("sessions counted = %d, want 1", report.SessionsCounted)
	}
}

func TestBuildReportAttributesTokenDeltasToLatestTurnContext(t *testing.T) {
	dir := t.TempDir()
	writeRollout(t, dir, "rollout-2026-06-01T00-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl",
		turnContext(t, "2026-06-01T00:01:00Z", "gpt-5.4", "high", "plan"),
		tokenEvent(t, "2026-06-01T00:02:00Z",
			Usage{InputTokens: 100, CachedInputTokens: 20, OutputTokens: 10, ReasoningOutputTokens: 3, TotalTokens: 110},
			Usage{InputTokens: 100, CachedInputTokens: 20, OutputTokens: 10, ReasoningOutputTokens: 3, TotalTokens: 110},
		),
		turnContext(t, "2026-06-01T00:03:00Z", "gpt-5.5", "xhigh", "default"),
		tokenEvent(t, "2026-06-01T00:04:00Z",
			Usage{InputTokens: 300, CachedInputTokens: 100, OutputTokens: 30, ReasoningOutputTokens: 8, TotalTokens: 330},
			Usage{InputTokens: 200, CachedInputTokens: 80, OutputTokens: 20, ReasoningOutputTokens: 5, TotalTokens: 220},
		),
	)

	report, err := BuildReport([]string{dir}, mustTime("2026-06-01T00:00:00Z"), mustTime("2026-06-02T00:00:00Z"), "model-effort", "")
	if err != nil {
		t.Fatal(err)
	}

	rows := map[string]ReportRow{}
	for _, row := range report.Rows {
		rows[row.Label] = row
	}
	if rows["gpt-5.4 / high"].Usage.TotalTokens != 110 {
		t.Fatalf("gpt-5.4 high total = %d, want 110", rows["gpt-5.4 / high"].Usage.TotalTokens)
	}
	if rows["gpt-5.5 / xhigh"].Usage.TotalTokens != 220 {
		t.Fatalf("gpt-5.5 xhigh total = %d, want 220", rows["gpt-5.5 / xhigh"].Usage.TotalTokens)
	}
}

func TestMalformedJSONLIsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeRollout(t, dir, "rollout-2026-06-01T00-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl",
		"{bad json",
		tokenEvent(t, "2026-06-01T00:02:00Z",
			Usage{InputTokens: 100, OutputTokens: 10, TotalTokens: 110},
			Usage{InputTokens: 100, OutputTokens: 10, TotalTokens: 110},
		),
	)

	report, err := BuildReport([]string{dir}, mustTime("2026-06-01T00:00:00Z"), mustTime("2026-06-02T00:00:00Z"), "day", "")
	if err != nil {
		t.Fatal(err)
	}

	if report.Totals.TotalTokens != 110 {
		t.Fatalf("total tokens = %d, want 110", report.Totals.TotalTokens)
	}
}

func TestOversizedIrrelevantJSONLLineIsIgnored(t *testing.T) {
	dir := t.TempDir()
	oversized := `{"timestamp":"2026-06-01T00:01:00Z","type":"response_item","payload":{"text":"` + strings.Repeat("x", 17*1024*1024) + `"}}`
	writeRollout(t, dir, "rollout-2026-06-01T00-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl",
		oversized,
		tokenEvent(t, "2026-06-01T00:02:00Z",
			Usage{InputTokens: 100, OutputTokens: 10, TotalTokens: 110},
			Usage{InputTokens: 100, OutputTokens: 10, TotalTokens: 110},
		),
	)

	report, err := BuildReport([]string{dir}, mustTime("2026-06-01T00:00:00Z"), mustTime("2026-06-02T00:00:00Z"), "day", "")
	if err != nil {
		t.Fatal(err)
	}

	if report.Totals.TotalTokens != 110 {
		t.Fatalf("total tokens = %d, want 110", report.Totals.TotalTokens)
	}
}

func TestEstimateCostUsesGPT55RatesByDefault(t *testing.T) {
	usage := Usage{InputTokens: 2_000_000, CachedInputTokens: 1_500_000, OutputTokens: 100_000, ReasoningOutputTokens: 50_000, TotalTokens: 2_100_000}
	if got := EstimateCostUSD(usage); got != 6.25 {
		t.Fatalf("cost = %.2f, want 6.25", got)
	}
}

func TestEstimateCostUsesModelSpecificRates(t *testing.T) {
	usage := Usage{InputTokens: 2_000_000, CachedInputTokens: 1_500_000, OutputTokens: 100_000, ReasoningOutputTokens: 50_000, TotalTokens: 2_100_000}
	if got := EstimateCostUSDForModel("gpt-5.4", usage); got != 3.125 {
		t.Fatalf("gpt-5.4 cost = %.4f, want 3.125", got)
	}
	if got := EstimateCostUSDForModel("codex-mini-latest", usage); got != 1.9125 {
		t.Fatalf("codex-mini-latest cost = %.4f, want 1.9125", got)
	}
}

func TestBuildReportAggregatesModelSpecificCosts(t *testing.T) {
	dir := t.TempDir()
	writeRollout(t, dir, "rollout-2026-06-01T00-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl",
		turnContext(t, "2026-06-01T00:01:00Z", "gpt-5.5", "xhigh", "default"),
		tokenEvent(t, "2026-06-01T00:02:00Z",
			Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000, TotalTokens: 2_000_000},
			Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000, TotalTokens: 2_000_000},
		),
		turnContext(t, "2026-06-01T00:03:00Z", "gpt-5-mini", "medium", "default"),
		tokenEvent(t, "2026-06-01T00:04:00Z",
			Usage{InputTokens: 2_000_000, OutputTokens: 2_000_000, TotalTokens: 4_000_000},
			Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000, TotalTokens: 2_000_000},
		),
	)

	report, err := BuildReport([]string{dir}, mustTime("2026-06-01T00:00:00Z"), mustTime("2026-06-02T00:00:00Z"), "day", "")
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(report.Rows))
	}
	if got := report.Rows[0].CostUSD; got != 37.25 {
		t.Fatalf("row cost = %.2f, want 37.25", got)
	}
	if report.TotalCostUSD != report.Rows[0].CostUSD {
		t.Fatalf("total cost = %.2f, row cost = %.2f", report.TotalCostUSD, report.Rows[0].CostUSD)
	}
}

func TestDefaultTimeframeIsLast30Days(t *testing.T) {
	now := mustTime("2026-06-01T12:00:00Z")
	start, end, label, err := ResolveTimeframe(TimeframeOptions{Last: "30d", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if label != "Last 30d" {
		t.Fatalf("label = %q, want Last 30d", label)
	}
	if !end.Equal(now) {
		t.Fatalf("end = %s, want %s", end, now)
	}
	if end.Sub(start) != 30*24*time.Hour {
		t.Fatalf("duration = %s, want 720h", end.Sub(start))
	}
}

func mustTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}
