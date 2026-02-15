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
			Title: "Peppa Pig",
			Type:  "series",
			Episodes: []models.EpisodeEntry{
				{Season: 1, Episode: 1, Path: "/media/peppa/s01e01.mp4", Title: "Muddy Puddles", Duration: 5},
				{Season: 1, Episode: 2, Path: "/media/peppa/s01e02.mp4", Title: "Mr Dinosaur", Duration: 5},
				{Season: 1, Episode: 3, Path: "/media/peppa/s01e03.mp4", Title: "Best Friend", Duration: 5},
				{Season: 2, Episode: 1, Path: "/media/peppa/s02e01.mp4", Title: "Bubbles", Duration: 5},
			},
		},
		"3001": {
			Path:  "smb://nas.local/kids/peppa.mp4",
			Title: "Peppa Single",
			Type:  "episode",
		},
	}
}

// --- Basic resolve tests ---

func TestConfigResolver_ResolveMovie(t *testing.T) {
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

func TestConfigResolver_ResolveSeries_ReturnsFirstEpisode(t *testing.T) {
	r := NewConfigResolver(testMedia())
	item, err := r.Resolve("2001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Muddy Puddles" {
		t.Errorf("expected first episode title, got %s", item.Title)
	}
	if item.Season != 1 || item.Episode != 1 {
		t.Errorf("expected S01E01, got S%02dE%02d", item.Season, item.Episode)
	}
	if item.Type != "episode" {
		t.Errorf("expected type episode, got %s", item.Type)
	}
}

// --- IsSeries ---

func TestConfigResolver_IsSeries(t *testing.T) {
	r := NewConfigResolver(testMedia())

	if !r.IsSeries("2001") {
		t.Error("expected 2001 to be a series")
	}
	if r.IsSeries("1001") {
		t.Error("expected 1001 to not be a series")
	}
	if r.IsSeries("9999") {
		t.Error("expected unknown tag to not be a series")
	}
}

// --- ResolveEpisode ---

func TestConfigResolver_ResolveEpisode(t *testing.T) {
	r := NewConfigResolver(testMedia())

	item, err := r.ResolveEpisode("2001", 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Mr Dinosaur" {
		t.Errorf("expected Mr Dinosaur, got %s", item.Title)
	}
	if item.Season != 1 || item.Episode != 2 {
		t.Errorf("expected S01E02, got S%02dE%02d", item.Season, item.Episode)
	}
}

func TestConfigResolver_ResolveEpisode_CrossSeason(t *testing.T) {
	r := NewConfigResolver(testMedia())

	item, err := r.ResolveEpisode("2001", 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Bubbles" {
		t.Errorf("expected Bubbles, got %s", item.Title)
	}
}

func TestConfigResolver_ResolveEpisode_NotFound(t *testing.T) {
	r := NewConfigResolver(testMedia())

	_, err := r.ResolveEpisode("2001", 5, 99)
	if err == nil {
		t.Fatal("expected error for missing episode")
	}
}

func TestConfigResolver_ResolveEpisode_NotASeries(t *testing.T) {
	r := NewConfigResolver(testMedia())

	_, err := r.ResolveEpisode("1001", 1, 1)
	if err == nil {
		t.Fatal("expected error for non-series tag")
	}
}

// --- NextEpisode ---

func TestConfigResolver_NextEpisode(t *testing.T) {
	r := NewConfigResolver(testMedia())

	next, err := r.NextEpisode("2001", 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Title != "Mr Dinosaur" {
		t.Errorf("expected Mr Dinosaur, got %s", next.Title)
	}
}

func TestConfigResolver_NextEpisode_CrossSeason(t *testing.T) {
	r := NewConfigResolver(testMedia())

	// S01E03 → S02E01
	next, err := r.NextEpisode("2001", 1, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Season != 2 || next.Episode != 1 {
		t.Errorf("expected S02E01, got S%02dE%02d", next.Season, next.Episode)
	}
}

func TestConfigResolver_NextEpisode_AtEnd(t *testing.T) {
	r := NewConfigResolver(testMedia())

	// S02E01 is the last episode.
	next, err := r.NextEpisode("2001", 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != nil {
		t.Error("expected nil for last episode")
	}
}

func TestConfigResolver_NextEpisode_UnknownTag(t *testing.T) {
	r := NewConfigResolver(testMedia())
	_, err := r.NextEpisode("9999", 1, 1)
	if err == nil {
		t.Fatal("expected error for unknown tag")
	}
}

// --- PrevEpisode ---

func TestConfigResolver_PrevEpisode(t *testing.T) {
	r := NewConfigResolver(testMedia())

	prev, err := r.PrevEpisode("2001", 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prev.Title != "Muddy Puddles" {
		t.Errorf("expected Muddy Puddles, got %s", prev.Title)
	}
}

func TestConfigResolver_PrevEpisode_AtStart(t *testing.T) {
	r := NewConfigResolver(testMedia())

	prev, err := r.PrevEpisode("2001", 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prev != nil {
		t.Error("expected nil for first episode")
	}
}

func TestConfigResolver_PrevEpisode_CrossSeason(t *testing.T) {
	r := NewConfigResolver(testMedia())

	// S02E01 → S01E03
	prev, err := r.PrevEpisode("2001", 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prev.Season != 1 || prev.Episode != 3 {
		t.Errorf("expected S01E03, got S%02dE%02d", prev.Season, prev.Episode)
	}
}

// --- FirstEpisode ---

func TestConfigResolver_FirstEpisode(t *testing.T) {
	r := NewConfigResolver(testMedia())

	first, err := r.FirstEpisode("2001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Season != 1 || first.Episode != 1 {
		t.Errorf("expected S01E01, got S%02dE%02d", first.Season, first.Episode)
	}
}

// --- EpisodeCount ---

func TestConfigResolver_EpisodeCount(t *testing.T) {
	r := NewConfigResolver(testMedia())

	if count := r.EpisodeCount("2001"); count != 4 {
		t.Errorf("expected 4 episodes, got %d", count)
	}
	if count := r.EpisodeCount("1001"); count != 0 {
		t.Errorf("expected 0 episodes for movie, got %d", count)
	}
	if count := r.EpisodeCount("9999"); count != 0 {
		t.Errorf("expected 0 for unknown tag, got %d", count)
	}
}

// --- EpisodeToItem title fallback ---

func TestConfigResolver_EpisodeToItem_FallbackTitle(t *testing.T) {
	media := map[string]models.MediaEntry{
		"test": {
			Title: "My Show",
			Type:  "series",
			Episodes: []models.EpisodeEntry{
				{Season: 3, Episode: 7, Path: "/test.mp4", Title: ""},
			},
		},
	}
	r := NewConfigResolver(media)

	item, err := r.ResolveEpisode("test", 3, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "My Show S03E07" {
		t.Errorf("expected fallback title 'My Show S03E07', got %s", item.Title)
	}
}

// --- Exists ---

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

	exists, _ := r.Exists("smb://nas.local/kids/peppa.mp4")
	if !exists {
		t.Error("expected SMB path to be treated as existing")
	}

	exists, _ = r.Exists("nfs://nas.local/kids/peppa.mp4")
	if !exists {
		t.Error("expected NFS path to be treated as existing")
	}
}
