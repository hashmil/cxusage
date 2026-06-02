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

func TestOverviewUsesRoundedPanelsAndSolidProgressBars(t *testing.T) {
	model := NewLoadedModel(sampleReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := updated.View().Content
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "╭") || !strings.Contains(plain, "╰") {
		t.Fatalf("overview should use rounded panels, got:\n%s", plain)
	}
	if !strings.Contains(plain, "█") {
		t.Fatalf("overview should use solid progress bars, got:\n%s", plain)
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
	if !strings.Contains(rendered, "K ┤") {
		t.Fatalf("trends should abbreviate chart axis labels:\n%s", rendered)
	}
}

func TestTotalRowRendersMetadataInsideTable(t *testing.T) {
	model := NewLoadedModel(sampleReportWithManyRows(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 160, Height: 30})
	updated := next.(Model)
	table := stripANSI(updated.renderWideRows(updated.report.Rows, 10))
	for _, line := range strings.Split(table, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "|") {
			t.Fatalf("table content escaped cell borders: %q\n%s", line, table)
		}
	}
}
