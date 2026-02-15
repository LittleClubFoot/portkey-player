package input

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// GPIOProvider abstracts GPIO pin operations for testability.
type GPIOProvider interface {
	ReadPin(pin int) bool // true = pressed (low)
}

// ButtonHandler monitors GPIO pins for button presses.
type ButtonHandler struct {
	gpio          GPIOProvider
	pins          map[models.InputEventType]int
	logger        *slog.Logger
	debounceDelay time.Duration
	pollInterval  time.Duration
	mu            sync.Mutex
}

// NewButtonHandler creates a handler that monitors the specified GPIO pins.
func NewButtonHandler(gpio GPIOProvider, pinMap map[models.InputEventType]int, logger *slog.Logger) *ButtonHandler {
	return &ButtonHandler{
		gpio:          gpio,
		pins:          pinMap,
		logger:        logger,
		debounceDelay: 200 * time.Millisecond,
		pollInterval:  50 * time.Millisecond,
	}
}

func (b *ButtonHandler) Listen(ctx context.Context) (<-chan models.InputEvent, error) {
	ch := make(chan models.InputEvent, 16)

	go func() {
		defer close(ch)

		lastPress := make(map[models.InputEventType]time.Time)
		ticker := time.NewTicker(b.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.mu.Lock()
				for eventType, pin := range b.pins {
					if !b.gpio.ReadPin(pin) {
						continue
					}

					now := time.Now()
					if now.Sub(lastPress[eventType]) < b.debounceDelay {
						continue
					}
					lastPress[eventType] = now

					b.logger.Debug("button pressed", "event", eventType.String(), "pin", pin)

					select {
					case ch <- models.InputEvent{
						Type: eventType,
						Time: now,
					}:
					default:
						b.logger.Warn("button event dropped, channel full")
					}
				}
				b.mu.Unlock()
			}
		}
	}()

	return ch, nil
}

func (b *ButtonHandler) Close() error {
	return nil
}
