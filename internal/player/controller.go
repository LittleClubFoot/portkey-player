package player

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// EpisodeAction is sent from the controller to signal episode-level events.
type EpisodeAction int

const (
	EpisodeNone       EpisodeAction = iota
	EpisodeNext                     // advance to next episode
	EpisodePrev                     // go to previous episode
	EpisodeRestart                  // restart current episode from beginning
	EpisodeResume                   // resume current episode from saved position
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

	// Series playback fields.
	seriesTag    string        // tag ID if currently playing a series
	promptResult chan EpisodeAction // used during resume/restart prompt
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

// StartPlaybackAtPosition begins playing at a specific position (for resume).
func (c *Controller) StartPlaybackAtPosition(item *models.MediaItem, positionSeconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.player.Play(item.Path); err != nil {
		return fmt.Errorf("starting playback of %q: %w", item.Title, err)
	}

	// Seek to the saved position after playback starts.
	if positionSeconds > 0 {
		if err := c.player.Seek(positionSeconds); err != nil {
			c.logger.Warn("failed to seek to saved position", "seconds", positionSeconds, "error", err)
		}
	}

	c.state = models.PlayerPlaying
	c.current = item
	c.started = time.Now()
	c.logger.Info("playback resumed", "title", item.Title, "position", positionSeconds)
	return nil
}

// EnterPromptState sets the controller to the prompting state and returns a
// channel that will receive the user's choice (resume or restart).
func (c *Controller) EnterPromptState(seriesTag string) <-chan EpisodeAction {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = models.PlayerPrompting
	c.seriesTag = seriesTag
	c.promptResult = make(chan EpisodeAction, 1)
	c.logger.Info("prompting user: resume or restart?", "tag", seriesTag)
	return c.promptResult
}

// SetSeriesTag sets the series tag for episode navigation during playback.
func (c *Controller) SetSeriesTag(tagID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seriesTag = tagID
}

// SeriesTag returns the current series tag, or empty string if not a series.
func (c *Controller) SeriesTag() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.seriesTag
}

// HandleInput processes a button input event during playback.
// Returns an EpisodeAction if episode navigation was requested.
func (c *Controller) HandleInput(event models.InputEvent) (EpisodeAction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// When prompting, only accept play/pause (resume) or stop (restart).
	if c.state == models.PlayerPrompting {
		return c.handlePromptInput(event)
	}

	switch event.Type {
	case models.InputPlayPause:
		return EpisodeNone, c.togglePause()
	case models.InputStop:
		return EpisodeNone, c.stop()
	case models.InputRewind:
		return EpisodeNone, c.player.Seek(-10)
	case models.InputForward:
		return EpisodeNone, c.player.Seek(10)
	case models.InputVolumeUp:
		return EpisodeNone, c.adjustVolume(5)
	case models.InputVolumeDown:
		return EpisodeNone, c.adjustVolume(-5)
	case models.InputNextEpisode:
		if c.seriesTag != "" {
			c.logger.Info("next episode requested")
			return EpisodeNext, c.stop()
		}
		return EpisodeNone, nil
	case models.InputPrevEpisode:
		if c.seriesTag != "" {
			c.logger.Info("previous episode requested")
			return EpisodePrev, c.stop()
		}
		return EpisodeNone, nil
	default:
		return EpisodeNone, nil
	}
}

// handlePromptInput handles input during the resume/restart prompt.
// Play/Pause = Resume, Stop = Restart from beginning.
func (c *Controller) handlePromptInput(event models.InputEvent) (EpisodeAction, error) {
	switch event.Type {
	case models.InputPlayPause:
		c.logger.Info("user chose: resume")
		c.state = models.PlayerIdle
		if c.promptResult != nil {
			c.promptResult <- EpisodeResume
			close(c.promptResult)
			c.promptResult = nil
		}
		return EpisodeResume, nil
	case models.InputStop:
		c.logger.Info("user chose: restart")
		c.state = models.PlayerIdle
		if c.promptResult != nil {
			c.promptResult <- EpisodeRestart
			close(c.promptResult)
			c.promptResult = nil
		}
		return EpisodeRestart, nil
	default:
		// Ignore other buttons during prompt.
		return EpisodeNone, nil
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
	c.seriesTag = ""
}
