package models

// MediaItem represents a resolved media item ready for playback.
type MediaItem struct {
	TagID    string
	Path     string
	Title    string
	Duration int    // minutes
	Type     string // "movie", "episode", "music", "audiobook", "series"
	Season   int    // populated for series episodes
	Episode  int    // populated for series episodes
}

// SeriesProgress tracks where a user left off in a series.
type SeriesProgress struct {
	TagID           string
	CurrentSeason   int
	CurrentEpisode  int
	PositionSeconds int // playback position within the current episode
	Completed       bool // true if the episode was finished
}

// PlayerState represents the current state of the media player.
type PlayerState int

const (
	PlayerIdle      PlayerState = iota
	PlayerPlaying
	PlayerPaused
	PlayerStopped
	PlayerPrompting // waiting for user to choose resume vs restart
)

func (s PlayerState) String() string {
	switch s {
	case PlayerIdle:
		return "idle"
	case PlayerPlaying:
		return "playing"
	case PlayerPaused:
		return "paused"
	case PlayerStopped:
		return "stopped"
	case PlayerPrompting:
		return "prompting"
	default:
		return "unknown"
	}
}
