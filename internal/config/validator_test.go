package config

import (
	"testing"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

func validConfig() *models.Config {
	return &models.Config{
		Version: "1.0",
		Media: map[string]models.MediaEntry{
			"1001": {
				Path:  "/media/test.mp4",
				Title: "Test Video",
				Type:  "movie",
			},
		},
		Rules: models.RulesConfig{
			MaxPlaysPerDay: 5,
		},
		Hardware: models.HardwareConfig{
			Player: "mpv",
		},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	cfg := validConfig()
	cfg.Version = ""
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
	if err.Error() != "version is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyMedia(t *testing.T) {
	cfg := validConfig()
	cfg.Media = nil
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty media")
	}
}

func TestValidate_MediaMissingPath(t *testing.T) {
	cfg := validConfig()
	cfg.Media["bad"] = models.MediaEntry{Title: "No Path", Type: "movie"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestValidate_MediaMissingTitle(t *testing.T) {
	cfg := validConfig()
	cfg.Media["bad"] = models.MediaEntry{Path: "/test.mp4"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestValidate_MediaInvalidType(t *testing.T) {
	cfg := validConfig()
	cfg.Media["bad"] = models.MediaEntry{Path: "/test.mp4", Title: "Test", Type: "invalid"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestValidate_MediaEmptyType(t *testing.T) {
	cfg := validConfig()
	cfg.Media["ok"] = models.MediaEntry{Path: "/test.mp4", Title: "Test", Type: ""}
	// Empty type is allowed (optional field).
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error for empty type, got: %v", err)
	}
}

func TestValidate_MissingPlayer(t *testing.T) {
	cfg := validConfig()
	cfg.Hardware.Player = ""
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing player")
	}
}

func TestValidate_QuietHoursValid(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.QuietHours = &models.QuietHours{Start: "20:00", End: "07:00"}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_QuietHoursInvalidStart(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.QuietHours = &models.QuietHours{Start: "25:00", End: "07:00"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid quiet hours start")
	}
}

func TestValidate_QuietHoursInvalidEnd(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.QuietHours = &models.QuietHours{Start: "20:00", End: "bad"}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid quiet hours end")
	}
}

func TestValidate_BedtimeOnlyAfterInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.BedtimeOnlyAfter = "not-a-time"
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid bedtime_only_after")
	}
}

func TestValidate_NegativeMaxPlays(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.MaxPlaysPerDay = -1
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for negative max plays")
	}
}

func TestValidate_NASShareValid(t *testing.T) {
	cfg := validConfig()
	cfg.Network.NASShares = []models.NASShare{
		{Type: "smb", Host: "nas.local", Share: "kids", MountPoint: "/mnt/nas"},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_NASShareInvalidType(t *testing.T) {
	cfg := validConfig()
	cfg.Network.NASShares = []models.NASShare{
		{Type: "ftp", Host: "nas.local", Share: "kids", MountPoint: "/mnt/nas"},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid NAS type")
	}
}

func TestValidate_NASShareMissingHost(t *testing.T) {
	cfg := validConfig()
	cfg.Network.NASShares = []models.NASShare{
		{Type: "smb", Host: "", Share: "kids", MountPoint: "/mnt/nas"},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing NAS host")
	}
}

func TestValidate_NASShareMissingShare(t *testing.T) {
	cfg := validConfig()
	cfg.Network.NASShares = []models.NASShare{
		{Type: "smb", Host: "nas.local", Share: "", MountPoint: "/mnt/nas"},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing NAS share")
	}
}

func TestValidate_NASShareMissingMountPoint(t *testing.T) {
	cfg := validConfig()
	cfg.Network.NASShares = []models.NASShare{
		{Type: "nfs", Host: "nas.local", Share: "kids", MountPoint: ""},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing NAS mount point")
	}
}
