package usage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	GPT55InputUSDPer1M       = 5.00
	GPT55CachedInputUSDPer1M = 0.50
	GPT55OutputUSDPer1M      = 30.00
	UnknownMetadata          = "unknown"
)

var sessionIDPattern = regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)

type Usage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

func (u Usage) Add(other Usage) Usage {
	return Usage{
		InputTokens:           u.InputTokens + other.InputTokens,
		CachedInputTokens:     u.CachedInputTokens + other.CachedInputTokens,
		OutputTokens:          u.OutputTokens + other.OutputTokens,
		ReasoningOutputTokens: u.ReasoningOutputTokens + other.ReasoningOutputTokens,
		TotalTokens:           u.TotalTokens + other.TotalTokens,
	}
}

func (u Usage) DeltaFrom(baseline Usage) Usage {
	return Usage{
		InputTokens:           nonNegative(u.InputTokens - baseline.InputTokens),
		CachedInputTokens:     nonNegative(u.CachedInputTokens - baseline.CachedInputTokens),
		OutputTokens:          nonNegative(u.OutputTokens - baseline.OutputTokens),
		ReasoningOutputTokens: nonNegative(u.ReasoningOutputTokens - baseline.ReasoningOutputTokens),
		TotalTokens:           nonNegative(u.TotalTokens - baseline.TotalTokens),
	}
}

func (u Usage) UncachedInputTokens() int64 {
	return nonNegative(u.InputTokens - u.CachedInputTokens)
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func EstimateCostUSD(usage Usage) float64 {
	return (float64(usage.UncachedInputTokens())*GPT55InputUSDPer1M +
		float64(usage.CachedInputTokens)*GPT55CachedInputUSDPer1M +
		float64(usage.OutputTokens)*GPT55OutputUSDPer1M) / 1_000_000
}

type TurnMetadata struct {
	Model  string `json:"model"`
	Effort string `json:"effort"`
	Mode   string `json:"mode"`
}

type TokenEvent struct {
	Timestamp time.Time
	Usage     Usage
	SessionID string
	Path      string
	Metadata  TurnMetadata
}

type ReportRow struct {
	Label    string       `json:"label"`
	Usage    Usage        `json:"usage"`
	Sessions int          `json:"sessions"`
	Models   []string     `json:"models"`
	Efforts  []string     `json:"efforts"`
	Modes    []string     `json:"modes"`
	CostUSD  float64      `json:"estimated_cost_usd"`
	Metadata TurnMetadata `json:"-"`
}

type Report struct {
	Title           string      `json:"title"`
	Start           time.Time   `json:"start"`
	End             time.Time   `json:"end"`
	GroupBy         string      `json:"group_by"`
	Rows            []ReportRow `json:"rows"`
	Totals          Usage       `json:"totals"`
	TotalCostUSD    float64     `json:"estimated_cost_usd"`
	SessionsCounted int         `json:"sessions_counted"`
	FilesCounted    int         `json:"files_counted"`
	EventsCounted   int         `json:"events_counted"`
}

type TimeframeOptions struct {
	Last      string
	Today     bool
	Yesterday bool
	Week      bool
	LastWeek  bool
	Month     bool
	Since     string
	Until     string
	Now       time.Time
}

func ResolveTimeframe(options TimeframeOptions) (time.Time, time.Time, string, error) {
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	local := now.In(time.Local)
	year, month, _ := local.Date()
	midnight := func(t time.Time) time.Time {
		y, m, d := t.In(time.Local).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	}

	switch {
	case options.Today:
		start := midnight(local)
		return start, local, "Today", nil
	case options.Yesterday:
		start := midnight(local).AddDate(0, 0, -1)
		return start, midnight(local), "Yesterday", nil
	case options.Week:
		start := midnight(local).AddDate(0, 0, -int(local.Weekday()+6)%7)
		return start, local, "Current Week", nil
	case options.LastWeek:
		thisWeek := midnight(local).AddDate(0, 0, -int(local.Weekday()+6)%7)
		return thisWeek.AddDate(0, 0, -7), thisWeek, "Last Week", nil
	case options.Month:
		start := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		return start, local, "Current Month", nil
	case options.Since != "" || options.Until != "":
		start := time.Time{}
		if options.Since != "" {
			parsed, err := parseLocalDate(options.Since)
			if err != nil {
				return time.Time{}, time.Time{}, "", err
			}
			start = parsed
		}
		end := local
		if options.Until != "" {
			parsed, err := parseLocalDate(options.Until)
			if err != nil {
				return time.Time{}, time.Time{}, "", err
			}
			end = parsed.AddDate(0, 0, 1)
		}
		return start, end, "Custom Range", nil
	default:
		last := options.Last
		if last == "" {
			last = "30d"
		}
		duration, err := ParseDuration(last)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}
		return local.Add(-duration), local, "Last " + last, nil
	}
}

func ParseDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if len(value) < 2 {
		return 0, fmt.Errorf("duration must look like 24h, 7d, 30d, 2w, or 3m")
	}
	amount, err := strconv.Atoi(value[:len(value)-1])
	if err != nil {
		return 0, fmt.Errorf("duration must look like 24h, 7d, 30d, 2w, or 3m")
	}
	switch value[len(value)-1] {
	case 'h':
		return time.Duration(amount) * time.Hour, nil
	case 'd':
		return time.Duration(amount) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(amount) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(amount) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("duration must look like 24h, 7d, 30d, 2w, or 3m")
	}
}

func parseLocalDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local), nil
}

func DefaultRoots(codexHome string) []string {
	if codexHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		codexHome = filepath.Join(home, ".codex")
	}
	return []string{
		filepath.Join(codexHome, "sessions"),
		filepath.Join(codexHome, "archived_sessions"),
	}
}

func BuildReport(roots []string, start, end time.Time, groupBy, title string) (Report, error) {
	if groupBy == "" {
		groupBy = "day"
	}
	if !validGroupBy(groupBy) {
		return Report{}, fmt.Errorf("unknown group: %s", groupBy)
	}
	if !start.Before(end) {
		return Report{}, errors.New("start time must be before end time")
	}
	files, err := DiscoverRolloutFiles(roots)
	if err != nil {
		return Report{}, err
	}

	grouped := map[string]Usage{}
	groupedSessions := map[string]map[string]struct{}{}
	groupedModels := map[string]map[string]struct{}{}
	groupedEfforts := map[string]map[string]struct{}{}
	groupedModes := map[string]map[string]struct{}{}
	sessionLabels := map[string]string{}
	sessionsWithUsage := map[string]struct{}{}
	filesCounted := 0
	eventsCounted := 0

	for _, file := range files {
		previous := Usage{}
		sawEvent := false
		fileHadWindowUsage := false
		events, err := IterTokenEvents(file)
		if err != nil {
			return Report{}, err
		}
		for _, event := range events {
			sawEvent = true
			if event.Timestamp.Before(start) {
				previous = event.Usage
				continue
			}
			if !event.Timestamp.Before(end) {
				break
			}
			delta := event.Usage.DeltaFrom(previous)
			previous = event.Usage
			if delta.TotalTokens == 0 {
				continue
			}
			label := groupLabel(event, groupBy)
			if groupBy == "session" {
				label = event.SessionID
				if _, ok := sessionLabels[event.SessionID]; !ok {
					sessionLabels[event.SessionID] = groupLabel(event, groupBy)
				}
			}
			grouped[label] = grouped[label].Add(delta)
			addToSet(groupedSessions, label, event.SessionID)
			addToSet(groupedModels, label, event.Metadata.Model)
			addToSet(groupedEfforts, label, event.Metadata.Effort)
			addToSet(groupedModes, label, event.Metadata.Mode)
			sessionsWithUsage[event.SessionID] = struct{}{}
			eventsCounted++
			fileHadWindowUsage = true
		}
		if sawEvent && fileHadWindowUsage {
			filesCounted++
		}
	}

	rows := make([]ReportRow, 0, len(grouped))
	for label, usage := range grouped {
		displayLabel := label
		if groupBy == "session" {
			displayLabel = sessionLabels[label]
		}
		rows = append(rows, ReportRow{
			Label:    displayLabel,
			Usage:    usage,
			Sessions: len(groupedSessions[label]),
			Models:   sortedSet(groupedModels[label]),
			Efforts:  sortedSet(groupedEfforts[label]),
			Modes:    sortedSet(groupedModes[label]),
			CostUSD:  EstimateCostUSD(usage),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if groupBy == "session" {
			return rows[i].Usage.TotalTokens > rows[j].Usage.TotalTokens
		}
		return rows[i].Label < rows[j].Label
	})

	totals := Usage{}
	for _, row := range rows {
		totals = totals.Add(row.Usage)
	}
	if title == "" {
		title = "Codex Usage Report - " + strings.Title(groupBy)
	}
	return Report{
		Title:           title,
		Start:           start,
		End:             end,
		GroupBy:         groupBy,
		Rows:            rows,
		Totals:          totals,
		TotalCostUSD:    EstimateCostUSD(totals),
		SessionsCounted: len(sessionsWithUsage),
		FilesCounted:    filesCounted,
		EventsCounted:   eventsCounted,
	}, nil
}

func DiscoverRolloutFiles(roots []string) ([]string, error) {
	bySession := map[string][]string{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasPrefix(filepath.Base(path), "rollout-") || !strings.HasSuffix(path, ".jsonl") {
				return nil
			}
			sessionID := SessionIDForPath(path)
			bySession[sessionID] = append(bySession[sessionID], path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	files := make([]string, 0, len(bySession))
	for _, paths := range bySession {
		sort.Slice(paths, func(i, j int) bool {
			iArchived := strings.Contains(paths[i], "archived_sessions")
			jArchived := strings.Contains(paths[j], "archived_sessions")
			if iArchived != jArchived {
				return !iArchived
			}
			iInfo, iErr := os.Stat(paths[i])
			jInfo, jErr := os.Stat(paths[j])
			if iErr == nil && jErr == nil {
				return iInfo.ModTime().After(jInfo.ModTime())
			}
			return paths[i] < paths[j]
		})
		files = append(files, paths[0])
	}
	sort.Strings(files)
	return files, nil
}

func SessionIDForPath(path string) string {
	matches := sessionIDPattern.FindStringSubmatch(filepath.Base(path))
	if len(matches) > 1 {
		return matches[1]
	}
	return path
}

func IterTokenEvents(path string) ([]TokenEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sessionID := SessionIDForPath(path)
	metadata := TurnMetadata{Model: UnknownMetadata, Effort: UnknownMetadata, Mode: UnknownMetadata}
	events := []TokenEvent{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "token_count") && !strings.Contains(line, "turn_context") {
			continue
		}
		var record rawRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		if record.Type == "turn_context" {
			metadata = metadataFromPayload(record.Payload)
			continue
		}
		if record.Type != "event_msg" {
			continue
		}
		var payload tokenPayload
		if err := json.Unmarshal(record.Payload, &payload); err != nil || payload.Type != "token_count" {
			continue
		}
		if payload.Info.TotalTokenUsage.TotalTokens == 0 && payload.Info.TotalTokenUsage.InputTokens == 0 && payload.Info.TotalTokenUsage.OutputTokens == 0 {
			continue
		}
		timestamp, err := time.Parse(time.RFC3339Nano, record.Timestamp)
		if err != nil {
			continue
		}
		events = append(events, TokenEvent{
			Timestamp: timestamp,
			Usage:     payload.Info.TotalTokenUsage,
			SessionID: sessionID,
			Path:      path,
			Metadata:  metadata,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

type rawRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type tokenPayload struct {
	Type string `json:"type"`
	Info struct {
		TotalTokenUsage Usage `json:"total_token_usage"`
		LastTokenUsage  Usage `json:"last_token_usage"`
	} `json:"info"`
}

type turnPayload struct {
	Model             string `json:"model"`
	Effort            string `json:"effort"`
	CollaborationMode struct {
		Mode     string `json:"mode"`
		Settings struct {
			Model           string `json:"model"`
			ReasoningEffort string `json:"reasoning_effort"`
		} `json:"settings"`
	} `json:"collaboration_mode"`
}

func metadataFromPayload(payload json.RawMessage) TurnMetadata {
	var turn turnPayload
	if err := json.Unmarshal(payload, &turn); err != nil {
		return TurnMetadata{Model: UnknownMetadata, Effort: UnknownMetadata, Mode: UnknownMetadata}
	}
	return TurnMetadata{
		Model:  metadataValue(firstNonEmpty(turn.Model, turn.CollaborationMode.Settings.Model)),
		Effort: metadataValue(firstNonEmpty(turn.Effort, turn.CollaborationMode.Settings.ReasoningEffort)),
		Mode:   metadataValue(turn.CollaborationMode.Mode),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func metadataValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return UnknownMetadata
	}
	return value
}

func groupLabel(event TokenEvent, groupBy string) string {
	local := event.Timestamp.In(time.Local)
	switch groupBy {
	case "day":
		return local.Format("2006-01-02")
	case "week":
		year, week := local.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case "month":
		return local.Format("2006-01")
	case "session":
		return fmt.Sprintf("%s %s", local.Format("2006-01-02 15:04"), event.SessionID[:min(8, len(event.SessionID))])
	case "model":
		return event.Metadata.Model
	case "effort":
		return event.Metadata.Effort
	case "mode":
		return event.Metadata.Mode
	case "model-effort":
		return event.Metadata.Model + " / " + event.Metadata.Effort
	default:
		return event.Timestamp.Format(time.RFC3339)
	}
}

func validGroupBy(groupBy string) bool {
	switch groupBy {
	case "day", "week", "month", "session", "model", "effort", "mode", "model-effort":
		return true
	default:
		return false
	}
}

func addToSet(target map[string]map[string]struct{}, key, value string) {
	if _, ok := target[key]; !ok {
		target[key] = map[string]struct{}{}
	}
	target[key][value] = struct{}{}
}

func sortedSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i] == UnknownMetadata {
			return false
		}
		if out[j] == UnknownMetadata {
			return true
		}
		return out[i] < out[j]
	})
	return out
}
