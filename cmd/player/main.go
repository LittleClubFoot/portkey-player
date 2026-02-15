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
				if err := handleScan(ctx, log, event, mediaResolver, ruleEngine, ctrl, eventRepo, cfg); err != nil {
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
	eventRepo *logger.SQLiteRepository,
	cfg *models.Config,
) error {
	tagID := event.Value
	log.Info("processing scan", "tag_id", tagID)

	// Resolve tag to media item.
	item, err := mediaResolver.Resolve(tagID)
	if err != nil {
		return fmt.Errorf("resolving tag %s: %w", tagID, err)
	}

	// Check parental control rules.
	allowed, reason, err := ruleEngine.CanPlay(tagID, time.Now())
	if err != nil {
		return fmt.Errorf("checking rules for tag %s: %w", tagID, err)
	}
	if !allowed {
		log.Warn("playback denied", "tag_id", tagID, "title", item.Title, "reason", reason)
		return nil
	}

	// Record the play for daily limit tracking.
	if err := ruleEngine.RecordPlay(tagID, time.Now()); err != nil {
		log.Error("failed to record play", "error", err)
	}

	// Start playback.
	startTime := time.Now()
	if err := ctrl.StartPlayback(item); err != nil {
		return fmt.Errorf("starting playback: %w", err)
	}

	// Start button handler for playback controls.
	buttonCh := startButtonHandler(ctx, log, cfg)

	// Monitor playback in a goroutine, handling button presses concurrently.
	doneCh := make(chan struct{})
	go func() {
		ctrl.WaitForEnd()
		close(doneCh)
	}()

	// Process button events until playback ends.
loop:
	for {
		select {
		case <-ctx.Done():
			ctrl.HandleInput(models.InputEvent{Type: models.InputStop})
			break loop
		case <-doneCh:
			break loop
		case btnEvent, ok := <-buttonCh:
			if !ok {
				continue
			}
			if err := ctrl.HandleInput(btnEvent); err != nil {
				log.Error("button handling failed", "event", btnEvent.Type.String(), "error", err)
			}
			if btnEvent.Type == models.InputStop {
				break loop
			}
		}
	}

	// Log playback event.
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

	ctrl.Reset()
	log.Info("playback ended", "title", item.Title, "duration_seconds", playbackEvent.DurationWatched)
	return nil
}

func startButtonHandler(ctx context.Context, log *slog.Logger, cfg *models.Config) <-chan models.InputEvent {
	// Build pin map from config.
	pinMap := map[models.InputEventType]int{
		models.InputPlayPause:  cfg.Hardware.Buttons.PlayPause,
		models.InputStop:       cfg.Hardware.Buttons.Stop,
		models.InputRewind:     cfg.Hardware.Buttons.Rewind,
		models.InputForward:    cfg.Hardware.Buttons.Forward,
		models.InputVolumeUp:   cfg.Hardware.Buttons.VolumeUp,
		models.InputVolumeDown: cfg.Hardware.Buttons.VolumeDown,
	}

	// Use a no-op GPIO provider when not on a Pi (for development).
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
