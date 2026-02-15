package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LittleClubFoot/portkey-player/internal/config"
	"github.com/LittleClubFoot/portkey-player/internal/input"
	"github.com/LittleClubFoot/portkey-player/internal/logger"
	"github.com/LittleClubFoot/portkey-player/internal/player"
	"github.com/LittleClubFoot/portkey-player/internal/resolver"
	"github.com/LittleClubFoot/portkey-player/internal/rules"
	"github.com/LittleClubFoot/portkey-player/internal/storage"
	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Build-time variables set via ldflags.
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "/opt/kidsmedia/config.json", "path to configuration file")
	dbPath := flag.String("db", "/opt/kidsmedia/playback.db", "path to SQLite database")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("kidsmedia %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	log := setupLogger(*logLevel)
	log.Info("kidsmedia starting", "version", version, "build_time", buildTime)

	if err := run(log, *configPath, *dbPath); err != nil {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func setupLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}

func run(log *slog.Logger, configPath, dbPath string) error {
	// Load configuration.
	cfgRepo := config.NewFileRepository()
	cfg, err := cfgRepo.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log.Info("config loaded", "media_count", len(cfg.Media), "version", cfg.Version)

	// Mount NAS shares.
	mounter := storage.NewMounter(log)
	if len(cfg.Network.NASShares) > 0 {
		if errs := mounter.MountAll(cfg.Network.NASShares); len(errs) > 0 {
			log.Warn("some NAS shares failed to mount", "error_count", len(errs))
		}
	}
	defer mounter.UnmountAll(cfg.Network.NASShares)

	// Initialize database.
	eventRepo, err := logger.NewSQLiteRepository(dbPath)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer eventRepo.Close()

	// Initialize components.
	mediaResolver := resolver.NewConfigResolver(cfg.Media)
	ruleEngine := rules.NewRuleEngine(cfg.Rules, eventRepo)
	mpv := player.NewMPVPlayer(cfg.Hardware.PlayerArgs, log)
	ctrl := player.NewController(mpv, log)

	// Set up context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Start scanner input listener.
	scanner := input.NewScanner(os.Stdin, log)
	scanCh, err := scanner.Listen(ctx)
	if err != nil {
		return fmt.Errorf("starting scanner: %w", err)
	}
	defer scanner.Close()

	log.Info("ready - waiting for scan input")

	// Main event loop.
	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down")
			mpv.Stop()
			return nil

		case event, ok := <-scanCh:
			if !ok {
				log.Info("scanner closed")
				return nil
			}

			if event.Type == models.InputScan {
				if err := handleScan(ctx, log, event, mediaResolver, ruleEngine, ctrl, mpv, eventRepo, cfg); err != nil {
					log.Error("scan handling failed", "tag_id", event.Value, "error", err)
				}
			}
		}
	}
}

func handleScan(
	ctx context.Context,
	log *slog.Logger,
	event models.InputEvent,
	mediaResolver *resolver.ConfigResolver,
	ruleEngine *rules.RuleEngine,
	ctrl *player.Controller,
	mpv *player.MPVPlayer,
	eventRepo *logger.SQLiteRepository,
	cfg *models.Config,
) error {
	tagID := event.Value
	log.Info("processing scan", "tag_id", tagID)

	// Check parental control rules.
	allowed, reason, err := ruleEngine.CanPlay(tagID, time.Now())
	if err != nil {
		return fmt.Errorf("checking rules for tag %s: %w", tagID, err)
	}
	if !allowed {
		log.Warn("playback denied", "tag_id", tagID, "reason", reason)
		return nil
	}

	// Record the play for daily limit tracking.
	if err := ruleEngine.RecordPlay(tagID, time.Now()); err != nil {
		log.Error("failed to record play", "error", err)
	}

	if mediaResolver.IsSeries(tagID) {
		return handleSeriesScan(ctx, log, tagID, mediaResolver, ctrl, mpv, eventRepo, cfg)
	}
	return handleSingleScan(ctx, log, tagID, mediaResolver, ctrl, eventRepo, cfg)
}

// handleSingleScan handles playback for non-series media (movies, music, etc.).
func handleSingleScan(
	ctx context.Context,
	log *slog.Logger,
	tagID string,
	mediaResolver *resolver.ConfigResolver,
	ctrl *player.Controller,
	eventRepo *logger.SQLiteRepository,
	cfg *models.Config,
) error {
	item, err := mediaResolver.Resolve(tagID)
	if err != nil {
		return fmt.Errorf("resolving tag %s: %w", tagID, err)
	}

	startTime := time.Now()
	if err := ctrl.StartPlayback(item); err != nil {
		return fmt.Errorf("starting playback: %w", err)
	}

	buttonCh := startButtonHandler(ctx, log, cfg)
	playbackLoop(ctx, log, ctrl, buttonCh)

	logPlaybackEvent(log, eventRepo, tagID, item, startTime)
	ctrl.Reset()
	return nil
}

