package player

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Controller orchestrates playback in response to input events.
type Controller struct {
	player  Player
	state   models.PlayerState
	current *models.MediaItem
	started time.Time
	volume  int
	mu      sync.Mutex
	logger  *slog.Logger
}

// NewController creates a playback controller wrapping the given player.
func NewController(player Player, logger *slog.Logger) *Controller {
	return &Controller{
		player: player,
		state:  models.PlayerIdle,
		volume: 100,
		logger: logger,
	}
}

// StartPlayback begins playing the given media item.
func (c *Controller) StartPlayback(item *models.MediaItem) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.player.Play(item.Path); err != nil {
		return fmt.Errorf("starting playback of %q: %w", item.Title, err)
	}

	c.state = models.PlayerPlaying
	c.current = item
	c.started = time.Now()
	c.logger.Info("playback started", "title", item.Title, "path", item.Path)
	return nil
}

// HandleInput processes a button input event during playback.
func (c *Controller) HandleInput(event models.InputEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch event.Type {
	case models.InputPlayPause:
		return c.togglePause()
	case models.InputStop:
		return c.stop()
	case models.InputRewind:
		return c.player.Seek(-10)
	case models.InputForward:
		return c.player.Seek(10)
	case models.InputVolumeUp:
		return c.adjustVolume(5)
	case models.InputVolumeDown:
		return c.adjustVolume(-5)
	default:
		return nil
	}
}

func (c *Controller) togglePause() error {
	switch c.state {
	case models.PlayerPlaying:
		if err := c.player.Pause(); err != nil {
			return err
		}
		c.state = models.PlayerPaused
		c.logger.Info("playback paused")
	case models.PlayerPaused:
		if err := c.player.Resume(); err != nil {
			return err
		}
		c.state = models.PlayerPlaying
		c.logger.Info("playback resumed")
	}
	return nil
}

func (c *Controller) stop() error {
	if err := c.player.Stop(); err != nil {
		return err
	}
	c.state = models.PlayerStopped
	c.logger.Info("playback stopped")
	return nil
}

func (c *Controller) adjustVolume(delta int) error {
	c.volume += delta
	if c.volume < 0 {
		c.volume = 0
	}
	if c.volume > 150 {
		c.volume = 150
	}
	return c.player.SetVolume(c.volume)
}

// WaitForEnd blocks until the current media finishes or is stopped.
func (c *Controller) WaitForEnd() error {
	return c.player.WaitForEnd()
}

// State returns the current player state.
func (c *Controller) State() models.PlayerState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Current returns the currently playing media item, or nil.
func (c *Controller) Current() *models.MediaItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// PlaybackDuration returns how long the current item has been playing.
func (c *Controller) PlaybackDuration() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == models.PlayerIdle || c.started.IsZero() {
		return 0
	}
	return time.Since(c.started)
}

// Reset sets the controller back to idle state after playback ends.
func (c *Controller) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = models.PlayerIdle
	c.current = nil
	c.started = time.Time{}
}
