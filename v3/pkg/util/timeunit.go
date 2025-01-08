package util

import (
	"fmt"
	"strings"
)

type TimeUnit = string

const (
	TimeUnitSeconds TimeUnit = "seconds"
	TimeUnitMinutes TimeUnit = "minutes"
	TimeUnitHours   TimeUnit = "hours"
)

func ParseTimeUnit(s string) (TimeUnit, error) {
	lower := strings.ToLower(s)
	switch lower {
	case "seconds", "second", "sec", "s":
		return TimeUnitSeconds, nil
	case "minutes", "minute", "min", "m":
		return TimeUnitMinutes, nil
	case "hours", "hour", "h":
		return TimeUnitHours, nil
	default:
		return TimeUnitSeconds, fmt.Errorf("%s is not a valid time unit", s)
	}
}
