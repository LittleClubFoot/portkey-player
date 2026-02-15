package input

import (
	"context"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Handler defines the interface for input sources (scanner, buttons).
type Handler interface {
	Listen(ctx context.Context) (<-chan models.InputEvent, error)
	Close() error
}
