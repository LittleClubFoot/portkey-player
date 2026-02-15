package input

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Scanner reads tag IDs from an input stream (stdin or device file).
// USB RFID/QR scanners act as HID keyboard devices and emit newline-terminated strings.
type Scanner struct {
	reader io.Reader
	logger *slog.Logger
}

// NewScanner creates a scanner that reads from the provided reader.
func NewScanner(reader io.Reader, logger *slog.Logger) *Scanner {
	return &Scanner{reader: reader, logger: logger}
}

func (s *Scanner) Listen(ctx context.Context) (<-chan models.InputEvent, error) {
	ch := make(chan models.InputEvent, 8)

	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(s.reader)

		for {
			// Check context before each scan.
			select {
			case <-ctx.Done():
				return
			default:
			}

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					s.logger.Error("scanner read error", "error", err)
				}
				return
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			s.logger.Info("tag scanned", "tag_id", line)

			select {
			case ch <- models.InputEvent{
				Type:  models.InputScan,
				Value: line,
				Time:  time.Now(),
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (s *Scanner) Close() error {
	if closer, ok := s.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
