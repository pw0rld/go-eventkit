package dateparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FormatDuration returns a human-readable duration like "1h 30m", "All Day", "3 days".
func FormatDuration(start, end time.Time, allDay bool) string {
	if allDay {
		days := int(end.Sub(start).Hours()/24 + 0.5)
		if days <= 1 {
			return "All Day"
		}
		return fmt.Sprintf("%d days", days)
	}
	d := end.Sub(start)
	if d < time.Minute {
		return "0m"
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours > 0 && mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", mins)
}

// FormatTimeRange returns a human-readable time range for table display.
func FormatTimeRange(start, end time.Time, allDay bool) string {
	if allDay {
		if todayAt(start, 0, 0).Equal(todayAt(end, 0, 0)) || end.Sub(start) <= 24*time.Hour {
			return "All Day"
		}
		return fmt.Sprintf("%s - %s", start.Format("Jan 02"), end.Format("Jan 02"))
	}
	if todayAt(start, 0, 0).Equal(todayAt(end, 0, 0)) {
		return fmt.Sprintf("%s - %s", start.Format("15:04"), end.Format("15:04"))
	}
	return fmt.Sprintf("%s - %s", start.Format("Jan 02 15:04"), end.Format("Jan 02 15:04"))
}

// ParseAlertDuration parses an alert offset string like "15m", "1h", "1d" into a [time.Duration].
func ParseAlertDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty alert duration")
	}

	re := regexp.MustCompile(`^(\d+)(m|min|mins|minutes?|h|hours?|d|days?)$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid alert duration: %q (use e.g. 15m, 1h, 1d)", s)
	}

	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("alert duration overflow: %w", err)
	}
	unit := m[2]
	var scale time.Duration
	switch {
	case strings.HasPrefix(unit, "m"):
		scale = time.Minute
	case strings.HasPrefix(unit, "h"):
		scale = time.Hour
	case strings.HasPrefix(unit, "d"):
		scale = 24 * time.Hour
	}
	if scale == 0 || n > int64(1<<63-1)/int64(scale) {
		return 0, fmt.Errorf("alert duration out of range")
	}
	return time.Duration(n) * scale, nil
}
