package tui

import (
	"fmt"
	"image/color"
	"math"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/guptarohit/asciigraph"
	"github.com/hashmil/cxusage/internal/usage"
)

type ThemeName string

const (
	ThemeAuto  ThemeName = "auto"
	ThemeDark  ThemeName = "dark"
	ThemeLight ThemeName = "light"
)

var viewNames = []string{"Overview", "Trends", "Breakdowns", "Sessions", "Help"}
var groupCycle = []string{"day", "week", "month", "session", "model", "effort", "mode", "model-effort"}
var loadingBanner = []string{
	" CCCCC  X   X  U   U   SSSS   AAA    GGGG  EEEEE",
	"C        X X   U   U  S      A   A  G      E",
	"C         X    U   U   SSS   AAAAA  G  GG  EEEE",
	"C        X X   U   U      S  A   A  G   G  E",
	" CCCCC  X   X   UUU   SSSS   A   A   GGG   EEEEE",
}

const loadingBannerMinWidth = 62

type Options struct {
	CodexHome   string
	GroupBy     string
	Timeframe   usage.TimeframeOptions
	Theme       string
	NoColor     bool
	NoAnimation bool
	DetectTheme func() ThemeName
}

type Model struct {
	options      Options
	report       usage.Report
	err          error
	loading      bool
	view         int
	width        int
	height       int
	selected     int
	filtering    bool
	filter       string
	groupBy      string
	timeframe    usage.TimeframeOptions
	frameIndex   int
	theme        ThemeName
	progress     float64
	loadingFrame int
	spinner      spinner.Model
}

type reportLoadedMsg struct {
	report usage.Report
	err    error
}

type animationTickMsg struct{}

func NewModel(options Options) Model {
	if options.GroupBy == "" {
		options.GroupBy = "day"
	}
	if options.Timeframe.Last == "" && !options.Timeframe.Today && !options.Timeframe.Yesterday && !options.Timeframe.Week && !options.Timeframe.LastWeek && !options.Timeframe.Month && options.Timeframe.Since == "" && options.Timeframe.Until == "" {
		options.Timeframe.Last = "30d"
	}
	m := Model{
		options:    options,
		loading:    true,
		width:      100,
		height:     28,
		groupBy:    options.GroupBy,
		timeframe:  options.Timeframe,
		frameIndex: timeframeIndex(options.Timeframe),
		theme:      resolveTheme(options.Theme, options.DetectTheme),
		progress:   0,
		spinner:    spinner.New(spinner.WithSpinner(spinner.Line)),
	}
	m.spinner.Style = m.styles().Accent
	if options.NoAnimation {
		m.progress = 1
	}
	return m
}

