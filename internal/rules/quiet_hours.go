package rules

import (
	"fmt"
	"time"
)

func (e *RuleEngine) checkQuietHours(currentTime time.Time) (bool, string) {
	qh := e.rules.QuietHours
	if qh == nil {
		return true, ""
	}

	start, err := parseTimeOfDay(qh.Start)
	if err != nil {
		return true, "" // invalid config, allow playback
	}
	end, err := parseTimeOfDay(qh.End)
	if err != nil {
		return true, ""
	}

	now := currentTime.Hour()*60 + currentTime.Minute()

	if start > end {
		// Quiet hours span midnight (e.g., 20:00 - 07:00).
		if now >= start || now < end {
			return false, fmt.Sprintf("quiet hours active (%s - %s)", qh.Start, qh.End)
		}
	} else {
		// Quiet hours within the same day (e.g., 13:00 - 15:00).
		if now >= start && now < end {
			return false, fmt.Sprintf("quiet hours active (%s - %s)", qh.Start, qh.End)
		}
	}

	return true, ""
}

func (e *RuleEngine) checkBedtimeRestriction(tagID string, currentTime time.Time) (bool, string) {
	if e.rules.BedtimeOnlyAfter == "" {
		return true, ""
	}

	threshold, err := parseTimeOfDay(e.rules.BedtimeOnlyAfter)
	if err != nil {
		return true, ""
	}

	now := currentTime.Hour()*60 + currentTime.Minute()
	if now < threshold {
		return true, "" // Before bedtime threshold, all content allowed.
	}

	// After bedtime threshold, only bedtime-tagged content is allowed.
	for _, bt := range e.rules.BedtimeTags {
		if bt == tagID {
			return true, ""
		}
	}

	return false, fmt.Sprintf("only bedtime content allowed after %s", e.rules.BedtimeOnlyAfter)
}

// parseTimeOfDay parses "HH:MM" into minutes since midnight.
func parseTimeOfDay(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("invalid time format %q: %w", s, err)
	}
	return t.Hour()*60 + t.Minute(), nil
}
