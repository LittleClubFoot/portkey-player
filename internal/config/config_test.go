package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileRepository_LoadValid(t *testing.T) {
	content := `{
		"version": "1.0",
		"media": {
			"1001": {
				"path": "/media/test.mp4",
				"title": "Test Video",
				"type": "movie"
			}
		},
		"rules": {
			"max_plays_per_day": 5
		},
		"hardware": {
			"player": "mpv"
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repo := NewFileRepository()
	cfg, err := repo.Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}
	if len(cfg.Media) != 1 {
		t.Errorf("expected 1 media entry, got %d", len(cfg.Media))
	}
	if cfg.Media["1001"].Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %s", cfg.Media["1001"].Title)
	}
}

func TestFileRepository_LoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}

	repo := NewFileRepository()
	_, err := repo.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFileRepository_LoadMissingFile(t *testing.T) {
	repo := NewFileRepository()
	_, err := repo.Load("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
