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

func TestController_StartPlaybackAtPosition(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	if err := ctrl.StartPlaybackAtPosition(item, 120); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mp.seekDelta != 120 {
		t.Errorf("expected seek to 120, got %d", mp.seekDelta)
	}
	if ctrl.State() != models.PlayerPlaying {
		t.Errorf("expected playing, got %v", ctrl.State())
	}
}

func TestController_StartPlaybackAtPosition_ZeroDoesNotSeek(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Test"}
	ctrl.StartPlaybackAtPosition(item, 0)

	if mp.seekDelta != 0 {
		t.Errorf("expected no seek for position 0, got %d", mp.seekDelta)
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
	ctrl.SetSeriesTag("2001")
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
	if ctrl.SeriesTag() != "" {
		t.Error("expected empty series tag after reset")
	}
}

func TestController_PlaybackDuration_Idle(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	if ctrl.PlaybackDuration() != 0 {
		t.Error("expected 0 duration when idle")
	}
}

// --- Series-specific controller tests ---

func TestController_NextEpisode_WithSeriesTag(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "2001", Path: "/test.mp4", Title: "Test", Season: 1, Episode: 1}
	ctrl.StartPlayback(item)
	ctrl.SetSeriesTag("2001")

	action, err := ctrl.HandleInput(models.InputEvent{Type: models.InputNextEpisode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != EpisodeNext {
		t.Errorf("expected EpisodeNext, got %v", action)
	}
	if !mp.stopped {
		t.Error("expected player to be stopped for episode switch")
	}
}

func TestController_PrevEpisode_WithSeriesTag(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "2001", Path: "/test.mp4", Title: "Test", Season: 1, Episode: 2}
	ctrl.StartPlayback(item)
	ctrl.SetSeriesTag("2001")

	action, err := ctrl.HandleInput(models.InputEvent{Type: models.InputPrevEpisode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != EpisodePrev {
		t.Errorf("expected EpisodePrev, got %v", action)
	}
}

func TestController_NextEpisode_WithoutSeriesTag(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	item := &models.MediaItem{TagID: "1001", Path: "/test.mp4", Title: "Movie"}
	ctrl.StartPlayback(item)

	action, _ := ctrl.HandleInput(models.InputEvent{Type: models.InputNextEpisode})
	if action != EpisodeNone {
		t.Errorf("expected EpisodeNone for non-series, got %v", action)
	}
	if mp.stopped {
		t.Error("should not stop player for non-series next episode")
	}
}

func TestController_PromptState_Resume(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	promptCh := ctrl.EnterPromptState("2001")
	if ctrl.State() != models.PlayerPrompting {
		t.Errorf("expected prompting state, got %v", ctrl.State())
	}

	// Press play/pause to choose resume.
	action, _ := ctrl.HandleInput(models.InputEvent{Type: models.InputPlayPause})
	if action != EpisodeResume {
		t.Errorf("expected EpisodeResume, got %v", action)
	}

	// Channel should have the result.
	select {
	case result := <-promptCh:
		if result != EpisodeResume {
			t.Errorf("expected EpisodeResume on channel, got %v", result)
		}
	default:
		t.Error("expected result on prompt channel")
	}

	if ctrl.State() != models.PlayerIdle {
		t.Errorf("expected idle after prompt resolved, got %v", ctrl.State())
	}
}

func TestController_PromptState_Restart(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	promptCh := ctrl.EnterPromptState("2001")

	// Press stop to choose restart.
	action, _ := ctrl.HandleInput(models.InputEvent{Type: models.InputStop})
	if action != EpisodeRestart {
		t.Errorf("expected EpisodeRestart, got %v", action)
	}

	select {
	case result := <-promptCh:
		if result != EpisodeRestart {
			t.Errorf("expected EpisodeRestart on channel, got %v", result)
		}
	default:
		t.Error("expected result on prompt channel")
	}
}

func TestController_PromptState_IgnoresOtherButtons(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	ctrl.EnterPromptState("2001")

	// Volume and seek should be ignored during prompt.
	action, _ := ctrl.HandleInput(models.InputEvent{Type: models.InputVolumeUp})
	if action != EpisodeNone {
		t.Errorf("expected EpisodeNone for volume during prompt, got %v", action)
	}
	action, _ = ctrl.HandleInput(models.InputEvent{Type: models.InputRewind})
	if action != EpisodeNone {
		t.Errorf("expected EpisodeNone for rewind during prompt, got %v", action)
	}

	// Should still be in prompting state.
	if ctrl.State() != models.PlayerPrompting {
		t.Errorf("expected still prompting, got %v", ctrl.State())
	}

	_ = mp
}

func TestController_SeriesTag(t *testing.T) {
	mp := &mockPlayer{}
	ctrl := NewController(mp, testControllerLogger())

	if ctrl.SeriesTag() != "" {
		t.Error("expected empty series tag initially")
	}

	ctrl.SetSeriesTag("2001")
	if ctrl.SeriesTag() != "2001" {
		t.Errorf("expected series tag 2001, got %s", ctrl.SeriesTag())
	}
}
