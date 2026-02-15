package logger

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// SQLiteRepository implements Repository using a SQLite database.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository opens (or creates) a SQLite database at the given path.
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS playback_events (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		tag_id          TEXT NOT NULL,
		title           TEXT NOT NULL,
		media_path      TEXT NOT NULL,
		start_time      DATETIME NOT NULL,
		end_time        DATETIME,
		duration_watched INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_events_start ON playback_events(start_time);
	CREATE INDEX IF NOT EXISTS idx_events_tag   ON playback_events(tag_id);

	CREATE TABLE IF NOT EXISTS series_progress (
		tag_id           TEXT PRIMARY KEY,
		current_season   INTEGER NOT NULL DEFAULT 1,
		current_episode  INTEGER NOT NULL DEFAULT 1,
		position_seconds INTEGER NOT NULL DEFAULT 0,
		completed        INTEGER NOT NULL DEFAULT 0
	);
	`
	_, err := db.Exec(schema)
	return err
}

func (r *SQLiteRepository) LogEvent(event *models.PlaybackEvent) error {
	result, err := r.db.Exec(
		`INSERT INTO playback_events (tag_id, title, media_path, start_time, end_time, duration_watched)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.TagID, event.Title, event.MediaPath,
		event.StartTime, event.EndTime, event.DurationWatched,
	)
	if err != nil {
		return fmt.Errorf("inserting playback event: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		event.ID = id
	}
	return nil
}

func (r *SQLiteRepository) GetEvents(from, to time.Time) ([]models.PlaybackEvent, error) {
	rows, err := r.db.Query(
		`SELECT id, tag_id, title, media_path, start_time, end_time, duration_watched
		 FROM playback_events
		 WHERE start_time >= ? AND start_time <= ?
		 ORDER BY start_time DESC`,
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()

	var events []models.PlaybackEvent
	for rows.Next() {
		var e models.PlaybackEvent
		if err := rows.Scan(&e.ID, &e.TagID, &e.Title, &e.MediaPath,
			&e.StartTime, &e.EndTime, &e.DurationWatched); err != nil {
			return nil, fmt.Errorf("scanning event row: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *SQLiteRepository) CountPlays(date time.Time) (int, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM playback_events WHERE start_time >= ? AND start_time < ?`,
		dayStart, dayEnd,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting plays: %w", err)
	}
	return count, nil
}

func (r *SQLiteRepository) RecordPlay(tagID string, timestamp time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO playback_events (tag_id, title, media_path, start_time, duration_watched)
		 VALUES (?, '', '', ?, 0)`,
		tagID, timestamp,
	)
	if err != nil {
		return fmt.Errorf("recording play: %w", err)
	}
	return nil
}

// GetProgress returns the viewing progress for a series tag, or nil if none exists.
func (r *SQLiteRepository) GetProgress(tagID string) (*models.SeriesProgress, error) {
	var p models.SeriesProgress
	var completed int
	err := r.db.QueryRow(
		`SELECT tag_id, current_season, current_episode, position_seconds, completed
		 FROM series_progress WHERE tag_id = ?`, tagID,
	).Scan(&p.TagID, &p.CurrentSeason, &p.CurrentEpisode, &p.PositionSeconds, &completed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting series progress for %s: %w", tagID, err)
	}
	p.Completed = completed != 0
	return &p, nil
}

// SaveProgress upserts the viewing progress for a series tag.
func (r *SQLiteRepository) SaveProgress(progress *models.SeriesProgress) error {
	completed := 0
	if progress.Completed {
		completed = 1
	}
	_, err := r.db.Exec(
		`INSERT INTO series_progress (tag_id, current_season, current_episode, position_seconds, completed)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(tag_id) DO UPDATE SET
		   current_season = excluded.current_season,
		   current_episode = excluded.current_episode,
		   position_seconds = excluded.position_seconds,
		   completed = excluded.completed`,
		progress.TagID, progress.CurrentSeason, progress.CurrentEpisode,
		progress.PositionSeconds, completed,
	)
	if err != nil {
		return fmt.Errorf("saving series progress for %s: %w", progress.TagID, err)
	}
	return nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}
