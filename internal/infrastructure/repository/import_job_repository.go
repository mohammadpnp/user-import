package repository

import (
	"context"
	"fmt"
	"time"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
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

func (r *ImportJobRepository) ClaimNext(ctx context.Context, leaseDuration time.Duration) (*domain.ImportJob, error) {
	var job models.ImportJob

	leaseSeconds := int(leaseDuration.Seconds())
	if leaseSeconds <= 0 {
		leaseSeconds = 60
	}

	query := `
WITH candidate AS (
    SELECT id
    FROM import_jobs
    WHERE
      (status = 'queued' OR (status = 'running' AND lease_expires_at < NOW()))
      AND attempts < max_attempts
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE import_jobs j
SET
    status = 'running',
    attempts = j.attempts + 1,
    started_at = COALESCE(j.started_at, NOW()),
    heartbeat_at = NOW(),
    lease_expires_at = NOW() + make_interval(secs => ?),
    error_message = NULL,
    updated_at = NOW()
FROM candidate
WHERE j.id = candidate.id
RETURNING j.*;
`

	if err := r.db.WithContext(ctx).Raw(query, leaseSeconds).Scan(&job).Error; err != nil {
		return nil, fmt.Errorf("claim import job: %w", err)
	}

	if job.ID == "" {
		return nil, nil
	}

	return &domain.ImportJob{
		ID:          job.ID,
		SourcePath:  job.SourcePath,
		Status:      job.Status,
		Attempts:    job.Attempts,
		MaxAttempts: job.MaxAttempts,
	}, nil
}

func (r *ImportJobRepository) Heartbeat(ctx context.Context, jobID string, leaseDuration time.Duration) error {
	leaseSeconds := int(leaseDuration.Seconds())
	if leaseSeconds <= 0 {
		leaseSeconds = 60
	}

	result := r.db.WithContext(ctx).Exec(`
UPDATE import_jobs
SET
  heartbeat_at = NOW(),
  lease_expires_at = NOW() + make_interval(secs => ?),
  updated_at = NOW()
WHERE id = ? AND status = 'running'
`, leaseSeconds, jobID)
	if result.Error != nil {
		return fmt.Errorf("heartbeat import job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("heartbeat import job: job not running")
	}
	return nil
}

func (r *ImportJobRepository) UpdateProgress(ctx context.Context, jobID string, progress domain.ImportProgress) error {
	result := r.db.WithContext(ctx).Exec(`
UPDATE import_jobs
SET
  progress_processed = ?,
  progress_total = ?,
  imported_count = ?,
  updated_count = ?,
  skipped_count = ?,
  failed_count = ?,
  updated_at = NOW()
WHERE id = ?
`, progress.ProcessedCount, progress.ProcessedCount, progress.ImportedCount, progress.UpdatedCount, progress.SkippedCount, progress.FailedCount, jobID)
	if result.Error != nil {
		return fmt.Errorf("update import job progress: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update import job progress: job not found")
	}
	return nil
}

func (r *ImportJobRepository) Complete(ctx context.Context, jobID string, summary domain.ImportSummary) error {
	result := r.db.WithContext(ctx).Exec(`
UPDATE import_jobs
SET
  status = 'succeeded',
  progress_processed = ?,
  progress_total = ?,
  imported_count = ?,
  updated_count = ?,
  skipped_count = ?,
  failed_count = ?,
  error_message = NULL,
  lease_expires_at = NULL,
  heartbeat_at = NOW(),
  finished_at = NOW(),
  updated_at = NOW()
WHERE id = ?
`, summary.ProcessedCount, summary.ProcessedCount, summary.ImportedCount, summary.UpdatedCount, summary.SkippedCount, summary.FailedCount, jobID)
	if result.Error != nil {
		return fmt.Errorf("complete import job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("complete import job: job not found")
	}
	return nil
}

func (r *ImportJobRepository) Requeue(ctx context.Context, jobID string, reason string) error {
	result := r.db.WithContext(ctx).Exec(`
UPDATE import_jobs
SET
  status = 'queued',
  lease_expires_at = NULL,
  heartbeat_at = NOW(),
  error_message = ?,
  updated_at = NOW()
WHERE id = ?
`, reason, jobID)
	if result.Error != nil {
		return fmt.Errorf("requeue import job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("requeue import job: job not found")
	}
	return nil
}

func (r *ImportJobRepository) Fail(ctx context.Context, jobID string, reason string) error {
	result := r.db.WithContext(ctx).Exec(`
UPDATE import_jobs
SET
  status = 'failed',
  lease_expires_at = NULL,
  heartbeat_at = NOW(),
  error_message = ?,
  finished_at = NOW(),
  updated_at = NOW()
WHERE id = ?
`, reason, jobID)
	if result.Error != nil {
		return fmt.Errorf("fail import job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fail import job: job not found")
	}
	return nil
}
