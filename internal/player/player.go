package player

// Player defines the interface for media playback control.
type Player interface {
	Play(mediaPath string) error
	Pause() error
	Resume() error
	Stop() error
	Seek(seconds int) error
	SetVolume(level int) error
	IsPlaying() bool
	GetPosition() (float64, error)
	WaitForEnd() error
	// ShowMessage displays an OSD text overlay on screen.
	// durationMs of 0 means the message persists until cleared or playback changes.
	ShowMessage(text string, durationMs int) error
}
