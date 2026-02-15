package models

// MediaItem represents a resolved media item ready for playback.
type MediaItem struct {
	TagID    string
	Path     string
	Title    string
	Duration int    // minutes
	Type     string // "movie", "episode", "music", "audiobook"
}

// PlayerState represents the current state of the media player.
type PlayerState int

const (
	PlayerIdle    PlayerState = iota
	PlayerPlaying
	PlayerPaused
	PlayerStopped
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
	default:
		return "unknown"
	}
}
