package models

import "time"

// PlaybackEvent records a single media playback session.
type PlaybackEvent struct {
	ID              int64
	TagID           string
	Title           string
	MediaPath       string
	StartTime       time.Time
	EndTime         time.Time
	DurationWatched int // seconds
}

// InputEventType identifies the kind of input received.
type InputEventType int

const (
	InputScan        InputEventType = iota // RFID/QR tag scanned
	InputPlayPause                         // Play/Pause button pressed
	InputStop                              // Stop button pressed
	InputRewind                            // Rewind button pressed
	InputForward                           // Forward button pressed
	InputVolumeUp                          // Volume up button pressed
	InputVolumeDown                        // Volume down button pressed
	InputNextEpisode                       // Next episode button pressed
	InputPrevEpisode                       // Previous episode button pressed
)

func (t InputEventType) String() string {
	switch t {
	case InputScan:
		return "scan"
	case InputPlayPause:
		return "play_pause"
	case InputStop:
		return "stop"
	case InputRewind:
		return "rewind"
	case InputForward:
		return "forward"
	case InputVolumeUp:
		return "volume_up"
	case InputVolumeDown:
		return "volume_down"
	case InputNextEpisode:
		return "next_episode"
	case InputPrevEpisode:
		return "prev_episode"
	default:
		return "unknown"
	}
}

// InputEvent represents an input from the scanner or a button press.
type InputEvent struct {
	Type    InputEventType
	Value   string // Tag ID for scan events, empty for buttons
	Time    time.Time
}