func NewLoadedModel(report usage.Report, options Options) Model {
	m := NewModel(options)
	m.report = report
	m.loading = false
	m.progress = 1
	if report.GroupBy != "" {
		m.groupBy = report.GroupBy
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.loading {
		if m.options.NoAnimation {
			return m.loadReportCmd()
		}
		return tea.Batch(m.loadReportCmd(), m.spinner.Tick, m.animationTick())
	}
	if m.options.NoAnimation {
		return nil
	}
	return m.animationTick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(36, msg.Width)
		m.height = max(12, msg.Height)
		return m, nil
	case reportLoadedMsg:
		m.loading = false
		m.report = msg.report
		m.err = msg.err
		m.loadingFrame = 0
		if m.options.NoAnimation {
			m.progress = 1
			return m, nil
		}
		m.progress = 0
		return m, m.animationTick()
	case animationTickMsg:
		if m.options.NoAnimation {
			return m, nil
		}
		if m.loading {
			m.loadingFrame++
			return m, m.animationTick()
		}
		if m.progress >= 1 {
			return m, nil
		}
		m.progress = math.Min(1, m.progress+0.08)
		return m, m.animationTick()
	case spinner.TickMsg:
		if !m.loading || m.options.NoAnimation {
			return m, nil
		}
		next, cmd := m.spinner.Update(msg)
		m.spinner = next
		return m, cmd
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) View() tea.View {
	content := m.render()
	content = clampBlock(content, m.width)
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m Model) ThemeName() ThemeName {
	return m.theme
}

func (m Model) ViewName() string {
	return viewNames[m.view]
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.filtering {
		switch key {
		case "enter", "esc":
			m.filtering = false
			return m, nil
		case "backspace", "delete":
			if m.filter != "" {
				m.filter = m.filter[:len(m.filter)-1]
			}
			return m, nil
		}
		if text := msg.Key().Text; text != "" {
			m.filter += text
		}
		return m, nil
	}

	switch key {
	case "q", "ctrl+c":
		return m, func() tea.Msg { return tea.Quit() }
	case "r":
		m.loading = true
		m.loadingFrame = 0
		m.err = nil
		if m.options.NoAnimation {
			return m, m.loadReportCmd()
		}
		return m, tea.Batch(m.loadReportCmd(), m.spinner.Tick, m.animationTick())
	case "1", "2", "3", "4", "5":
		m.view = int(key[0] - '1')
		return m, nil
	case "right", "l":
		m.view = (m.view + 1) % len(viewNames)
		return m, nil
	case "left", "h":
		m.view = (m.view + len(viewNames) - 1) % len(viewNames)
		return m, nil
	case "down", "j":
		m.selected = min(m.selected+1, max(0, len(m.filteredRows())-1))
		return m, nil
	case "up", "k":
		m.selected = max(0, m.selected-1)
		return m, nil
	case "f":
		m.frameIndex = (m.frameIndex + 1) % len(timeframePresets)
		m.timeframe = timeframePresets[m.frameIndex].options
		m.loading = true
		m.loadingFrame = 0
		if m.options.NoAnimation {
			return m, m.loadReportCmd()
		}
		return m, tea.Batch(m.loadReportCmd(), m.spinner.Tick, m.animationTick())
	case "g":
		m.groupBy = nextGroup(m.groupBy)
		m.loading = true
		m.loadingFrame = 0
		if m.options.NoAnimation {
			return m, m.loadReportCmd()
		}
		return m, tea.Batch(m.loadReportCmd(), m.spinner.Tick, m.animationTick())
	case "/":
		m.view = 3
		m.filtering = true
		return m, nil
	case "a":
		m.options.NoAnimation = !m.options.NoAnimation
		if m.options.NoAnimation {
			m.progress = 1
			return m, nil
		}
		return m, m.animationTick()
	case "t":
		if m.theme == ThemeDark {
			m.theme = ThemeLight
		} else {
			m.theme = ThemeDark
		}
		return m, nil
	case "?":
		m.view = 4
		return m, nil
	}
	return m, nil
}

func (m Model) loadReportCmd() tea.Cmd {
	options := m.options
	groupBy := m.groupBy
	timeframe := m.timeframe
	return func() tea.Msg {
		start, end, label, err := usage.ResolveTimeframe(timeframe)
		if err != nil {
			return reportLoadedMsg{err: err}
		}
		report, err := usage.BuildReport(
			usage.DefaultRoots(options.CodexHome),
			start,
			end,
			groupBy,
			"Codex Usage Report - "+label,
		)
		return reportLoadedMsg{report: report, err: err}
	}
}

func (m Model) animationTick() tea.Cmd {
	return tea.Tick(35*time.Millisecond, func(time.Time) tea.Msg {
		return animationTickMsg{}
	})
}

func (m Model) render() string {
	s := m.styles()
	if m.loading {
		return m.renderLoading()
	}
	if m.err != nil {
		return strings.Join([]string{
			s.Title.Render("cxusage"),
			"",
			s.Danger.Render("Error: " + m.err.Error()),
			s.Muted.Render("Press r to retry or q to quit."),
		}, "\n")
	}

	parts := []string{
		m.renderHeader(),
		m.renderNav(),
		"",
	}
	switch m.view {
	case 0:
		parts = append(parts, m.renderOverview())
	case 1:
		parts = append(parts, m.renderTrends())
	case 2:
		parts = append(parts, m.renderBreakdowns())
	case 3:
		parts = append(parts, m.renderSessions())
	default:
		parts = append(parts, m.renderHelp())
	}
	parts = append(parts, "", m.renderFooter())
	return strings.Join(parts, "\n")
}

func (m Model) renderLoading() string {
	s := m.styles()
	spin := "..."
	if !m.options.NoAnimation {
		spin = m.spinner.View()
	}
	rows := []string{m.renderLoadingTitle()}
	rows = append(rows,
		"",
		fmt.Sprintf("%s scanning local Codex logs", s.Accent.Render(spin)),
		s.Muted.Render("No network calls. Reading ~/.codex/sessions and ~/.codex/archived_sessions."),
	)
	return strings.Join(rows, "\n")
}

func (m Model) renderLoadingTitle() string {
	s := m.styles()
	if m.width < loadingBannerMinWidth || m.height < 14 {
		return s.Title.Render("cxusage")
	}
	lines := make([]string, len(loadingBanner))
	for i, line := range loadingBanner {
		lines[i] = m.renderLoadingBannerLine(line, i)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderLoadingBannerLine(line string, row int) string {
	if m.options.NoColor {
		return line
	}
	if m.options.NoAnimation {
		return m.styles().Title.Render(line)
	}
	s := m.styles()
	runes := []rune(line)
	out := strings.Builder{}
	sweep := (m.loadingFrame * 2) % (len(runes) + 18)
	for col, char := range runes {
		if char == ' ' {
			out.WriteRune(char)
			continue
		}
		distance := abs(col + row - sweep)
		switch {
		case distance <= 1:
			out.WriteString(s.Accent.Render(string(char)))
		case distance <= 3:
			out.WriteString(s.Output.Render(string(char)))
		case distance <= 6:
			out.WriteString(s.Success.Render(string(char)))
		default:
			out.WriteString(s.Muted.Render(string(char)))
		}
	}
	return out.String()
}

func (m Model) renderHeader() string {
	s := m.styles()
	right := fmt.Sprintf("%s · %s · theme %s", m.report.GroupBy, timeframeLabel(m.timeframe), m.theme)
	title := s.Title.Render("cxusage")
	if m.width < 72 {
		return title + "\n" + s.Muted.Render(right)
	}
	gap := max(1, m.width-ansi.StringWidth(ansi.Strip(title))-ansi.StringWidth(right))
	return title + strings.Repeat(" ", gap) + s.Muted.Render(right)
}

func (m Model) renderNav() string {
	s := m.styles()
	items := make([]string, 0, len(viewNames))
	for i, name := range viewNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if i == m.view {
			items = append(items, s.TabActive.Render(label))
		} else {
			items = append(items, s.Tab.Render(label))
		}
	}
	return strings.Join(items, " ")
}

func (m Model) renderOverview() string {
	rows := []string{}
	total := scaleInt(m.report.Totals.TotalTokens, m.progress)
	cost := m.report.TotalCostUSD * m.progress
	input := scaleInt(m.report.Totals.InputTokens, m.progress)
	output := scaleInt(m.report.Totals.OutputTokens, m.progress)
	cached := scaleInt(m.report.Totals.CachedInputTokens, m.progress)
	reasoning := scaleInt(m.report.Totals.ReasoningOutputTokens, m.progress)

	cards := []string{
		m.kpiCard("Total tokens", formatInt(total), m.styles().Accent),
		m.kpiCard("Estimated cost", formatCost(cost), m.styles().Success),
		m.kpiCard("Sessions", formatInt(int64(m.report.SessionsCounted)), m.styles().Warning),
		m.kpiCard("Files / events", fmt.Sprintf("%s / %s", formatInt(int64(m.report.FilesCounted)), formatInt(int64(m.report.EventsCounted))), m.styles().Output),
	}
	if m.width >= 80 {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top, cards[0], " ", cards[1]),
			lipgloss.JoinHorizontal(lipgloss.Top, cards[2], " ", cards[3]),
		)
	} else {
		rows = append(rows, cards...)
	}
	rows = append(rows, "",
		m.metricLine("Input", input, m.report.Totals.TotalTokens),
		m.metricLine("Cached input", cached, m.report.Totals.TotalTokens),
		m.metricLine("Output", output, m.report.Totals.TotalTokens),
		m.metricLine("Reasoning output", reasoning, m.report.Totals.TotalTokens),
	)
	rows = append(rows, "", m.styles().Section.Render("Recent usage"), m.renderSessionsTable(m.filteredRows(), min(8, max(3, m.height-17))))
	return strings.Join(rows, "\n")
}

