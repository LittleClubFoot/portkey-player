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

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}
