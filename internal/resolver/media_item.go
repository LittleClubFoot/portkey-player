package resolver

import "github.com/LittleClubFoot/portkey-player/pkg/models"

// MediaResolver maps tag IDs to playable media items.
type MediaResolver interface {
	Resolve(tagID string) (*models.MediaItem, error)
	Exists(mediaPath string) (bool, error)
}

// SeriesResolver extends MediaResolver with series-aware episode resolution.
type SeriesResolver interface {
	MediaResolver
	// IsSeries returns true if the tag maps to a multi-episode series.
	IsSeries(tagID string) bool
	// ResolveEpisode returns the media item for a specific episode in a series.
	ResolveEpisode(tagID string, season, episode int) (*models.MediaItem, error)
	// NextEpisode returns the episode after the given one, or nil if at the end.
	NextEpisode(tagID string, season, episode int) (*models.MediaItem, error)
	// FirstEpisode returns the first episode in the series.
	FirstEpisode(tagID string) (*models.MediaItem, error)
	// EpisodeCount returns the total number of episodes in a series.
	EpisodeCount(tagID string) int
}
