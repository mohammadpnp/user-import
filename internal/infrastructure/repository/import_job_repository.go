package repository

import (
	"context"
	"fmt"

	"github.com/mohammadpnp/user-import/internal/infrastructure/db/models"
	"gorm.io/gorm"
)

type ImportJobRepository struct {
	db *gorm.DB
}

func NewImportJobRepository(db *gorm.DB) *ImportJobRepository {
	return &ImportJobRepository{db: db}
}

func (r *ImportJobRepository) Enqueue(ctx context.Context, sourcePath string) (string, error) {
	job := models.ImportJob{
		SourcePath: sourcePath,
		Status:     "queued",
	}

	if err := r.db.WithContext(ctx).Create(&job).Error; err != nil {
		return "", fmt.Errorf("create import job: %w", err)
	}

	return job.ID, nil
}
