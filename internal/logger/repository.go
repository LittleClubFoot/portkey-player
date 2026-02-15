package logger

import (
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Repository defines persistence for playback events.
type Repository interface {
	LogEvent(event *models.PlaybackEvent) error
	GetEvents(from, to time.Time) ([]models.PlaybackEvent, error)
	CountPlays(date time.Time) (int, error)
	RecordPlay(tagID string, timestamp time.Time) error
	Close() error
}
