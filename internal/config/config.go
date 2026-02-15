package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// FileRepository implements Repository by reading/writing JSON files.
type FileRepository struct {
	mu sync.RWMutex
}

// NewFileRepository creates a new file-based config repository.
func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) Load(path string) (*models.Config, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *FileRepository) Save(path string, config *models.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := Validate(config); err != nil {
		return fmt.Errorf("validating config before save: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
