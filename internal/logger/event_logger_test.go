package logger

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

func setupTestDB(t *testing.T) *SQLiteRepository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestSQLiteRepository_LogAndGetEvents(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now()
	event := &models.PlaybackEvent{
		TagID:           "1001",
		Title:           "Frozen",
		MediaPath:       "/media/frozen.mp4",
		StartTime:       now,
		EndTime:         now.Add(30 * time.Minute),
		DurationWatched: 1800,
	}

	if err := repo.LogEvent(event); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	if event.ID == 0 {
		t.Error("expected event ID to be set")
	}

	events, err := repo.GetEvents(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.TagID != "1001" {
		t.Errorf("expected tag ID 1001, got %s", e.TagID)
	}
	if e.Title != "Frozen" {
		t.Errorf("expected title Frozen, got %s", e.Title)
	}
	if e.DurationWatched != 1800 {
		t.Errorf("expected duration 1800, got %d", e.DurationWatched)
	}
}

func TestSQLiteRepository_GetEventsEmpty(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now()
	events, err := repo.GetEvents(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestSQLiteRepository_GetEventsDateFilter(t *testing.T) {
	repo := setupTestDB(t)

	yesterday := time.Now().Add(-24 * time.Hour)
	today := time.Now()

	// Log yesterday's event.
	repo.LogEvent(&models.PlaybackEvent{
		TagID: "1001", Title: "Yesterday", MediaPath: "/test.mp4",
		StartTime: yesterday, EndTime: yesterday.Add(time.Hour),
	})
	// Log today's event.
	repo.LogEvent(&models.PlaybackEvent{
		TagID: "1002", Title: "Today", MediaPath: "/test2.mp4",
		StartTime: today, EndTime: today.Add(time.Hour),
	})

	// Query only today.
	events, err := repo.GetEvents(today.Add(-time.Minute), today.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event for today, got %d", len(events))
	}
	if events[0].Title != "Today" {
		t.Errorf("expected Today, got %s", events[0].Title)
	}
}

func TestSQLiteRepository_CountPlays(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now()
	for i := 0; i < 3; i++ {
		repo.LogEvent(&models.PlaybackEvent{
			TagID: "1001", Title: "Test", MediaPath: "/test.mp4",
			StartTime: now, EndTime: now.Add(time.Minute),
		})
	}

	count, err := repo.CountPlays(now)
	if err != nil {
		t.Fatalf("failed to count plays: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestSQLiteRepository_CountPlays_DifferentDays(t *testing.T) {
	repo := setupTestDB(t)

	today := time.Now()
	yesterday := today.Add(-24 * time.Hour)

	// Log 2 plays yesterday.
	for i := 0; i < 2; i++ {
		repo.LogEvent(&models.PlaybackEvent{
			TagID: "1001", Title: "Test", MediaPath: "/test.mp4",
			StartTime: yesterday, EndTime: yesterday.Add(time.Minute),
		})
	}
	// Log 1 play today.
	repo.LogEvent(&models.PlaybackEvent{
		TagID: "1001", Title: "Test", MediaPath: "/test.mp4",
		StartTime: today, EndTime: today.Add(time.Minute),
	})

	count, err := repo.CountPlays(today)
	if err != nil {
		t.Fatalf("failed to count plays: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 play today, got %d", count)
	}
}

func TestSQLiteRepository_RecordPlay(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now()
	if err := repo.RecordPlay("1001", now); err != nil {
		t.Fatalf("failed to record play: %v", err)
	}

	count, err := repo.CountPlays(now)
	if err != nil {
		t.Fatalf("failed to count plays: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestSQLiteRepository_MultipleEvents(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now()
	tags := []string{"1001", "1002", "1003", "1004", "1005"}
	for _, tag := range tags {
		repo.LogEvent(&models.PlaybackEvent{
			TagID: tag, Title: "Test " + tag, MediaPath: "/test.mp4",
			StartTime: now, EndTime: now.Add(time.Minute),
		})
	}

	events, err := repo.GetEvents(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}
}