// handleSeriesScan handles the full series playback flow:
// 1. Look up progress to determine current episode
// 2. If episode in-progress, show OSD prompt and wait for resume/restart
// 3. Play the episode
// 4. On episode end, auto-advance to next episode
func handleSeriesScan(
	ctx context.Context,
	log *slog.Logger,
	tagID string,
	mediaResolver *resolver.ConfigResolver,
	ctrl *player.Controller,
	mpv *player.MPVPlayer,
	eventRepo *logger.SQLiteRepository,
	cfg *models.Config,
) error {
	// Look up saved progress for this series.
	progress, err := eventRepo.GetProgress(tagID)
	if err != nil {
		return fmt.Errorf("getting series progress: %w", err)
	}

	// Determine which episode to play.
	var item *models.MediaItem
	var resumePosition int

	if progress == nil {
		// No progress - start from the first episode.
		item, err = mediaResolver.FirstEpisode(tagID)
		if err != nil {
			return fmt.Errorf("resolving first episode: %w", err)
		}
		log.Info("starting series from beginning", "title", item.Title)
	} else if progress.Completed {
		// Last episode was completed - advance to next.
		next, err := mediaResolver.NextEpisode(tagID, progress.CurrentSeason, progress.CurrentEpisode)
		if err != nil {
			return fmt.Errorf("resolving next episode: %w", err)
		}
		if next == nil {
			// Finished the series; loop back to the first episode.
			item, err = mediaResolver.FirstEpisode(tagID)
			if err != nil {
				return fmt.Errorf("resolving first episode for restart: %w", err)
			}
			log.Info("series completed, restarting from beginning", "title", item.Title)
		} else {
			item = next
			log.Info("advancing to next episode", "title", item.Title)
		}
	} else {
		// Episode is in-progress - prompt user to resume or restart.
		item, err = mediaResolver.ResolveEpisode(tagID, progress.CurrentSeason, progress.CurrentEpisode)
		if err != nil {
			return fmt.Errorf("resolving current episode: %w", err)
		}

		log.Info("episode in-progress, prompting user",
			"title", item.Title,
			"position", progress.PositionSeconds,
		)

		choice, err := promptResumeOrRestart(ctx, log, ctrl, mpv, item, cfg)
		if err != nil {
			return err
		}

		if choice == player.EpisodeResume {
			resumePosition = progress.PositionSeconds
		}
		// EpisodeRestart: resumePosition stays 0
	}

	// Play episodes in a loop (for auto-advance).
	return playSeriesLoop(ctx, log, tagID, item, resumePosition, mediaResolver, ctrl, eventRepo, cfg)
}

// promptResumeOrRestart shows the episode paused with an OSD message and waits
// for the user to press Play/Pause (resume) or Stop (restart).
func promptResumeOrRestart(
	ctx context.Context,
	log *slog.Logger,
	ctrl *player.Controller,
	mpv *player.MPVPlayer,
	item *models.MediaItem,
	cfg *models.Config,
) (player.EpisodeAction, error) {
	// Start the episode paused so the user sees the first frame on screen.
	if err := mpv.PlayPaused(item.Path); err != nil {
		return player.EpisodeNone, fmt.Errorf("starting paused playback for prompt: %w", err)
	}

	// Show OSD prompt on screen.
	promptText := fmt.Sprintf("%s\n\nPress PLAY to resume\nPress STOP to restart", item.Title)
	if err := mpv.ShowMessage(promptText, 0); err != nil {
		log.Warn("failed to show OSD prompt", "error", err)
	}

	promptCh := ctrl.EnterPromptState(item.TagID)
	buttonCh := startButtonHandler(ctx, log, cfg)

	for {
		select {
		case <-ctx.Done():
			mpv.Stop()
			return player.EpisodeNone, ctx.Err()
		case choice := <-promptCh:
			// Stop the paused preview before real playback starts.
			mpv.Stop()
			return choice, nil
		case btnEvent, ok := <-buttonCh:
			if !ok {
				continue
			}
			action, err := ctrl.HandleInput(btnEvent)
			if err != nil {
				log.Error("prompt input error", "error", err)
			}
			if action == player.EpisodeResume || action == player.EpisodeRestart {
				// Stop the paused preview before real playback starts.
				mpv.Stop()
				return action, nil
			}
		}
	}
}