func (m Model) kpiCard(label, value string, valueStyle lipgloss.Style) string {
	width := 34
	if m.width < 80 {
		width = max(28, m.width-2)
	}
	s := m.styles()
	return s.Panel.Width(width).Render(
		s.Muted.Render(label) + "\n" +
			valueStyle.Render(truncatePlain(value, max(1, width-4))),
	)
}

func (m Model) metricLine(label string, value, total int64) string {
	barWidth := max(8, min(36, m.width-32))
	ratio := 0.0
	if total > 0 {
		ratio = float64(value) / float64(total)
	}
	fill := m.breakdownFillColor("Token types", label, 0)
	valueText := formatInt(value)
	if !m.options.NoColor {
		valueText = lipgloss.NewStyle().Foreground(fill).Render(valueText)
	}
	return fmt.Sprintf("%-16s %s  %s", label, m.progressBarWithColor(ratio, barWidth, fill), padRight(valueText, 14))
}

func (m Model) renderTrends() string {
	rows := m.filteredRows()
	if len(rows) == 0 {
		return m.styles().Muted.Render("No token events in this timeframe.")
	}
	values := make([]float64, len(rows))
	for i, row := range rows {
		values[i] = float64(row.Usage.TotalTokens)
	}
	chart := m.lineChart(values, max(30, min(m.width-4, 118)), max(6, min(m.height-13, 10)))
	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		m.statChip("Peak", formatInt(maxUsage(rows))+" tokens", m.styles().Danger),
		" ",
		m.statChip("Average", formatInt(avgUsage(rows))+" tokens/day", m.styles().Accent),
		" ",
		m.statChip("Estimated cost", formatCost(sumCost(rows)), m.styles().Success),
	)
	rangeLabel := ""
	if len(rows) > 1 {
		rangeLabel = m.styles().Muted.Render(rows[0].Label + " → " + rows[len(rows)-1].Label)
	}
	return strings.Join([]string{
		m.styles().Section.Render("Token trend"),
		chart,
		rangeLabel,
		"",
		stats,
	}, "\n")
}

