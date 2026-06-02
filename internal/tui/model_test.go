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

func sampleCalendarReport() usage.Report {
	rows := []usage.ReportRow{
		{
			Label:    "2026-05-25",
			Sessions: 1,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"medium"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 900, CachedInputTokens: 200, OutputTokens: 100, TotalTokens: 1000},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 900, CachedInputTokens: 200, OutputTokens: 100, TotalTokens: 1000}),
		},
		{
			Label:    "2026-06-01",
			Sessions: 3,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"xhigh"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 3800, CachedInputTokens: 1200, OutputTokens: 400, TotalTokens: 4200},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 3800, CachedInputTokens: 1200, OutputTokens: 400, TotalTokens: 4200}),
		},
		{
			Label:    "2026-06-03",
			Sessions: 2,
			Models:   []string{"codex-auto-review"},
			Efforts:  []string{"high"},
			Modes:    []string{"plan"},
			Usage:    usage.Usage{InputTokens: 1600, CachedInputTokens: 800, OutputTokens: 250, TotalTokens: 1850},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 1600, CachedInputTokens: 800, OutputTokens: 250, TotalTokens: 1850}),
		},
	}
	return reportFromRows(rows, "day", time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC))
}

func sampleMonthReport() usage.Report {
	rows := []usage.ReportRow{
		{
			Label:    "2026-04",
			Sessions: 2,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"medium"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 1000, OutputTokens: 100, TotalTokens: 1100},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 1000, OutputTokens: 100, TotalTokens: 1100}),
		},
		{
			Label:    "2026-05",
			Sessions: 6,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"xhigh"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 9000, OutputTokens: 900, TotalTokens: 9900},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 9000, OutputTokens: 900, TotalTokens: 9900}),
		},
		{
			Label:    "2026-06",
			Sessions: 4,
			Models:   []string{"codex-auto-review"},
			Efforts:  []string{"high"},
			Modes:    []string{"plan"},
			Usage:    usage.Usage{InputTokens: 3200, OutputTokens: 250, TotalTokens: 3450},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 3200, OutputTokens: 250, TotalTokens: 3450}),
		},
	}
	return reportFromRows(rows, "month", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
}

func sampleLocalTimezoneCalendarReport() usage.Report {
	location := time.FixedZone("GST", 4*60*60)
	rows := []usage.ReportRow{
		{
			Label:    "2026-05-04",
			Sessions: 1,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"high"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 3000, CachedInputTokens: 1200, OutputTokens: 200, TotalTokens: 3200},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 3000, CachedInputTokens: 1200, OutputTokens: 200, TotalTokens: 3200}),
		},
	}
	return reportFromRows(rows, "day", time.Date(2026, 5, 3, 19, 44, 0, 0, location), time.Date(2026, 6, 2, 19, 44, 0, 0, location))
}

func emptyCalendarReport() usage.Report {
	location := time.FixedZone("GST", 4*60*60)
	return usage.Report{
		Title:   "Codex Usage Report",
		Start:   time.Date(2026, 5, 3, 19, 44, 0, 0, location),
		End:     time.Date(2026, 6, 2, 19, 44, 0, 0, location),
		GroupBy: "day",
	}
}

func sampleTodayReport() usage.Report {
	location := time.FixedZone("GST", 4*60*60)
	rows := []usage.ReportRow{
		{
			Label:    "2026-06-02",
			Sessions: 4,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"xhigh"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 9000, CachedInputTokens: 4000, OutputTokens: 600, TotalTokens: 9600},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 9000, CachedInputTokens: 4000, OutputTokens: 600, TotalTokens: 9600}),
		},
	}
	return reportFromRows(rows, "day", time.Date(2026, 6, 2, 0, 0, 0, 0, location), time.Date(2026, 6, 2, 19, 44, 0, 0, location))
}

