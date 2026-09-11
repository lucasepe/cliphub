package shared

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// TimeRange describes an interval on the source video's timeline.
type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ParseClockTime converts hh:mm:ss or hh:mm:ss.xxx into seconds.
func ParseClockTime(value string) (float64, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time %q, expected hh:mm:ss", value)
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours < 0 {
		return 0, fmt.Errorf("invalid hours %q", parts[0])
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, fmt.Errorf("invalid minutes %q", parts[1])
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("invalid seconds %q", parts[2])
	}
	return float64(hours*3600+minutes*60) + seconds, nil
}

// ValidateTimeRange checks both timestamps and rejects empty or inverted ranges.
func ValidateTimeRange(r TimeRange) (float64, float64, error) {
	from, err := ParseClockTime(r.From)
	if err != nil {
		return 0, 0, fmt.Errorf("from: %w", err)
	}
	to, err := ParseClockTime(r.To)
	if err != nil {
		return 0, 0, fmt.Errorf("to: %w", err)
	}
	if to <= from {
		return 0, 0, errors.New("to must be greater than from")
	}
	return from, to, nil
}
