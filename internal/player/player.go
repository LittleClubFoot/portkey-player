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
}