func sampleWeekReport() usage.Report {
	location := time.FixedZone("GST", 4*60*60)
	rows := []usage.ReportRow{
		{
			Label:    "2026-06-01",
			Sessions: 2,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"high"},
			Modes:    []string{"default"},
			Usage:    usage.Usage{InputTokens: 2000, OutputTokens: 200, TotalTokens: 2200},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 2000, OutputTokens: 200, TotalTokens: 2200}),
		},
		{
			Label:    "2026-06-03",
			Sessions: 3,
			Models:   []string{"gpt-5.5"},
			Efforts:  []string{"xhigh"},
			Modes:    []string{"plan"},
			Usage:    usage.Usage{InputTokens: 7000, OutputTokens: 500, TotalTokens: 7500},
			CostUSD:  usage.EstimateCostUSD(usage.Usage{InputTokens: 7000, OutputTokens: 500, TotalTokens: 7500}),
		},
	}
	return reportFromRows(rows, "day", time.Date(2026, 6, 1, 0, 0, 0, 0, location), time.Date(2026, 6, 4, 12, 0, 0, 0, location))
}

func reportFromRows(rows []usage.ReportRow, groupBy string, start, end time.Time) usage.Report {
	total := usage.Usage{}
	cost := 0.0
	sessions := 0
	for _, row := range rows {
		total = total.Add(row.Usage)
		cost += row.CostUSD
		sessions += row.Sessions
	}
	return usage.Report{
		Title:           "Codex Usage Report",
		Start:           start,
		End:             end,
		GroupBy:         groupBy,
		Rows:            rows,
		Totals:          total,
		TotalCostUSD:    cost,
		SessionsCounted: sessions,
		FilesCounted:    len(rows),
		EventsCounted:   len(rows),
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

func TestOverviewUsesKPIBoxesAndActivityHeatmap(t *testing.T) {
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
	if !strings.Contains(plain, "Usage activity") {
		t.Fatalf("overview should render usage activity heatmap, got:\n%s", plain)
	}
}

func TestOverviewReplacesTokenBarsWithCalendarHeatmap(t *testing.T) {
	model := NewLoadedModel(sampleCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := updated.renderOverview()
	plain := stripANSI(rendered)
	for _, expected := range []string{"Usage activity", "May", "Jun", "Mon", "Wed", "Fri", "Less", "More"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("overview heatmap should include %q, got:\n%s", expected, plain)
		}
	}
	for _, oldMetric := range []string{"Input", "Cached input", "Output", "Reasoning output"} {
		if strings.Contains(plain, oldMetric) {
			t.Fatalf("overview should replace token metric bars, still found %q in:\n%s", oldMetric, plain)
		}
	}
	heatmapRendered := rendered[strings.Index(rendered, "Usage activity"):]
	if index := strings.Index(heatmapRendered, "Recent usage"); index >= 0 {
		heatmapRendered = heatmapRendered[:index]
	}
	for _, sequence := range []string{
		rgbSequence(49, 46, 129),
		rgbSequence(79, 70, 229),
		rgbSequence(124, 58, 237),
		rgbSequence(196, 181, 253),
	} {
		if !strings.Contains(heatmapRendered, sequence) {
			t.Fatalf("overview heatmap should use a blue-violet activity gradient; missing %q in:\n%s", sequence, heatmapRendered)
		}
	}
	for _, sequence := range []string{
		rgbSequence(0, 166, 214),
		rgbSequence(68, 208, 123),
		rgbSequence(242, 232, 94),
		rgbSequence(239, 131, 84),
	} {
		if strings.Contains(heatmapRendered, sequence) {
			t.Fatalf("overview heatmap should not use category-like multi-color cells; found %q in:\n%s", sequence, heatmapRendered)
		}
	}
}

func TestOverviewCalendarHeatmapMapsLocalTimezoneDates(t *testing.T) {
	model := NewLoadedModel(sampleLocalTimezoneCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	grid := stripANSI(overviewHeatmapGrid(updated.renderOverview()))
	if !strings.Contains(grid, "█") {
		t.Fatalf("overview heatmap should map local timeframe dates to activity cells, got:\n%s", grid)
	}
}

func TestOverviewCalendarHeatmapKeepsYearContextForSparseRange(t *testing.T) {
	model := NewLoadedModel(sampleLocalTimezoneCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	for _, expected := range []string{"Jul", "Jan", "Apr", "May", "Jun"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("overview heatmap should keep year calendar context with %q, got:\n%s", expected, rendered)
		}
	}
}

func TestOverviewCalendarHeatmapShowsEmptyYearScaffold(t *testing.T) {
	model := NewLoadedModel(emptyCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	if strings.Contains(rendered, "No token events in this timeframe.") {
		t.Fatalf("overview should render an empty calendar scaffold instead of a no-data message:\n%s", rendered)
	}
	for _, expected := range []string{"Usage activity", "Jan", "Mon", "Wed", "Fri", "Less", "More"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("empty overview heatmap should include %q, got:\n%s", expected, rendered)
		}
	}
}

func TestOverviewHeatmapAdaptsToTodayRange(t *testing.T) {
	model := NewLoadedModel(sampleTodayReport(), Options{Theme: "dark", NoAnimation: true, Timeframe: usage.TimeframeOptions{Today: true}})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	grid := stripANSI(overviewHeatmapGrid(updated.renderOverview()))
	for _, expected := range []string{"Tue", "Jun 02", "█"} {
		if !strings.Contains(grid, expected) {
			t.Fatalf("today overview should render focused day activity with %q, got:\n%s", expected, grid)
		}
	}
	if strings.Contains(rendered, "Jan") || strings.Contains(rendered, "Jul") {
		t.Fatalf("today overview should not render the full year calendar, got:\n%s", rendered)
	}
}

func TestOverviewHeatmapAdaptsToWeekRange(t *testing.T) {
	model := NewLoadedModel(sampleWeekReport(), Options{Theme: "dark", NoAnimation: true, Timeframe: usage.TimeframeOptions{Week: true}})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	grid := stripANSI(overviewHeatmapGrid(updated.renderOverview()))
	for _, expected := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun", "█"} {
		if !strings.Contains(grid, expected) {
			t.Fatalf("week overview should render 7-day activity with %q, got:\n%s", expected, grid)
		}
	}
	if strings.Contains(rendered, "Jan") || strings.Contains(rendered, "Jul") {
		t.Fatalf("week overview should not render the full year calendar, got:\n%s", rendered)
	}
}

func TestOverviewRendersTimeframeSelectorBelowHeatmap(t *testing.T) {
	model := NewLoadedModel(sampleCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	heatmapIndex := strings.Index(rendered, "Usage activity")
	rangeIndex := strings.Index(rendered, "Last 30d")
	recentIndex := strings.Index(rendered, "Recent usage")
	if heatmapIndex < 0 || rangeIndex < 0 || recentIndex < 0 {
		t.Fatalf("overview should render heatmap, range selector, and recent usage, got:\n%s", rendered)
	}
	if !(heatmapIndex < rangeIndex && rangeIndex < recentIndex) {
		t.Fatalf("range selector should render below heatmap and above recent usage, got:\n%s", rendered)
	}
	for _, expected := range []string{"Last 30d", "Today", "Yesterday", "Current Week", "Last Week", "Current Month"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("range selector should include %q, got:\n%s", expected, rendered)
		}
	}
}

func TestOverviewArrowSelectorAppliesTimeframe(t *testing.T) {
	model := NewLoadedModel(sampleCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)

	next, _ = updated.Update(tea.KeyPressMsg(tea.Key{Text: "j", Code: 'j'}))
	updated = next.(Model)

	next, _ = updated.Update(tea.KeyPressMsg(tea.Key{Text: "l", Code: 'l'}))
	updated = next.(Model)
	if timeframeLabel(updated.timeframe) != "Last 30d" {
		t.Fatalf("timeframe should not apply until enter, got %q", timeframeLabel(updated.timeframe))
	}

	next, cmd := updated.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	updated = next.(Model)
	if cmd == nil {
		t.Fatalf("enter should trigger a report reload")
	}
	if timeframeLabel(updated.timeframe) != "Today" {
		t.Fatalf("enter should apply selected timeframe, got %q", timeframeLabel(updated.timeframe))
	}
	if updated.view != 0 {
		t.Fatalf("timeframe selector navigation should keep overview active, got view %d", updated.view)
	}
}

func TestFShortcutStillCyclesTimeframe(t *testing.T) {
	model := NewLoadedModel(sampleCalendarReport(), Options{Theme: "dark", NoAnimation: true})
	next, cmd := model.Update(tea.KeyPressMsg(tea.Key{Text: "f", Code: 'f'}))
	updated := next.(Model)
	if cmd == nil {
		t.Fatalf("f should trigger a report reload")
	}
	if timeframeLabel(updated.timeframe) != "Today" {
		t.Fatalf("f should still cycle to Today, got %q", timeframeLabel(updated.timeframe))
	}
}

func TestOverviewHeatmapUsesBlueVioletGradientInLightTheme(t *testing.T) {
	model := NewLoadedModel(sampleCalendarReport(), Options{Theme: "light", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := updated.renderOverview()
	heatmapRendered := rendered[strings.Index(rendered, "Usage activity"):]
	if index := strings.Index(heatmapRendered, "Recent usage"); index >= 0 {
		heatmapRendered = heatmapRendered[:index]
	}
	if !strings.Contains(heatmapRendered, rgbSequence(167, 139, 250)) {
		t.Fatalf("light overview heatmap should use a blue-violet peak activity color:\n%s", heatmapRendered)
	}
	for _, sequence := range []string{
		rgbSequence(234, 88, 12),
		rgbSequence(202, 138, 4),
		rgbSequence(22, 163, 74),
	} {
		if strings.Contains(heatmapRendered, sequence) {
			t.Fatalf("light overview heatmap should not use orange/green activity colors; found %q in:\n%s", sequence, heatmapRendered)
		}
	}
}

func TestOverviewHeatmapColumnsTouch(t *testing.T) {
	model := NewLoadedModel(emptyCalendarReport(), Options{Theme: "dark", NoAnimation: true, NoColor: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	row := firstHeatmapRow(rendered, "Mon")
	cells := strings.TrimSpace(strings.TrimPrefix(row, "Mon"))
	if strings.Contains(cells, " ") {
		t.Fatalf("calendar heatmap columns should touch without spaces, got row %q", row)
	}
}

func TestNoColorHeatmapUsesDitheredBlocks(t *testing.T) {
	model := NewLoadedModel(sampleCalendarReport(), Options{Theme: "dark", NoAnimation: true, NoColor: true})
	cells := []string{
		model.heatmapCell(0, 4),
		model.heatmapCell(1, 4),
		model.heatmapCell(2, 4),
		model.heatmapCell(3, 4),
		model.heatmapCell(4, 4),
	}
	if strings.Join(cells, "") != "·░▒▓█" {
		t.Fatalf("no-color heatmap should use dithered intensity blocks, got %q", strings.Join(cells, ""))
	}
}

func TestOverviewUsesGroupedHeatmapForMonthRows(t *testing.T) {
	model := NewLoadedModel(sampleMonthReport(), Options{Theme: "dark", NoAnimation: true})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	updated := next.(Model)
	rendered := stripANSI(updated.renderOverview())
	for _, expected := range []string{"Usage activity", "Apr", "May", "Jun", "Less", "More"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("month overview heatmap should include %q, got:\n%s", expected, rendered)
		}
	}
	for _, line := range strings.Split(rendered, "\n") {
		if len([]rune(line)) > 90 {
			t.Fatalf("line width %d exceeds 90: %q", len([]rune(line)), line)
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

func overviewHeatmapGrid(rendered string) string {
	start := strings.Index(rendered, "Usage activity")
	if start < 0 {
		return rendered
	}
	value := rendered[start:]
	if index := strings.Index(value, "Less"); index >= 0 {
		value = value[:index]
	}
	return value
}

func firstHeatmapRow(rendered, prefix string) string {
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}