func (m Model) renderBreakdowns() string {
	sections := []string{
		m.breakdown("Models", aggregateRows(m.report.Rows, func(row usage.ReportRow) []string { return row.Models })),
		m.breakdown("Efforts", aggregateRows(m.report.Rows, func(row usage.ReportRow) []string { return row.Efforts })),
		m.breakdown("Modes", aggregateRows(m.report.Rows, func(row usage.ReportRow) []string { return row.Modes })),
		m.breakdown("Token types", map[string]int64{
			"input":            m.report.Totals.InputTokens,
			"cached input":     m.report.Totals.CachedInputTokens,
			"output":           m.report.Totals.OutputTokens,
			"reasoning output": m.report.Totals.ReasoningOutputTokens,
		}),
	}
	return strings.Join(sections, "\n\n")
}

func (m Model) breakdown(title string, values map[string]int64) string {
	s := m.styles()
	if len(values) == 0 {
		return s.Section.Render(title) + "\n" + s.Muted.Render("No data.")
	}
	keys := make([]string, 0, len(values))
	maxValue := int64(0)
	for key, value := range values {
		keys = append(keys, key)
		maxValue = max(maxValue, value)
	}
	sort.Slice(keys, func(i, j int) bool {
		return values[keys[i]] > values[keys[j]]
	})
	barWidth := max(8, min(34, m.width-34))
	lines := []string{s.Section.Render(title)}
	for i, key := range keys[:min(len(keys), 8)] {
		ratio := 0.0
		if maxValue > 0 {
			ratio = float64(values[key]) / float64(maxValue)
		}
		lines = append(lines, fmt.Sprintf("%-18s %s %12s", truncatePlain(key, 18), m.breakdownBar(title, key, i, ratio, barWidth), formatInt(values[key])))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderSessions() string {
	lines := []string{}
	if m.filtering {
		lines = append(lines, m.styles().Warning.Render("filter: "+m.filter+"_"))
	} else if m.filter != "" {
		lines = append(lines, m.styles().Muted.Render("filter: "+m.filter+"  (/ to edit)"))
	} else {
		lines = append(lines, m.styles().Muted.Render("/ to filter sessions, models, efforts, or modes"))
	}
	lines = append(lines, m.renderSessionsTable(m.filteredRows(), max(3, m.height-8)))
	return strings.Join(lines, "\n")
}

func (m Model) renderSessionsTable(rows []usage.ReportRow, limit int) string {
	if len(rows) == 0 {
		return m.styles().Muted.Render("No rows matched.")
	}
	if m.width < 97 {
		return m.renderCompactRows(rows, limit)
	}
	return m.renderWideRows(rows, limit)
}

func (m Model) renderWideRows(rows []usage.ReportRow, limit int) string {
	s := m.styles()
	markerW, dateW, sessW, tokenW, costW := 2, 12, 8, 13, 9
	gapWidth := 2
	columnCount := 8
	fixedWidth := markerW + dateW + sessW + tokenW + costW + gapWidth*(columnCount-1)
	metaW := max(10, (m.width-fixedWidth)/3)
	widths := []int{markerW, dateW, sessW, metaW, metaW, metaW, tokenW, costW}
	aligns := []string{"left", "left", "right", "left", "left", "left", "right", "right"}
	headers := [][]string{{""}, {"Date"}, {"Sessions"}, {"Models"}, {"Efforts"}, {"Modes"}, {"Tokens"}, {"Cost"}}
	headerStyles := []lipgloss.Style{s.Muted, s.Muted, s.Muted, s.Muted, s.Muted, s.Muted, s.Muted, s.Muted}
	lines := []string{ledgerMultiRow(headers, widths, headerStyles, aligns)}
	for i, row := range rows[:min(len(rows), limit)] {
		marker := ""
		if i == m.selected {
			marker = "›"
		}
		cells := [][]string{
			{marker},
			{row.Label},
			{formatInt(int64(row.Sessions))},
			wrapList(row.Models, metaW),
			wrapList(row.Efforts, metaW),
			wrapList(row.Modes, metaW),
			{formatInt(row.Usage.TotalTokens)},
			{formatCost(row.CostUSD)},
		}
		styles := []lipgloss.Style{s.Warning, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()}
		lines = append(lines, ledgerMultiRow(cells, widths, styles, aligns))
	}
	lines = append(lines, s.Border.Render(strings.Repeat("─", ledgerWidth(widths))))
	totalStyles := []lipgloss.Style{s.Total, s.Total, s.Total, s.Total, s.Total, s.Total, s.Total, s.Total}
	lines = append(lines, ledgerMultiRow([][]string{
		{""},
		{"Total"},
		{formatInt(int64(m.report.SessionsCounted))},
		joinListLines(m.report.Rows, func(r usage.ReportRow) []string { return r.Models }, metaW),
		joinListLines(m.report.Rows, func(r usage.ReportRow) []string { return r.Efforts }, metaW),
		joinListLines(m.report.Rows, func(r usage.ReportRow) []string { return r.Modes }, metaW),
		{formatInt(m.report.Totals.TotalTokens)},
		{formatCost(m.report.TotalCostUSD)},
	}, widths, totalStyles, aligns))
	return strings.Join(lines, "\n")
}

func (m Model) renderCompactRows(rows []usage.ReportRow, limit int) string {
	s := m.styles()
	lines := []string{}
	for i, row := range rows[:min(len(rows), limit)] {
		marker := " "
		if i == m.selected {
			marker = s.Warning.Render("›")
		}
		top := fmt.Sprintf("%s %s  sessions %s  %s  %s",
			marker,
			row.Label,
			formatInt(int64(row.Sessions)),
			formatInt(row.Usage.TotalTokens),
			formatCost(row.CostUSD),
		)
		lines = append(lines, s.Header.Render(top))
		lines = append(lines, compactMetaLine("models", row.Models, max(10, m.width-10), s))
		lines = append(lines, compactMetaLine("efforts", row.Efforts, max(10, m.width-10), s))
		lines = append(lines, compactMetaLine("modes", row.Modes, max(10, m.width-10), s))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderHelp() string {
	s := m.styles()
	lines := []string{
		s.Section.Render("Keys"),
		"q / ctrl+c   quit",
		"r            refresh logs",
		"1-5          switch views",
		"left/right   switch views",
		"j/k          move selection",
		"f            cycle timeframe",
		"g            cycle grouping",
		"/            filter rows",
		"a            toggle animation",
		"t            toggle dark/light theme",
		"?            help",
		"",
		s.Section.Render("Legend"),
		"Cost is an estimate using the GPT-5.5 rates configured in the parser.",
		"Service tier is omitted because local historical logs do not reliably expose it.",
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderFooter() string {
	s := m.styles()
	status := fmt.Sprintf("rows %d · group %s · timeframe %s · t theme · a animation · ? help", len(m.filteredRows()), m.groupBy, timeframeLabel(m.timeframe))
	if m.options.NoAnimation {
		status += " · animation off"
	}
	if m.options.NoColor {
		status += " · color off"
	}
	return s.Muted.Render(status)
}

func (m Model) filteredRows() []usage.ReportRow {
	if strings.TrimSpace(m.filter) == "" {
		return m.report.Rows
	}
	query := strings.ToLower(strings.TrimSpace(m.filter))
	out := []usage.ReportRow{}
	for _, row := range m.report.Rows {
		haystack := strings.ToLower(strings.Join(append(append(append([]string{row.Label}, row.Models...), row.Efforts...), row.Modes...), " "))
		if strings.Contains(haystack, query) {
			out = append(out, row)
		}
	}
	return out
}

type styleSet struct {
	Title     lipgloss.Style
	Header    lipgloss.Style
	Section   lipgloss.Style
	Muted     lipgloss.Style
	Accent    lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Danger    lipgloss.Style
	Output    lipgloss.Style
	Chart     lipgloss.Style
	Bar       lipgloss.Style
	Border    lipgloss.Style
	Panel     lipgloss.Style
	Tab       lipgloss.Style
	TabActive lipgloss.Style
	Total     lipgloss.Style
}

func (m Model) styles() styleSet {
	if m.options.NoColor {
		base := lipgloss.NewStyle()
		return styleSet{
			Title: base.Bold(true), Header: base.Bold(true), Section: base.Bold(true),
			Muted: base, Accent: base, Success: base, Warning: base, Danger: base,
			Output: base, Chart: base, Bar: base, Border: base, Tab: base,
			Panel:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			TabActive: base.Bold(true), Total: base.Bold(true),
		}
	}
	if m.theme == ThemeLight {
		return styleSet{
			Title:     lipgloss.NewStyle().Foreground(lipgloss.Color("#0F766E")).Bold(true),
			Header:    lipgloss.NewStyle().Foreground(lipgloss.Color("#1F2937")),
			Section:   lipgloss.NewStyle().Foreground(lipgloss.Color("#4F46E5")).Bold(true),
			Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
			Accent:    lipgloss.NewStyle().Foreground(lipgloss.Color("#EA580C")).Bold(true),
			Success:   lipgloss.NewStyle().Foreground(lipgloss.Color("#15803D")).Bold(true),
			Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color("#A16207")).Bold(true),
			Danger:    lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true),
			Output:    lipgloss.NewStyle().Foreground(lipgloss.Color("#0891B2")).Bold(true),
			Chart:     lipgloss.NewStyle().Foreground(lipgloss.Color("#2563EB")),
			Bar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#15803D")),
			Border:    lipgloss.NewStyle().Foreground(lipgloss.Color("#CBD5E1")),
			Panel:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#CBD5E1")).Padding(0, 1),
			Tab:       lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")),
			TabActive: lipgloss.NewStyle().Foreground(lipgloss.Color("#111827")).Background(lipgloss.Color("#D9E0FF")),
			Total:     lipgloss.NewStyle().Foreground(lipgloss.Color("#4F46E5")).Bold(true),
		}
	}
	return styleSet{
		Title:     lipgloss.NewStyle().Foreground(lipgloss.Color("#8BD5CA")).Bold(true),
		Header:    lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")),
		Section:   lipgloss.NewStyle().Foreground(lipgloss.Color("#C6D0F5")).Bold(true),
		Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color("#7F849C")),
		Accent:    lipgloss.NewStyle().Foreground(lipgloss.Color("#EF8354")).Bold(true),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color("#80C990")).Bold(true),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color("#EBCB8B")).Bold(true),
		Danger:    lipgloss.NewStyle().Foreground(lipgloss.Color("#E78284")).Bold(true),
		Output:    lipgloss.NewStyle().Foreground(lipgloss.Color("#8BD5CA")).Bold(true),
		Chart:     lipgloss.NewStyle().Foreground(lipgloss.Color("#C6D0F5")),
		Bar:       lipgloss.NewStyle().Foreground(lipgloss.Color("#80C990")),
		Border:    lipgloss.NewStyle().Foreground(lipgloss.Color("#45475A")),
		Panel:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#45475A")).Padding(0, 1),
		Tab:       lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")),
		TabActive: lipgloss.NewStyle().Foreground(lipgloss.Color("#181825")).Background(lipgloss.Color("#C6D0F5")),
		Total:     lipgloss.NewStyle().Foreground(lipgloss.Color("#EBCB8B")).Bold(true),
	}
}

func DetectSystemTheme() ThemeName {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
		if err == nil && strings.Contains(strings.ToLower(string(out)), "dark") {
			return ThemeDark
		}
		if err != nil {
			return ThemeLight
		}
	}
	return ThemeDark
}

