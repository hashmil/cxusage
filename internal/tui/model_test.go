package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/hashmil/cxusage/internal/usage"
)

func sampleReport() usage.Report {
	rows := []usage.ReportRow{
		{
			Label:    "2026-06-01",
			Sessions: 2,
			Models:   []string{"gpt-5.5", "codex-auto-review"},
			Efforts:  []string{"xhigh", "medium"},
			Modes:    []string{"default", "plan"},
			Usage:    usage.Usage{InputTokens: 1000, CachedInputTokens: 600, OutputTokens: 50, ReasoningOutputTokens: 12, TotalTokens: 1050},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 1000, CachedInputTokens: 600, OutputTokens: 50, ReasoningOutputTokens: 12, TotalTokens: 1050}),
		},
	}
	total := rows[0].Usage
	return usage.Report{
		Title:           "Codex Usage Report - Day",
		Start:           time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		End:             time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
		GroupBy:         "day",
		Rows:            rows,
		Totals:          total,
		TotalCostUSD:    usage.EstimateCostUSD(total),
		SessionsCounted: 2,
		FilesCounted:    1,
		EventsCounted:   1,
	}
}

func sampleReportWithManyRows() usage.Report {
	rows := []usage.ReportRow{
		{
			Label:    "2026-05-20",
			Sessions: 7,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"xhigh"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 1000, CachedInputTokens: 500, OutputTokens: 100, ReasoningOutputTokens: 25, TotalTokens: 1100},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 1000, CachedInputTokens: 500, OutputTokens: 100, ReasoningOutputTokens: 25, TotalTokens: 1100}),
		},
		{
			Label:    "2026-05-21",
			Sessions: 3,
			Models:   []string{"gpt-5.5", "unknown"},
			Efforts:  []string{"medium", "xhigh", "unknown"},
			Modes:    []string{"default", "plan", "unknown"},
			Usage:    usage.Usage{InputTokens: 8000, CachedInputTokens: 6000, OutputTokens: 500, ReasoningOutputTokens: 120, TotalTokens: 8500},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 8000, CachedInputTokens: 6000, OutputTokens: 500, ReasoningOutputTokens: 120, TotalTokens: 8500}),
		},
		{
			Label:    "2026-05-22",
			Sessions: 11,
			Models:   []string{"codex-auto-review", "gpt-5.5"},
			Efforts:  []string{"high", "low"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 3000, CachedInputTokens: 1200, OutputTokens: 250, ReasoningOutputTokens: 80, TotalTokens: 3250},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 3000, CachedInputTokens: 1200, OutputTokens: 250, ReasoningOutputTokens: 80, TotalTokens: 3250}),
		},
	}
	total := usage.Usage{}
	cost := 0.0
	sessions := 0
	for _, row := range rows {
		total = total.Add(row.Usage)
		cost += row.CostUSD
		sessions += row.Sessions
	}
	return usage.Report{
		Title:           "Codex Usage Report - Day",
		Start:           time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		End:             time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
		GroupBy:         "day",
		Rows:            rows,
		Totals:          total,
		TotalCostUSD:    cost,
		SessionsCounted: sessions,
		FilesCounted:    3,
		EventsCounted:   9,
	}
}

func TestAutoThemeUsesDetector(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{
		Theme:       "auto",
		NoAnimation: true,
		DetectTheme: func() ThemeName {
			return ThemeLight
		},
	})
	if model.ThemeName() != ThemeLight {
		t.Fatalf("theme = %q, want light", model.ThemeName())
	}
}

func TestThemeToggleKeySwitchesExplicitTheme(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
	updated := next.(Model)
	if updated.ThemeName() != ThemeLight {
		t.Fatalf("theme = %q, want light", updated.ThemeName())
	}
}

func TestNumberKeysSwitchViews(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Text: "2", Code: '2'}))
	updated := next.(Model)
	if updated.ViewName() != "Trends" {
		t.Fatalf("view = %q, want Trends", updated.ViewName())
	}
}

