package config

import (
	"fmt"
	"regexp"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

var timeFormatRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// Validate checks a Config for required fields and correct formats.
func Validate(cfg *models.Config) error {
	if cfg.Version == "" {
		return fmt.Errorf("version is required")
	}

	if cfg.Media == nil || len(cfg.Media) == 0 {
		return fmt.Errorf("at least one media entry is required")
	}

	for tagID, entry := range cfg.Media {
		if err := validateMediaEntry(tagID, entry); err != nil {
			return err
		}
	}

	if err := validateRules(cfg.Rules); err != nil {
		return err
	}

	for i, share := range cfg.Network.NASShares {
		if err := validateNASShare(i, share); err != nil {
			return err
		}
	}

	if cfg.Hardware.Player == "" {
		return fmt.Errorf("hardware.player is required")
	}

	return nil
}

func validateMediaEntry(tagID string, entry models.MediaEntry) error {
	if entry.Title == "" {
		return fmt.Errorf("media[%s]: title is required", tagID)
	}

	validTypes := map[string]bool{
		"movie": true, "episode": true, "music": true, "audiobook": true, "series": true,
	}
	if entry.Type != "" && !validTypes[entry.Type] {
		return fmt.Errorf("media[%s]: invalid type %q (must be movie, episode, music, audiobook, or series)", tagID, entry.Type)
	}

	if entry.Type == "series" {
		return validateSeriesEntry(tagID, entry)
	}

	// Non-series entries require a path.
	if entry.Path == "" {
		return fmt.Errorf("media[%s]: path is required", tagID)
	}
	return nil
}

func validateSeriesEntry(tagID string, entry models.MediaEntry) error {
	if len(entry.Episodes) == 0 {
		return fmt.Errorf("media[%s]: series must have at least one episode", tagID)
	}

	seen := make(map[string]bool)
	for i, ep := range entry.Episodes {
		if ep.Path == "" {
			return fmt.Errorf("media[%s].episodes[%d]: path is required", tagID, i)
		}
		if ep.Season <= 0 {
			return fmt.Errorf("media[%s].episodes[%d]: season must be > 0", tagID, i)
		}
		if ep.Episode <= 0 {
			return fmt.Errorf("media[%s].episodes[%d]: episode must be > 0", tagID, i)
		}
		key := fmt.Sprintf("S%02dE%02d", ep.Season, ep.Episode)
		if seen[key] {
			return fmt.Errorf("media[%s].episodes[%d]: duplicate episode %s", tagID, i, key)
		}
		seen[key] = true
	}
	return nil
}

func validateRules(rules models.RulesConfig) error {
	if rules.MaxPlaysPerDay < 0 {
		return fmt.Errorf("rules.max_plays_per_day must be >= 0")
	}

	if rules.QuietHours != nil {
		if !timeFormatRe.MatchString(rules.QuietHours.Start) {
			return fmt.Errorf("rules.quiet_hours.start must be HH:MM format, got %q", rules.QuietHours.Start)
		}
		if !timeFormatRe.MatchString(rules.QuietHours.End) {
			return fmt.Errorf("rules.quiet_hours.end must be HH:MM format, got %q", rules.QuietHours.End)
		}
	}

	if rules.BedtimeOnlyAfter != "" {
		if !timeFormatRe.MatchString(rules.BedtimeOnlyAfter) {
			return fmt.Errorf("rules.bedtime_only_after must be HH:MM format, got %q", rules.BedtimeOnlyAfter)
		}
	}

	return nil
}

func validateNASShare(index int, share models.NASShare) error {
	if share.Type != "smb" && share.Type != "nfs" {
		return fmt.Errorf("network.nas_shares[%d]: type must be smb or nfs, got %q", index, share.Type)
	}
	if share.Host == "" {
		return fmt.Errorf("network.nas_shares[%d]: host is required", index)
	}
	if share.Share == "" {
		return fmt.Errorf("network.nas_shares[%d]: share is required", index)
	}
	if share.MountPoint == "" {
		return fmt.Errorf("network.nas_shares[%d]: mount_point is required", index)
	}
	return nil
}
