package rules

import (
	"testing"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// mockCounter is a test double for PlayCounter.
type mockCounter struct {
	count int
	plays []struct {
		tagID     string
		timestamp time.Time
	}
}

func (m *mockCounter) CountPlays(date time.Time) (int, error) {
	return m.count, nil
}

func (m *mockCounter) RecordPlay(tagID string, timestamp time.Time) error {
	m.plays = append(m.plays, struct {
		tagID     string
		timestamp time.Time
	}{tagID, timestamp})
	return nil
}

func timeAt(hour, min int) time.Time {
	return time.Date(2026, 1, 15, hour, min, 0, 0, time.UTC)
}

func TestRuleEngine_CanPlay_NoRules(t *testing.T) {
	counter := &mockCounter{}
	engine := NewRuleEngine(models.RulesConfig{}, counter)

	ok, reason, err := engine.CanPlay("1001", timeAt(12, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected play allowed, got denied: %s", reason)
	}
}

func TestRuleEngine_QuietHours_DuringQuiet(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		QuietHours: &models.QuietHours{Start: "20:00", End: "07:00"},
	}
	engine := NewRuleEngine(rules, counter)

	// 21:00 is during quiet hours.
	ok, reason, err := engine.CanPlay("1001", timeAt(21, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected play denied during quiet hours")
	}
	if reason == "" {
		t.Error("expected a reason for denial")
	}
}

func TestRuleEngine_QuietHours_BeforeQuiet(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		QuietHours: &models.QuietHours{Start: "20:00", End: "07:00"},
	}
	engine := NewRuleEngine(rules, counter)

	// 15:00 is outside quiet hours.
	ok, _, err := engine.CanPlay("1001", timeAt(15, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected play allowed outside quiet hours")
	}
}

func TestRuleEngine_QuietHours_EarlyMorning(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		QuietHours: &models.QuietHours{Start: "20:00", End: "07:00"},
	}
	engine := NewRuleEngine(rules, counter)

	// 05:00 is during quiet hours (spans midnight).
	ok, _, err := engine.CanPlay("1001", timeAt(5, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected play denied at 5am during quiet hours")
	}
}

func TestRuleEngine_QuietHours_AtEnd(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		QuietHours: &models.QuietHours{Start: "20:00", End: "07:00"},
	}
	engine := NewRuleEngine(rules, counter)

	// 07:00 is exactly at end - should be allowed.
	ok, _, err := engine.CanPlay("1001", timeAt(7, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected play allowed at exactly end of quiet hours")
	}
}

func TestRuleEngine_QuietHours_SameDay(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		QuietHours: &models.QuietHours{Start: "13:00", End: "15:00"},
	}
	engine := NewRuleEngine(rules, counter)

	// 14:00 is during quiet hours (same-day range).
	ok, _, err := engine.CanPlay("1001", timeAt(14, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected play denied during same-day quiet hours")
	}

	// 12:00 is before quiet hours.
	ok, _, err = engine.CanPlay("1001", timeAt(12, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected play allowed before same-day quiet hours")
	}
}

func TestRuleEngine_BedtimeRestriction_BeforeThreshold(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		BedtimeOnlyAfter: "19:00",
		BedtimeTags:      []string{"story1", "lullaby1"},
	}
	engine := NewRuleEngine(rules, counter)

	// Before 19:00, all content allowed.
	ok, _, err := engine.CanPlay("movie1", timeAt(18, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected non-bedtime content allowed before threshold")
	}
}

func TestRuleEngine_BedtimeRestriction_AfterThreshold_NonBedtime(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		BedtimeOnlyAfter: "19:00",
		BedtimeTags:      []string{"story1", "lullaby1"},
	}
	engine := NewRuleEngine(rules, counter)

	// After 19:00, non-bedtime content denied.
	ok, _, err := engine.CanPlay("movie1", timeAt(19, 30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected non-bedtime content denied after threshold")
	}
}

func TestRuleEngine_BedtimeRestriction_AfterThreshold_BedtimeTag(t *testing.T) {
	counter := &mockCounter{}
	rules := models.RulesConfig{
		BedtimeOnlyAfter: "19:00",
		BedtimeTags:      []string{"story1", "lullaby1"},
	}
	engine := NewRuleEngine(rules, counter)

	// After 19:00, bedtime content allowed.
	ok, _, err := engine.CanPlay("story1", timeAt(19, 30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected bedtime content allowed after threshold")
	}
}

func TestRuleEngine_DailyLimit_UnderLimit(t *testing.T) {
	counter := &mockCounter{count: 3}
	rules := models.RulesConfig{
		MaxPlaysPerDay: 5,
	}
	engine := NewRuleEngine(rules, counter)

	ok, _, err := engine.CanPlay("1001", timeAt(12, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected play allowed under daily limit")
	}
}

func TestRuleEngine_DailyLimit_AtLimit(t *testing.T) {
	counter := &mockCounter{count: 5}
	rules := models.RulesConfig{
		MaxPlaysPerDay: 5,
	}
	engine := NewRuleEngine(rules, counter)

	ok, reason, err := engine.CanPlay("1001", timeAt(12, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected play denied at daily limit")
	}
	if reason == "" {
		t.Error("expected a reason for denial")
	}
}

func TestRuleEngine_DailyLimit_Unlimited(t *testing.T) {
	counter := &mockCounter{count: 100}
	rules := models.RulesConfig{
		MaxPlaysPerDay: 0, // 0 means unlimited
	}
	engine := NewRuleEngine(rules, counter)

	ok, _, err := engine.CanPlay("1001", timeAt(12, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected play allowed with unlimited daily limit")
	}
}

func TestRuleEngine_RecordPlay(t *testing.T) {
	counter := &mockCounter{}
	engine := NewRuleEngine(models.RulesConfig{}, counter)

	now := time.Now()
	if err := engine.RecordPlay("1001", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counter.plays) != 1 {
		t.Fatalf("expected 1 recorded play, got %d", len(counter.plays))
	}
	if counter.plays[0].tagID != "1001" {
		t.Errorf("expected tag ID 1001, got %s", counter.plays[0].tagID)
	}
}

func TestRuleEngine_CombinedRules(t *testing.T) {
	counter := &mockCounter{count: 4}
	rules := models.RulesConfig{
		MaxPlaysPerDay:   5,
		QuietHours:       &models.QuietHours{Start: "20:00", End: "07:00"},
		BedtimeOnlyAfter: "19:00",
		BedtimeTags:      []string{"story1"},
	}
	engine := NewRuleEngine(rules, counter)

	// 15:00, under limit, not quiet hours, before bedtime - allowed.
	ok, _, err := engine.CanPlay("movie1", timeAt(15, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected play allowed at 15:00")
	}

	// 19:30, non-bedtime content - denied by bedtime rule.
	ok, _, err = engine.CanPlay("movie1", timeAt(19, 30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected play denied by bedtime rule at 19:30")
	}

	// 19:30, bedtime content - allowed.
	ok, _, err = engine.CanPlay("story1", timeAt(19, 30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected bedtime content allowed at 19:30")
	}

	// 21:00 - denied by quiet hours (before bedtime check).
	ok, _, err = engine.CanPlay("story1", timeAt(21, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected play denied by quiet hours at 21:00")
	}
}
