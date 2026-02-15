package input

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestScanner_SingleTag(t *testing.T) {
	reader := strings.NewReader("1001\n")
	scanner := NewScanner(reader, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := scanner.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-ch:
		if event.Type != models.InputScan {
			t.Errorf("expected InputScan, got %v", event.Type)
		}
		if event.Value != "1001" {
			t.Errorf("expected tag 1001, got %s", event.Value)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for scan event")
	}
}

func TestScanner_MultipleTags(t *testing.T) {
	reader := strings.NewReader("1001\n2001\n3001\n")
	scanner := NewScanner(reader, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := scanner.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"1001", "2001", "3001"}
	for _, exp := range expected {
		select {
		case event := <-ch:
			if event.Value != exp {
				t.Errorf("expected %s, got %s", exp, event.Value)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for tag %s", exp)
		}
	}
}

func TestScanner_SkipEmptyLines(t *testing.T) {
	reader := strings.NewReader("\n\n1001\n\n")
	scanner := NewScanner(reader, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := scanner.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-ch:
		if event.Value != "1001" {
			t.Errorf("expected 1001, got %s", event.Value)
		}
	case <-ctx.Done():
		t.Fatal("timed out")
	}
}

func TestScanner_TrimWhitespace(t *testing.T) {
	reader := strings.NewReader("  1001  \n")
	scanner := NewScanner(reader, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := scanner.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-ch:
		if event.Value != "1001" {
			t.Errorf("expected trimmed '1001', got %q", event.Value)
		}
	case <-ctx.Done():
		t.Fatal("timed out")
	}
}

func TestScanner_ContextCancellation(t *testing.T) {
	// Use a pipe so the scanner blocks waiting for input.
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pw.Close()

	scanner := NewScanner(pr, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := scanner.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel immediately.
	cancel()
	pr.Close()

	// Channel should eventually close.
	select {
	case _, ok := <-ch:
		if ok {
			// Got an event, that's fine; channel will close eventually.
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after context cancellation")
	}
}

func TestScanner_EventHasTimestamp(t *testing.T) {
	reader := strings.NewReader("1001\n")
	scanner := NewScanner(reader, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	before := time.Now()
	ch, err := scanner.Listen(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-ch:
		after := time.Now()
		if event.Time.Before(before) || event.Time.After(after) {
			t.Errorf("event timestamp %v outside expected range [%v, %v]", event.Time, before, after)
		}
	case <-ctx.Done():
		t.Fatal("timed out")
	}
}