func resolveTheme(value string, detect func() ThemeName) ThemeName {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dark":
		return ThemeDark
	case "light":
		return ThemeLight
	case "auto", "":
		if detect == nil {
			detect = DetectSystemTheme
		}
		detected := detect()
		if detected == ThemeLight {
			return ThemeLight
		}
		return ThemeDark
	default:
		return ThemeDark
	}
}

type timeframePreset struct {
	label   string
	options usage.TimeframeOptions
}

var timeframePresets = []timeframePreset{
	{label: "Last 30d", options: usage.TimeframeOptions{Last: "30d"}},
	{label: "Today", options: usage.TimeframeOptions{Today: true}},
	{label: "Yesterday", options: usage.TimeframeOptions{Yesterday: true}},
	{label: "Current Week", options: usage.TimeframeOptions{Week: true}},
	{label: "Last Week", options: usage.TimeframeOptions{LastWeek: true}},
	{label: "Current Month", options: usage.TimeframeOptions{Month: true}},
}

func timeframeIndex(options usage.TimeframeOptions) int {
	for i, preset := range timeframePresets {
		if preset.options.Last == options.Last &&
			preset.options.Today == options.Today &&
			preset.options.Yesterday == options.Yesterday &&
			preset.options.Week == options.Week &&
			preset.options.LastWeek == options.LastWeek &&
			preset.options.Month == options.Month {
			return i
		}
	}
	return 0
}

