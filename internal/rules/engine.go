package rules

import (
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Engine defines the interface for parental control rule evaluation.
type Engine interface {
	CanPlay(tagID string, currentTime time.Time) (bool, string, error)
	RecordPlay(tagID string, timestamp time.Time) error
	GetPlayCount(date time.Time) (int, error)
}

// PlayCounter provides play count data for limit enforcement.
type PlayCounter interface {
	CountPlays(date time.Time) (int, error)
	RecordPlay(tagID string, timestamp time.Time) error
}

// RuleEngine evaluates parental control rules against the current state.
type RuleEngine struct {
	rules   models.RulesConfig
	counter PlayCounter
}

// NewRuleEngine creates a rule engine with the given config and play counter.
func NewRuleEngine(rules models.RulesConfig, counter PlayCounter) *RuleEngine {
	return &RuleEngine{rules: rules, counter: counter}
}

func (e *RuleEngine) CanPlay(tagID string, currentTime time.Time) (bool, string, error) {
	if ok, reason := e.checkQuietHours(currentTime); !ok {
		return false, reason, nil
	}

	if ok, reason := e.checkBedtimeRestriction(tagID, currentTime); !ok {
		return false, reason, nil
	}

	if ok, reason, err := e.checkDailyLimit(currentTime); !ok || err != nil {
		return ok, reason, err
	}

	return true, "", nil
}

func (e *RuleEngine) RecordPlay(tagID string, timestamp time.Time) error {
	return e.counter.RecordPlay(tagID, timestamp)
}

func (e *RuleEngine) GetPlayCount(date time.Time) (int, error) {
	return e.counter.CountPlays(date)
}
