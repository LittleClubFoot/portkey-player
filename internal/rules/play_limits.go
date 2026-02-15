package rules

import (
	"fmt"
	"time"
)

func (e *RuleEngine) checkDailyLimit(currentTime time.Time) (bool, string, error) {
	if e.rules.MaxPlaysPerDay <= 0 {
		return true, "", nil // 0 means unlimited
	}

	count, err := e.counter.CountPlays(currentTime)
	if err != nil {
		return false, "", fmt.Errorf("checking daily play count: %w", err)
	}

	if count >= e.rules.MaxPlaysPerDay {
		return false, fmt.Sprintf("daily limit reached (%d/%d)", count, e.rules.MaxPlaysPerDay), nil
	}

	return true, "", nil
}
