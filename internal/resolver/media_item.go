package resolver

import "github.com/LittleClubFoot/portkey-player/pkg/models"

// MediaResolver maps tag IDs to playable media items.
type MediaResolver interface {
	Resolve(tagID string) (*models.MediaItem, error)
	Exists(mediaPath string) (bool, error)
}