// playSeriesLoop plays episodes sequentially with auto-advance.
func playSeriesLoop(
	ctx context.Context,
	log *slog.Logger,
	tagID string,
	startItem *models.MediaItem,
	resumePosition int,
	mediaResolver *resolver.ConfigResolver,
	ctrl *player.Controller,
	eventRepo *logger.SQLiteRepository,
	cfg *models.Config,
) error {
	item := startItem
	position := resumePosition

	for {
		ctrl.SetSeriesTag(tagID)

		// Save progress as in-progress before starting.
		eventRepo.SaveProgress(&models.SeriesProgress{
			TagID:           tagID,
			CurrentSeason:   item.Season,
			CurrentEpisode:  item.Episode,
			PositionSeconds: position,
			Completed:       false,
		})

		startTime := time.Now()
		var err error
		if position > 0 {
			err = ctrl.StartPlaybackAtPosition(item, position)
		} else {
			err = ctrl.StartPlayback(item)
		}
		if err != nil {
			return fmt.Errorf("starting playback of %q: %w", item.Title, err)
		}

		// Reset position for subsequent episodes.
		position = 0

		buttonCh := startButtonHandler(ctx, log, cfg)
		playbackLoop(ctx, log, ctrl, buttonCh)

		// Save position when stopping mid-episode.
		duration := time.Since(startTime)
		logPlaybackEvent(log, eventRepo, tagID, item, startTime)

		if ctrl.State() == models.PlayerStopped {
			// User pressed stop - save position and exit.
			eventRepo.SaveProgress(&models.SeriesProgress{
				TagID: tagID, CurrentSeason: item.Season, CurrentEpisode: item.Episode,
				PositionSeconds: int(duration.Seconds()), Completed: false,
			})
			ctrl.Reset()
			return nil
		}

		// Episode finished naturally - mark as completed, try next episode.
		next, err := mediaResolver.NextEpisode(tagID, item.Season, item.Episode)
		if err != nil {
			log.Error("failed to resolve next episode", "error", err)
			eventRepo.SaveProgress(&models.SeriesProgress{
				TagID: tagID, CurrentSeason: item.Season, CurrentEpisode: item.Episode,
				Completed: true,
			})
			ctrl.Reset()
			return nil
		}
		if next == nil {
			// Series finished.
			log.Info("series completed", "tag", tagID)
			eventRepo.SaveProgress(&models.SeriesProgress{
				TagID: tagID, CurrentSeason: item.Season, CurrentEpisode: item.Episode,
				Completed: true,
			})
			ctrl.Reset()
			return nil
		}

		// Auto-advance to next episode.
		eventRepo.SaveProgress(&models.SeriesProgress{
			TagID: tagID, CurrentSeason: next.Season, CurrentEpisode: next.Episode,
			PositionSeconds: 0, Completed: false,
		})
		item = next
		log.Info("auto-advancing to next episode", "title", item.Title)
	}
}

// playbackLoop runs the button event loop during playback.
func playbackLoop(
	ctx context.Context,
	log *slog.Logger,
	ctrl *player.Controller,
	buttonCh <-chan models.InputEvent,
) {
	doneCh := make(chan struct{})
	go func() {
		ctrl.WaitForEnd()
		close(doneCh)
	}()

	for {
		select {
		case <-ctx.Done():
			ctrl.HandleInput(models.InputEvent{Type: models.InputStop})
			return
		case <-doneCh:
			return
		case btnEvent, ok := <-buttonCh:
			if !ok {
				continue
			}
			if _, err := ctrl.HandleInput(btnEvent); err != nil {
				log.Error("button handling failed", "event", btnEvent.Type.String(), "error", err)
			}
			if btnEvent.Type == models.InputStop {
				return
			}
		}
	}
}

func logPlaybackEvent(
	log *slog.Logger,
	eventRepo *logger.SQLiteRepository,
	tagID string,
	item *models.MediaItem,
	startTime time.Time,
) {
	endTime := time.Now()
	playbackEvent := &models.PlaybackEvent{
		TagID:           tagID,
		Title:           item.Title,
		MediaPath:       item.Path,
		StartTime:       startTime,
		EndTime:         endTime,
		DurationWatched: int(endTime.Sub(startTime).Seconds()),
	}
	if err := eventRepo.LogEvent(playbackEvent); err != nil {
		log.Error("failed to log playback event", "error", err)
	}
	log.Info("playback ended", "title", item.Title, "duration_seconds", playbackEvent.DurationWatched)
}

func startButtonHandler(ctx context.Context, log *slog.Logger, cfg *models.Config) <-chan models.InputEvent {
	pinMap := map[models.InputEventType]int{
		models.InputPlayPause: cfg.Hardware.Buttons.PlayPause,
		models.InputStop:      cfg.Hardware.Buttons.Stop,
		models.InputRewind:    cfg.Hardware.Buttons.Rewind,
		models.InputForward:   cfg.Hardware.Buttons.Forward,
	}

	gpio := &noopGPIO{}
	handler := input.NewButtonHandler(gpio, pinMap, log)

	ch, err := handler.Listen(ctx)
	if err != nil {
		log.Error("failed to start button handler", "error", err)
		emptyCh := make(chan models.InputEvent)
		close(emptyCh)
		return emptyCh
	}
	return ch
}

// noopGPIO is a no-op GPIO provider for development/testing.
type noopGPIO struct{}

func (n *noopGPIO) ReadPin(pin int) bool { return false }
