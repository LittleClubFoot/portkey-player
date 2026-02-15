package resolver

import (
	"fmt"
	"os"
	"strings"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// ConfigResolver resolves tag IDs to media items using the loaded config.
// It implements both MediaResolver and SeriesResolver.
type ConfigResolver struct {
	media map[string]models.MediaEntry
}

// NewConfigResolver creates a resolver from the config's media map.
func NewConfigResolver(media map[string]models.MediaEntry) *ConfigResolver {
	return &ConfigResolver{media: media}
}

// Resolve returns the media item for a tag. For series, it returns the first episode.
func (r *ConfigResolver) Resolve(tagID string) (*models.MediaItem, error) {
	entry, ok := r.media[tagID]
	if !ok {
		return nil, fmt.Errorf("unknown tag ID: %s", tagID)
	}

	if entry.Type == "series" {
		return r.FirstEpisode(tagID)
	}

	return &models.MediaItem{
		TagID:    tagID,
		Path:     entry.Path,
		Title:    entry.Title,
		Duration: entry.Duration,
		Type:     entry.Type,
	}, nil
}

// IsSeries returns true if the tag maps to a multi-episode series.
func (r *ConfigResolver) IsSeries(tagID string) bool {
	entry, ok := r.media[tagID]
	if !ok {
		return false
	}
	return entry.Type == "series" && len(entry.Episodes) > 0
}

// ResolveEpisode returns the media item for a specific season/episode in a series.
func (r *ConfigResolver) ResolveEpisode(tagID string, season, episode int) (*models.MediaItem, error) {
	entry, ok := r.media[tagID]
	if !ok {
		return nil, fmt.Errorf("unknown tag ID: %s", tagID)
	}
	if entry.Type != "series" {
		return nil, fmt.Errorf("tag %s is not a series", tagID)
	}

	for _, ep := range entry.Episodes {
		if ep.Season == season && ep.Episode == episode {
			return r.episodeToItem(tagID, entry.Title, ep), nil
		}
	}

	return nil, fmt.Errorf("episode S%02dE%02d not found for tag %s", season, episode, tagID)
}

// NextEpisode returns the episode after the given one in the ordered list.
// Returns nil (not an error) if already at the last episode.
func (r *ConfigResolver) NextEpisode(tagID string, season, episode int) (*models.MediaItem, error) {
	entry, ok := r.media[tagID]
	if !ok {
		return nil, fmt.Errorf("unknown tag ID: %s", tagID)
	}

	idx := r.findEpisodeIndex(entry.Episodes, season, episode)
	if idx < 0 {
		return nil, fmt.Errorf("current episode S%02dE%02d not found for tag %s", season, episode, tagID)
	}
	if idx >= len(entry.Episodes)-1 {
		return nil, nil // at the end
	}

	next := entry.Episodes[idx+1]
	return r.episodeToItem(tagID, entry.Title, next), nil
}

// PrevEpisode returns the episode before the given one in the ordered list.
// Returns nil (not an error) if already at the first episode.
func (r *ConfigResolver) PrevEpisode(tagID string, season, episode int) (*models.MediaItem, error) {
	entry, ok := r.media[tagID]
	if !ok {
		return nil, fmt.Errorf("unknown tag ID: %s", tagID)
	}

	idx := r.findEpisodeIndex(entry.Episodes, season, episode)
	if idx < 0 {
		return nil, fmt.Errorf("current episode S%02dE%02d not found for tag %s", season, episode, tagID)
	}
	if idx == 0 {
		return nil, nil // at the start
	}

	prev := entry.Episodes[idx-1]
	return r.episodeToItem(tagID, entry.Title, prev), nil
}

// FirstEpisode returns the first episode in the series.
func (r *ConfigResolver) FirstEpisode(tagID string) (*models.MediaItem, error) {
	entry, ok := r.media[tagID]
	if !ok {
		return nil, fmt.Errorf("unknown tag ID: %s", tagID)
	}
	if len(entry.Episodes) == 0 {
		return nil, fmt.Errorf("series %s has no episodes", tagID)
	}

	return r.episodeToItem(tagID, entry.Title, entry.Episodes[0]), nil
}

// EpisodeCount returns the total number of episodes in a series.
func (r *ConfigResolver) EpisodeCount(tagID string) int {
	entry, ok := r.media[tagID]
	if !ok {
		return 0
	}
	return len(entry.Episodes)
}

func (r *ConfigResolver) Exists(mediaPath string) (bool, error) {
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

func (r *ConfigResolver) findEpisodeIndex(episodes []models.EpisodeEntry, season, episode int) int {
	for i, ep := range episodes {
		if ep.Season == season && ep.Episode == episode {
			return i
		}
	}
	return -1
}

func (r *ConfigResolver) episodeToItem(tagID, seriesTitle string, ep models.EpisodeEntry) *models.MediaItem {
	title := ep.Title
	if title == "" {
		title = fmt.Sprintf("%s S%02dE%02d", seriesTitle, ep.Season, ep.Episode)
	}
	return &models.MediaItem{
		TagID:    tagID,
		Path:     ep.Path,
		Title:    title,
		Duration: ep.Duration,
		Type:     "episode",
		Season:   ep.Season,
		Episode:  ep.Episode,
	}
}