func timeframeLabel(options usage.TimeframeOptions) string {
	if options.Since != "" || options.Until != "" {
		return "custom"
	}
	for _, preset := range timeframePresets {
		if timeframeIndex(options) == timeframeIndex(preset.options) {
			return preset.label
		}
	}
	if options.Last != "" {
		return "Last " + options.Last
	}
	return "Last 30d"
}

func nextGroup(current string) string {
	for i, group := range groupCycle {
		if current == group {
			return groupCycle[(i+1)%len(groupCycle)]
		}
	}
	return groupCycle[0]
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (m Model) progressBar(ratio float64, width int) string {
	fill := lipgloss.Color("#80C990")
	if m.theme == ThemeLight {
		fill = lipgloss.Color("#15803D")
	}
	return m.progressBarWithColor(ratio, width, fill)
}

func (m Model) progressBarWithColor(ratio float64, width int, fill color.Color) string {
	ratio = math.Max(0, math.Min(1, ratio))
	if m.options.NoColor {
		return bar(ratio, width)
	}
	empty := lipgloss.Color("#313244")
	if m.theme == ThemeLight {
		empty = lipgloss.Color("#E5E7EB")
	}
	p := progress.New(
		progress.WithWidth(width),
		progress.WithoutPercentage(),
		progress.WithFillCharacters('█', '░'),
		progress.WithColors(fill, empty),
	)
	return p.ViewAs(ratio)
}

func (m Model) breakdownBar(title, key string, index int, ratio float64, width int) string {
	return m.progressBarWithColor(ratio, width, m.breakdownFillColor(title, key, index))
}

func (m Model) breakdownFillColor(title, key string, index int) color.Color {
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))
	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	if m.theme == ThemeLight {
		switch normalizedTitle {
		case "efforts":
			return mapColor(normalizedKey, map[string]string{
				"xhigh":   "#DC2626",
				"high":    "#A16207",
				"medium":  "#EA580C",
				"low":     "#15803D",
				"unknown": "#6B7280",
			}, "#0891B2")
		case "modes":
			return mapColor(normalizedKey, map[string]string{
				"default": "#0891B2",
				"plan":    "#4F46E5",
				"unknown": "#6B7280",
			}, "#0891B2")
		case "token types":
			return mapColor(normalizedKey, map[string]string{
				"input":            "#15803D",
				"cached input":     "#0891B2",
				"output":           "#EA580C",
				"reasoning output": "#4F46E5",
			}, "#0891B2")
		default:
			return paletteColor(index, []string{"#0891B2", "#4F46E5", "#EA580C", "#A16207"})
		}
	}
	switch normalizedTitle {
	case "efforts":
		return mapColor(normalizedKey, map[string]string{
			"xhigh":   "#E78284",
			"high":    "#EBCB8B",
			"medium":  "#EF8354",
			"low":     "#80C990",
			"unknown": "#7F849C",
		}, "#8BD5CA")
	case "modes":
		return mapColor(normalizedKey, map[string]string{
			"default": "#8BD5CA",
			"plan":    "#C6D0F5",
			"unknown": "#7F849C",
		}, "#8BD5CA")
	case "token types":
		return mapColor(normalizedKey, map[string]string{
			"input":            "#80C990",
			"cached input":     "#8BD5CA",
			"output":           "#EF8354",
			"reasoning output": "#C6D0F5",
		}, "#8BD5CA")
	default:
		return paletteColor(index, []string{"#8BD5CA", "#C6D0F5", "#EF8354", "#EBCB8B"})
	}
}

