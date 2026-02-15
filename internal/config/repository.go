package config

import "github.com/LittleClubFoot/portkey-player/pkg/models"

// Repository defines the interface for config persistence.
type Repository interface {
	Load(path string) (*models.Config, error)
	Save(path string, config *models.Config) error
}
