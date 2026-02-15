package player

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// mockPlayer is a test double for Player.
type mockPlayer struct {
	playing    bool
	paused     bool
	stopped    bool
	position   float64
	volume     int
	seekDelta  int
	playPath   string
	waitCalled bool
	err        error
}

func (m *mockPlayer) Play(path string) error {
	if m.err != nil {
		return m.err
	}
	m.playPath = path
	m.playing = true
	m.paused = false
	m.stopped = false
	return nil
}
func (m *mockPlayer) Pause() error {
	m.paused = true
	m.playing = false
	return m.err
}
func (m *mockPlayer) Resume() error {
	m.paused = false
	m.playing = true
	return m.err
}
func (m *mockPlayer) Stop() error {
	m.stopped = true
	m.playing = false
	m.paused = false
	return m.err
}
func (m *mockPlayer) Seek(seconds int) error {
	m.seekDelta = seconds
	return m.err
}
func (m *mockPlayer) SetVolume(level int) error {
	m.volume = level
	return m.err
}
func (m *mockPlayer) IsPlaying() bool { return m.playing }
func (m *mockPlayer) GetPosition() (float64, error) {
	return m.position, m.err
}
func (m *mockPlayer) WaitForEnd() error {
	m.waitCalled = true
	return m.err
}

func testControllerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestController_StartPlayback(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{
		TagID: "1001", Path: "/media/frozen.mp4", Title: "Frozen",
	}

	if err := ctrl.StartPlayback(item); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctrl.State() != models.PlayerPlaying {
		t.Errorf("expected playing state, got %v", ctrl.State())
	}
	if ctrl.Current().Title != "Frozen" {
		t.Errorf("expected current title Frozen, got %s", ctrl.Current().Title)
	}
	if mp.playPath != "/media/frozen.mp4" {
		t.Errorf("expected play path /media/frozen.mp4, got %s", mp.playPath)
	}
}

func TestController_StartPlayback_Error(t *testing.T) {
	mp := &mockPlayer{err: fmt.Errorf("play failed")}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	if err := ctrl.StartPlayback(item); err == nil {
		t.Fatal("expected error")
	}
}

func TestController_HandlePlayPause(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlayback(item)

	// Pause.
	ctrl.HandleInput(models.InputEvent{Type: models.InputPlayPause})
	if ctrl.State() != models.PlayerPaused {
		t.Errorf("expected paused, got %v", ctrl.State())
	}
	if !mp.paused {
		t.Error("expected mock player to be paused")
	}

	// Resume.
	ctrl.HandleInput(models.InputEvent{Type: models.InputPlayPause})
	if ctrl.State() != models.PlayerPlaying {
		t.Errorf("expected playing, got %v", ctrl.State())
	}
}

func TestController_HandleStop(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlayback(item)

	ctrl.HandleInput(models.InputEvent{Type: models.InputStop})
	if ctrl.State() != models.PlayerStopped {
		t.Errorf("expected stopped, got %v", ctrl.State())
	}
	if !mp.stopped {
		t.Error("expected mock player to be stopped")
	}
}

func TestController_HandleSeek(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlayback(item)

	ctrl.HandleInput(models.InputEvent{Type: models.InputRewind})
	if mp.seekDelta != -10 {
		t.Errorf("expected seek -10, got %d", mp.seekDelta)
	}

	ctrl.HandleInput(models.InputEvent{Type: models.InputForward})
	if mp.seekDelta != 10 {
		t.Errorf("expected seek 10, got %d", mp.seekDelta)
	}
}

func TestController_HandleVolume(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlayback(item)

	ctrl.HandleInput(models.InputEvent{Type: models.InputVolumeUp})
	if mp.volume != 105 {
		t.Errorf("expected volume 105, got %d", mp.volume)
	}

	ctrl.HandleInput(models.InputEvent{Type: models.InputVolumeDown})
	if mp.volume != 100 {
		t.Errorf("expected volume 100, got %d", mp.volume)
	}
}

func TestController_VolumeClamp(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlayback(item)

	// Volume down many times to hit floor.
	for i := 0; i < 30; i++ {
		ctrl.HandleInput(models.InputEvent{Type: models.InputVolumeDown})
	}
	if mp.volume < 0 {
		t.Errorf("volume should not go below 0, got %d", mp.volume)
	}
}

func TestController_Reset(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlayback(item)
	ctrl.Reset()

	if ctrl.State() != models.PlayerIdle {
		t.Errorf("expected idle after reset, got %v", ctrl.State())
	}
	if ctrl.Current() != nil {
		t.Error("expected nil current after reset")
	}
	if ctrl.PlaybackDuration() != 0 {
		t.Error("expected 0 duration after reset")
	}
}

func TestController_PlaybackDuration_Idle(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	if ctrl.PlaybackDuration() != 0 {
		t.Error("expected 0 duration when idle")
	}
}