func TestNarrowWidthViewDoesNotOverflow(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 48, Height: 18})
	updated := next.(Model)
	rendered := updated.View().Content
	for _, line := range strings.Split(rendered, "\n") {
		if len([]rune(stripANSI(line))) > 48 {
			t.Fatalf("line width %d exceeds 48: %q", len([]rune(stripANSI(line))), stripANSI(line))
		}
	}
}

func TestWideWidthTableDoesNotOverflow(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated := next.(Model)
	next, _ = updated.Update(tea.KeyPressMsg(tea.Key{Text: "4", Code: '4'}))
	updated = next.(Model)
	rendered := updated.View().Content
	for _, line := range strings.Split(rendered, "\n") {
		if len([]rune(stripANSI(line))) > 120 {
			t.Fatalf("line width %d exceeds 120: %q", len([]rune(stripANSI(line))), stripANSI(line))
		}
	}
}

func TestLoadingViewUsesWideAnimatedBanner(t *testing.T) {
	model := NewModel(Options{Theme: "dark"})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated := next.(Model)

	first := updated.View().Content
	plain := stripANSI(first)
	if !strings.Contains(plain, " CCCCC  X   X  U   U   SSSS   AAA    GGGG  EEEEE") {
		t.Fatalf("loading view should render wide CXUSAGE banner, got:\n%s", plain)
	}
	if strings.Contains(plain, "╭") || strings.Contains(plain, "╰") {
		t.Fatalf("loading banner should not use a boxed panel, got:\n%s", plain)
	}

	next, _ = updated.Update(animationTickMsg{})
	animated := next.(Model)
	second := animated.View().Content
	if first == second {
		t.Fatalf("loading animation tick should change styled banner output")
	}
	if stripANSI(first) != stripANSI(second) {
		t.Fatalf("loading animation should only change styling, not layout:\nfirst:\n%s\nsecond:\n%s", stripANSI(first), stripANSI(second))
	}
}

func TestLoadingViewFallsBackForNarrowTerminals(t *testing.T) {
	model := NewModel(Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 48, Height: 14})
	updated := next.(Model)
	rendered := updated.View().Content
	plain := stripANSI(rendered)
	if strings.Contains(plain, "CCCCC") {
		t.Fatalf("narrow loading view should not render wide banner, got:\n%s", plain)
	}
	if !strings.Contains(plain, "cxusage") {
		t.Fatalf("narrow loading view should keep compact title, got:\n%s", plain)
	}
	for _, line := range strings.Split(plain, "\n") {
		if len([]rune(line)) > 48 {
			t.Fatalf("line width %d exceeds 48: %q", len([]rune(line)), line)
		}
	}
}

func TestOverviewUsesKPIBoxesAndSolidProgressBars(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := updated.View().Content
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Total tokens") || !strings.Contains(plain, "Estimated cost") {
		t.Fatalf("overview should render KPI boxes, got:\n%s", plain)
	}
	if !hasStandaloneRoundedPanelLine(plain) {
		t.Fatalf("overview should render boxed KPI cards, got:\n%s", plain)
	}
	if !strings.Contains(plain, "█") {
		t.Fatalf("overview should use solid progress bars, got:\n%s", plain)
	}
}

func TestOverviewTokenMetricsUseSemanticColorsAndAlignedValues(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := updated.renderOverview()
	for _, sequence := range []string{
		rgbSequence(128, 201, 144),
		rgbSequence(139, 213, 202),
		rgbSequence(239, 131, 84),
		rgbSequence(198, 208, 245),
	} {
		if !strings.Contains(rendered, sequence) {
			t.Fatalf("overview token rows should use semantic colors; missing %q in:\n%s", sequence, rendered)
		}
	}

	plain := stripANSI(rendered)
	lines := overviewMetricLines(plain)
	if len(lines) != 4 {
		t.Fatalf("expected 4 token metric lines, got %d:\n%s", len(lines), plain)
	}
	values := []string{
		formatInt(sampleReport().Totals.InputTokens),
		formatInt(sampleReport().Totals.CachedInputTokens),
		formatInt(sampleReport().Totals.OutputTokens),
		formatInt(sampleReport().Totals.ReasoningOutputTokens),
	}
	valueColumn := -1
	for i, line := range lines {
		index := strings.Index(line, values[i])
		if index < 0 {
			t.Fatalf("metric line missing value %q: %q", values[i], line)
		}
		if valueColumn < 0 {
			valueColumn = index
			continue
		}
		if index != valueColumn {
			t.Fatalf("metric values should start at column %d, got %d in line %q", valueColumn, index, line)
		}
	}
}

