package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileChecker provides file existence and media discovery helpers.
type FileChecker struct {
	mediaDir string
}

// NewFileChecker creates a checker rooted at the given media directory.
func NewFileChecker(mediaDir string) *FileChecker {
	return &FileChecker{mediaDir: mediaDir}
}

// Exists returns whether the given path exists on disk.
func (fc *FileChecker) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking path %s: %w", path, err)
}

// ListMedia returns all supported media files under the media directory.
func (fc *FileChecker) ListMedia() ([]string, error) {
	supportedExts := map[string]bool{
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".mp3": true, ".aac": true, ".flac": true,
	}

	var files []string
	err := filepath.Walk(fc.mediaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if supportedExts[ext] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking media directory %s: %w", fc.mediaDir, err)
	}
	return files, nil
}
