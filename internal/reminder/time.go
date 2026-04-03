package reminder

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseOffset parses a relative offset string like "-1d", "2h", "30m".
// Returns the duration to subtract from the due date.
func ParseOffset(offset string) (time.Duration, error) {
	s := strings.TrimPrefix(offset, "-")
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid offset %q", offset)
	}

	numStr := s[:len(s)-1]
	unit := s[len(s)-1]

	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid offset number %q: %w", numStr, err)
	}

	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown offset unit %q", string(unit))
	}
}

// ComputeFireAt calculates the fire time for a relative reminder.
// dueDate is YYYY-MM-DD, offset is like "-1d". Resolves to 09:00 in the given timezone.
func ComputeFireAt(offset, dueDate, timezone string) (time.Time, error) {
	loc, err := loadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}

	due, err := time.ParseInLocation("2006-01-02", dueDate, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse due date %q: %w", dueDate, err)
	}

	// Set to 09:00 in the workspace timezone
	due = time.Date(due.Year(), due.Month(), due.Day(), 9, 0, 0, 0, loc)

	if offset == "0" || offset == "" {
		return due, nil
	}

	dur, err := ParseOffset(offset)
	if err != nil {
		return time.Time{}, err
	}

	return due.Add(-dur), nil
}

// NextRecurrence computes the next fire time for a recurring reminder.
func NextRecurrence(rec *Recurrence, after time.Time, timezone string) (time.Time, error) {
	loc, err := loadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}

	h, m, err := parseTimeOfDay(rec.Time)
	if err != nil {
		return time.Time{}, err
	}

	now := after.In(loc)

	switch rec.Frequency {
	case "daily":
		next := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, loc)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil

	case "weekly":
		targetDay, err := parseWeekday(rec.Day)
		if err != nil {
			return time.Time{}, err
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, loc)
		// Advance to the target weekday
		for next.Weekday() != targetDay || !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil

	case "monthly":
		dom := rec.DayOfMonth
		if dom < 1 || dom > 31 {
			dom = 1
		}
		next := time.Date(now.Year(), now.Month(), dom, h, m, 0, 0, loc)
		if !next.After(now) {
			next = next.AddDate(0, 1, 0)
			// Re-clamp the day in case the month doesn't have that many days
			next = time.Date(next.Year(), next.Month(), dom, h, m, 0, 0, loc)
		}
		return next, nil

	default:
		return time.Time{}, fmt.Errorf("unknown frequency %q", rec.Frequency)
	}
}

func loadLocation(tz string) (*time.Location, error) {
	if tz == "" || tz == "Local" {
		return time.Local, nil
	}
	return time.LoadLocation(tz)
}

func parseTimeOfDay(s string) (int, int, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return h, m, nil
}

func parseWeekday(s string) (time.Weekday, error) {
	switch strings.ToLower(s) {
	case "sunday":
		return time.Sunday, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("unknown weekday %q", s)
	}
}