func mapColor(key string, values map[string]string, fallback string) color.Color {
	if color, ok := values[key]; ok {
		return lipgloss.Color(color)
	}
	return lipgloss.Color(fallback)
}

func paletteColor(index int, colors []string) color.Color {
	if len(colors) == 0 {
		return lipgloss.Color("")
	}
	return lipgloss.Color(colors[index%len(colors)])
}

func (m Model) lineChart(values []float64, width, height int) string {
	if len(values) == 0 {
		return ""
	}
	options := []asciigraph.Option{
		asciigraph.Width(max(12, width-14)),
		asciigraph.Height(max(4, height)),
		asciigraph.LowerBound(0),
		asciigraph.Precision(0),
		asciigraph.YAxisValueFormatter(func(value float64) string {
			return formatCompactInt(int64(math.Round(value)))
		}),
	}
	if !m.options.NoColor {
		options = append(options, asciigraph.SeriesColors(asciigraph.Cyan), asciigraph.AxisColor(asciigraph.DarkGray), asciigraph.LabelColor(asciigraph.LightSlateGray))
	}
	return m.withChartBottomAxis(asciigraph.Plot(values, options...))
}

func (m Model) statChip(label, value string, valueStyle lipgloss.Style) string {
	s := m.styles()
	return lipgloss.NewStyle().Width(22).Render(s.Muted.Render(label) + "\n" + valueStyle.Render(truncatePlain(value, 22)))
}

func compactMetaLine(label string, values []string, width int, s styleSet) string {
	prefix := "  " + s.Muted.Render(padRight(label, 7)) + " "
	lines := wrapList(values, max(8, width-10))
	return prefix + strings.Join(lines, "\n          ")
}

func ledgerWidth(widths []int) int {
	if len(widths) == 0 {
		return 0
	}
	total := 0
	for _, width := range widths {
		total += width
	}
	return total + (len(widths)-1)*2
}

func ledgerMultiRow(cells [][]string, widths []int, styles []lipgloss.Style, aligns []string) string {
	height := 1
	for _, cell := range cells {
		height = max(height, len(cell))
	}
	lines := make([]string, height)
	for row := 0; row < height; row++ {
		parts := make([]string, len(widths))
		for col, width := range widths {
			value := ""
			if col < len(cells) && row < len(cells[col]) {
				value = cells[col][row]
			}
			if col < len(styles) {
				value = styles[col].Render(value)
			}
			value = truncatePlain(value, width)
			if col < len(aligns) && aligns[col] == "right" {
				parts[col] = padLeft(value, width)
			} else {
				parts[col] = padRight(value, width)
			}
		}
		lines[row] = strings.Join(parts, "  ")
	}
	return strings.Join(lines, "\n")
}

