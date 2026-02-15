package models

// Config is the top-level application configuration.
type Config struct {
	Version  string          `json:"version"`
	Media    map[string]MediaEntry `json:"media"`
	Rules    RulesConfig     `json:"rules"`
	Network  NetworkConfig   `json:"network"`
	Hardware HardwareConfig  `json:"hardware"`
}

// MediaEntry represents a single media item or a series mapped to a tag ID.
// When Type is "series", the Episodes field contains ordered episode data
// and Path/Duration on the entry itself are ignored.
type MediaEntry struct {
	Path      string            `json:"path,omitempty"`
	Title     string            `json:"title"`
	Duration  int               `json:"duration,omitempty"` // minutes
	Type      string            `json:"type"`               // "movie", "episode", "music", "audiobook", "series"
	AgeRating string            `json:"age_rating,omitempty"`
	Season    int               `json:"season,omitempty"`
	Episode   int               `json:"episode,omitempty"`
	Episodes  []EpisodeEntry    `json:"episodes,omitempty"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// EpisodeEntry represents a single episode within a series.
type EpisodeEntry struct {
	Season   int    `json:"season"`
	Episode  int    `json:"episode"`
	Path     string `json:"path"`
	Title    string `json:"title"`
	Duration int    `json:"duration,omitempty"` // minutes
}

// RulesConfig defines parental control rules.
type RulesConfig struct {
	MaxPlaysPerDay   int          `json:"max_plays_per_day"`
	QuietHours       *QuietHours  `json:"quiet_hours,omitempty"`
	BedtimeOnlyAfter string       `json:"bedtime_only_after,omitempty"`
	BedtimeTags      []string     `json:"bedtime_tags,omitempty"`
}

// QuietHours defines when playback is not allowed.
type QuietHours struct {
	Start string `json:"start"` // "HH:MM" format
	End   string `json:"end"`   // "HH:MM" format
}

// NetworkConfig holds network-related settings.
type NetworkConfig struct {
	NASShares []NASShare `json:"nas_shares,omitempty"`
}

// NASShare represents a network-attached storage mount.
type NASShare struct {
	Type       string `json:"type"`        // "smb" or "nfs"
	Host       string `json:"host"`
	Share      string `json:"share"`
	MountPoint string `json:"mount_point"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

// HardwareConfig defines hardware pin mappings and player settings.
type HardwareConfig struct {
	Buttons    ButtonPins `json:"buttons"`
	Player     string     `json:"player"`      // "mpv", "vlc", etc.
	PlayerArgs []string   `json:"player_args,omitempty"`
}

// ButtonPins maps button functions to GPIO pin numbers.
// NextEpisode and PrevEpisode are optional and only used for series playback.
type ButtonPins struct {
	PlayPause   int `json:"play_pause"`
	Stop        int `json:"stop"`
	Rewind      int `json:"rewind"`
	Forward     int `json:"forward"`
	VolumeUp    int `json:"volume_up"`
	VolumeDown  int `json:"volume_down"`
	NextEpisode int `json:"next_episode,omitempty"`
	PrevEpisode int `json:"prev_episode,omitempty"`
}
