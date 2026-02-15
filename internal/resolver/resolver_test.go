package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

func testMedia() map[string]models.MediaEntry {
	return map[string]models.MediaEntry{
		"1001": {
			Path:     "/media/movies/frozen.mp4",
			Title:    "Frozen",
			Duration: 102,
			Type:     "movie",
		},
		"2001": {
			Path:     "smb://nas.local/kids/peppa.mp4",
			Title:    "Peppa Pig",
			Duration: 5,
			Type:     "episode",
		},
	}
}

func TestConfigResolver_ResolveFound(t *testing.T) {
	r := NewConfigResolver(testMedia())
	item, err := r.Resolve("1001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.TagID != "1001" {
		t.Errorf("expected tag ID 1001, got %s", item.TagID)
	}
	if item.Title != "Frozen" {
		t.Errorf("expected title Frozen, got %s", item.Title)
	}
	if item.Path != "/media/movies/frozen.mp4" {
		t.Errorf("expected path /media/movies/frozen.mp4, got %s", item.Path)
	}
	if item.Duration != 102 {
		t.Errorf("expected duration 102, got %d", item.Duration)
	}
	if item.Type != "movie" {
		t.Errorf("expected type movie, got %s", item.Type)
	}
}

func TestConfigResolver_ResolveNotFound(t *testing.T) {
	r := NewConfigResolver(testMedia())
	_, err := r.Resolve("9999")
	if err == nil {
		t.Fatal("expected error for unknown tag")
	}
}

func TestConfigResolver_ExistsLocalFile(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.mp4")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewConfigResolver(testMedia())

	exists, err := r.Exists(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}
}

func TestConfigResolver_ExistsLocalFileMissing(t *testing.T) {
	r := NewConfigResolver(testMedia())
	exists, err := r.Exists("/nonexistent/file.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected file to not exist")
	}
}

func TestConfigResolver_ExistsNetworkPath(t *testing.T) {
	r := NewConfigResolver(testMedia())

	// SMB paths are assumed reachable.
	exists, err := r.Exists("smb://nas.local/kids/peppa.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected SMB path to be treated as existing")
	}

	// NFS paths are assumed reachable.
	exists, err = r.Exists("nfs://nas.local/kids/peppa.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected NFS path to be treated as existing")
	}
}

func TestConfigResolver_ResolveAllEntries(t *testing.T) {
	media := testMedia()
	r := NewConfigResolver(media)

	for tagID, entry := range media {
		item, err := r.Resolve(tagID)
		if err != nil {
			t.Errorf("resolve failed for tag %s: %v", tagID, err)
			continue
		}
		if item.Title != entry.Title {
			t.Errorf("tag %s: expected title %q, got %q", tagID, entry.Title, item.Title)
		}
	}
}
