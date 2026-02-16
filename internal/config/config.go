package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// FileRepository loads configuration from JSON files.
type FileRepository struct{}

// NewFileRepository creates a new file-based config loader.
func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) Load(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}