func (m Model) withChartBottomAxis(chart string) string {
	lines := strings.Split(chart, "\n")
	axisColumn := -1
	maxWidth := 0
	for _, line := range lines {
		plain := stripANSI(line)
		width := ansi.StringWidth(plain)
		maxWidth = max(maxWidth, width)
		for _, marker := range []string{"┤", "┼"} {
			if index := strings.Index(plain, marker); index >= 0 {
				axisColumn = ansi.StringWidth(plain[:index])
				break
			}
		}
	}
	if axisColumn < 0 || maxWidth <= axisColumn {
		return chart
	}
	baseline := strings.Repeat(" ", axisColumn) + "└" + strings.Repeat("─", maxWidth-axisColumn-1)
	return chart + "\n" + m.styles().Muted.Render(baseline)
}

func wrapList(values []string, width int) []string {
	if len(values) == 0 {
		return []string{"-"}
	}
	return wrapCommaList(values, width)
}

func wrapCommaList(values []string, width int) []string {
	lines := []string{}
	current := ""
	for _, value := range values {
		chunk := strings.TrimSpace(value)
		if chunk == "" {
			continue
		}
		next := chunk
		if current != "" {
			next = current + ", " + chunk
		}
		if ansi.StringWidth(next) <= width {
			current = next
			continue
		}
		if current != "" {
			lines = append(lines, current)
		}
		if ansi.StringWidth(chunk) > width {
			lines = append(lines, truncatePlain(chunk, width))
			current = ""
		} else {
			current = chunk
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{"-"}
	}
	return lines
}

func joinListLines(rows []usage.ReportRow, values func(usage.ReportRow) []string, width int) []string {
	set := map[string]struct{}{}
	for _, row := range rows {
		for _, value := range values(row) {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return wrapList(out, width)
}

func aggregateRows(rows []usage.ReportRow, values func(usage.ReportRow) []string) map[string]int64 {
	out := map[string]int64{}
	for _, row := range rows {
		for _, value := range values(row) {
			out[value] += row.Usage.TotalTokens
		}
	}
	return out
}

func bar(ratio float64, width int) string {
	ratio = math.Max(0, math.Min(1, ratio))
	filled := int(math.Round(ratio * float64(width)))
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func maxUsage(rows []usage.ReportRow) int64 {
	out := int64(0)
	for _, row := range rows {
		out = max(out, row.Usage.TotalTokens)
	}
	return out
}

func avgUsage(rows []usage.ReportRow) int64 {
	if len(rows) == 0 {
		return 0
	}
	total := int64(0)
	for _, row := range rows {
		total += row.Usage.TotalTokens
	}
	return total / int64(len(rows))
}

func sumCost(rows []usage.ReportRow) float64 {
	total := 0.0
	for _, row := range rows {
		total += row.CostUSD
	}
	return total
}

func scaleInt(value int64, progress float64) int64 {
	return int64(math.Round(float64(value) * progress))
}

func formatInt(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	raw := fmt.Sprintf("%d", value)
	var out []byte
	for i, count := len(raw)-1, 0; i >= 0; i, count = i-1, count+1 {
		if count > 0 && count%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, raw[i])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}

func formatCompactInt(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	type suffix struct {
		threshold int64
		label     string
	}
	suffixes := []suffix{
		{threshold: 1_000_000_000, label: "B"},
		{threshold: 1_000_000, label: "M"},
		{threshold: 1_000, label: "K"},
	}
	sign := ""
	if negative {
		sign = "-"
	}
	for _, suffix := range suffixes {
		if value >= suffix.threshold {
			scaled := float64(value) / float64(suffix.threshold)
			if scaled >= 10 || math.Mod(scaled, 1) == 0 {
				return fmt.Sprintf("%s%.0f%s", sign, scaled, suffix.label)
			}
			return fmt.Sprintf("%s%.1f%s", sign, scaled, suffix.label)
		}
	}
	return sign + fmt.Sprintf("%d", value)
}

func formatCost(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func padRight(value string, width int) string {
	visible := ansi.StringWidth(value)
	if visible >= width {
		return value
	}
	return value + strings.Repeat(" ", width-visible)
}

func padLeft(value string, width int) string {
	visible := ansi.StringWidth(value)
	if visible >= width {
		return value
	}
	return strings.Repeat(" ", width-visible) + value
}

func truncatePlain(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= width {
		return value
	}
	tail := ""
	if width > 1 {
		tail = "~"
	}
	return ansi.Truncate(value, width, tail)
}

func clampBlock(value string, width int) string {
	if width <= 0 {
		return value
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}

func stripANSI(value string) string {
	return ansi.Strip(value)
}
