package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

var (
	habitCardStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475A")).Padding(0, 1)
	streakStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9E2AF"))
	activityColors = []string{"#585B70", "#6A9F8B", "#89DCAA", "#C2F5B0"}
)

// entryDays preserves the recorded calendar date, including older RFC3339 responses.
func entryDays(entries []habitclient.HabitEntry) map[string]int {
	days := make(map[string]int, len(entries))
	for _, entry := range entries {
		date, err := time.Parse(time.DateOnly, entry.Date)
		if err != nil {
			date, err = time.Parse(time.RFC3339, entry.Date)
		}
		if err == nil {
			days[date.Format(time.DateOnly)] += entry.Duration
		}
	}
	return days
}

func renderHabitCards(habits []habitclient.Habit, end, today time.Time, width int) string {
	inner := max(1, min(92, width-4))
	text := lipgloss.NewStyle().Width(inner)
	cards := make([]string, 0, len(habits))
	for _, habit := range habits {
		days := entryDays(habit.Entries)
		minutes := 0
		for _, entry := range habit.Entries {
			minutes += entry.Duration
		}
		description := habit.Description
		if strings.TrimSpace(description) == "" {
			description = "No description yet."
		}
		dayLabel := "days"
		if habit.Streak == 1 {
			dayLabel = "day"
		}
		highestDayLabel := "days"
		if habit.HighestStreak == 1 {
			highestDayLabel = "day"
		}
		entryLabel := "entries"
		if len(habit.Entries) == 1 {
			entryLabel = "entry"
		}
		content := text.Render(titleStyle.Render(habit.Name)) + "\n" +
			text.Render(description) + "\n" +
			text.Render(streakStyle.Render(fmt.Sprintf("Current streak: %d %s", habit.Streak, dayLabel))) + "\n" +
			text.Render(streakStyle.Render(fmt.Sprintf("Highest streak: %d %s", habit.HighestStreak, highestDayLabel))) + "\n" +
			text.Render(fmt.Sprintf("%d %s · %d min total", len(habit.Entries), entryLabel, minutes)) + "\n\n" +
			renderActivityGrid(days, end, today, inner)
		if width >= 12 {
			content = habitCardStyle.Width(inner + habitCardStyle.GetHorizontalFrameSize()).Render(content)
		}
		cards = append(cards, content)
	}
	return strings.Join(cards, "\n\n")
}

func renderActivityGrid(days map[string]int, end, today time.Time, width int) string {
	// Each row starts with its date; narrower terminals get more rows.
	columns := max(1, min(activityDays, (width-8)/2))
	start := end.AddDate(0, 0, 1-activityDays)
	var b strings.Builder
	logged, minutes := 0, 0
	for i := range activityDays {
		day := start.AddDate(0, 0, i)
		if i%columns == 0 {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(helpStyle.Render(day.Format("01/02") + " "))
		}
		duration, exists := days[day.Format(time.DateOnly)]
		level, glyph := 0, "□"
		if exists {
			logged++
			minutes += duration
			switch {
			case duration >= 30:
				level, glyph = 3, "★"
			case duration >= 15:
				level, glyph = 2, "■"
			default:
				level, glyph = 1, "▪"
			}
		}
		if day.Format(time.DateOnly) == today.Format(time.DateOnly) {
			glyph = "[" + glyph + "]"
		} else {
			glyph += " "
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(activityColors[level])).Render(glyph))
	}
	b.WriteString("\n" + lipgloss.NewStyle().Width(max(1, width)).Render(
		fmt.Sprintf("%d/%d days · %d min", logged, activityDays, minutes)))
	return b.String()
}
