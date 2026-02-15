package resolver

import (
	"fmt"
	"os"
	"strings"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// ConfigResolver resolves tag IDs to media items using the loaded config.
type ConfigResolver struct {
	media map[string]models.MediaEntry
}

// NewConfigResolver creates a resolver from the config's media map.
func NewConfigResolver(media map[string]models.MediaEntry) *ConfigResolver {
	return &ConfigResolver{media: media}
}

func (r *ConfigResolver) Resolve(tagID string) (*models.MediaItem, error) {
	entry, ok := r.media[tagID]
	if !ok {
		return nil, fmt.Errorf("unknown tag ID: %s", tagID)
	}

	return &models.MediaItem{
		TagID:    tagID,
		Path:     entry.Path,
		Title:    entry.Title,
		Duration: entry.Duration,
		Type:     entry.Type,
	}, nil
}

func (r *ConfigResolver) Exists(mediaPath string) (bool, error) {
	// Network paths (smb://, nfs://) are assumed reachable if the mount exists.
	if strings.HasPrefix(mediaPath, "smb://") || strings.HasPrefix(mediaPath, "nfs://") {
		return true, nil
	}

	_, err := os.Stat(mediaPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking media path %s: %w", mediaPath, err)
}