func TestTrendsUsesSingleLineChart(t *testing.T) {
	model := NewLoadedModel(sampleReportWithManyRows(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	next, _ = updated.Update(tea.KeyPressMsg(tea.Key{Text: "2", Code: '2'}))
	updated = next.(Model)
	rendered := stripANSI(updated.View().Content)
	if strings.Contains(rendered, "ASCII graph") {
		t.Fatalf("trends should not render duplicate ASCII graph:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Token trend") {
		t.Fatalf("trends should render a single token trend panel:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Peak") || !strings.Contains(rendered, "Average") {
		t.Fatalf("trends should include summary stats:\n%s", rendered)
	}
	if hasStandaloneRoundedPanelLine(rendered) {
		t.Fatalf("trends should avoid heavy rounded panels:\n%s", rendered)
	}
	if strings.Contains(rendered, "•") {
		t.Fatalf("trends should not use a dot-only scatter chart:\n%s", rendered)
	}
	if !strings.Contains(rendered, "─") && !strings.Contains(rendered, "│") {
		t.Fatalf("trends should use continuous line glyphs:\n%s", rendered)
	}
	if !strings.Contains(rendered, "0 ┼") && !strings.Contains(rendered, "0 ┤") {
		t.Fatalf("trends should start the y-axis at 0:\n%s", rendered)
	}
	if !strings.Contains(rendered, "└──") {
		t.Fatalf("trends should render a bottom horizontal axis:\n%s", rendered)
	}
	if !strings.Contains(rendered, "K ┤") {
		t.Fatalf("trends should abbreviate chart axis labels:\n%s", rendered)
	}
}

func TestWideRowsUseBorderlessLedgerTable(t *testing.T) {
	model := NewLoadedModel(sampleReportWithManyRows(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 160, Height: 30})
	updated := next.(Model)
	table := stripANSI(updated.renderWideRows(updated.report.Rows, 10))
	if strings.Contains(table, "+---") {
		t.Fatalf("table should not use broken ASCII dash borders:\n%s", table)
	}
	for _, line := range strings.Split(table, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			t.Fatalf("table should not use full row boxing: %q\n%s", line, table)
		}
	}
	if !strings.Contains(table, "Total") {
		t.Fatalf("table should keep a total row:\n%s", table)
	}
	if !strings.Contains(table, "─") {
		t.Fatalf("table should use a single thin separator before totals:\n%s", table)
	}
}

func TestBreakdownsUseMeaningfulColorPalette(t *testing.T) {
	model := NewLoadedModel(sampleReportWithManyRows(), Options{Theme: "dark", NoAnimation: true})
	rendered := model.renderBreakdowns()
	for _, sequence := range []string{
		rgbSequence(139, 213, 202),
		rgbSequence(198, 208, 245),
		rgbSequence(239, 131, 84),
		rgbSequence(231, 130, 132),
	} {
		if !strings.Contains(rendered, sequence) {
			t.Fatalf("breakdowns should use semantic bar colors; missing %q in:\n%s", sequence, rendered)
		}
	}
}

func hasStandaloneRoundedPanelLine(value string) bool {
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "╭") || strings.HasPrefix(trimmed, "╰") {
			return true
		}
	}
	return false
}

func rgbSequence(red, green, blue int) string {
	return "\x1b[38;2;" + formatInt(int64(red)) + ";" + formatInt(int64(green)) + ";" + formatInt(int64(blue)) + "m"
}

func overviewMetricLines(value string) []string {
	prefixes := []string{"Input", "Cached input", "Output", "Reasoning output"}
	out := []string{}
	for _, line := range strings.Split(value, "\n") {
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				out = append(out, line)
				break
			}
		}
	}
	return out
}
