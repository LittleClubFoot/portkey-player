package input

import (
	"context"
	"testing"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

type mockGPIO struct {
	pressed map[int]bool
}

func (m *mockGPIO) ReadPin(pin int) bool {
	return m.pressed[pin]
}

func TestButtonHandler_DetectsPress(t *testing.T) {
	gpio := &mockGPIO{pressed: map[int]bool{17: true}}
	pinMap := map[models.InputEventType]int{
		models.InputPlayPause: 17,
	}
	handler := NewButtonHandler(gpio, pinMap, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := handler.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-ch:
		if event.Type != models.InputPlayPause {
			t.Errorf("expected InputPlayPause, got %v", event.Type)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for button event")
	}
}

func TestButtonHandler_NoPressWhenNotPressed(t *testing.T) {
	gpio := &mockGPIO{pressed: map[int]bool{}}
	pinMap := map[models.InputEventType]int{
		models.InputPlayPause: 17,
		models.InputStop:      27,
	}
	handler := NewButtonHandler(gpio, pinMap, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ch, err := handler.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-ch:
		t.Errorf("expected no event, got %v", event.Type)
	case <-ctx.Done():
		// Expected: no events.
	}
}

func TestButtonHandler_Debounce(t *testing.T) {
	gpio := &mockGPIO{pressed: map[int]bool{17: true}}
	pinMap := map[models.InputEventType]int{
		models.InputPlayPause: 17,
	}
	handler := NewButtonHandler(gpio, pinMap, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := handler.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should get exactly 1 event due to debounce (200ms debounce, 500ms window).
	count := 0
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				goto done
			}
			count++
		case <-ctx.Done():
			goto done
		}
	}
done:
	// With 200ms debounce and 500ms window, we expect at most 2-3 events.
	if count == 0 {
		t.Error("expected at least 1 button event")
	}
	if count > 3 {
		t.Errorf("debounce not working: got %d events in 500ms", count)
	}
}

func TestButtonHandler_MultipleButtons(t *testing.T) {
	gpio := &mockGPIO{pressed: map[int]bool{17: true, 27: true}}
	pinMap := map[models.InputEventType]int{
		models.InputPlayPause: 17,
		models.InputStop:      27,
	}
	handler := NewButtonHandler(gpio, pinMap, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := handler.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotPlayPause := false
	gotStop := false

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				goto check
			}
			switch event.Type {
			case models.InputPlayPause:
				gotPlayPause = true
			case models.InputStop:
				gotStop = true
			}
			if gotPlayPause && gotStop {
				goto check
			}
		case <-ctx.Done():
			goto check
		}
	}
check:
	if !gotPlayPause {
		t.Error("expected PlayPause event")
	}
	if !gotStop {
		t.Error("expected Stop event")
	}
}
