package models

import "time"

type ImportJob struct {
	ID                string  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	SourcePath        string  `gorm:"type:text;not null"`
	Status            string  `gorm:"type:text;not null"`
	ProgressProcessed int64   `gorm:"not null;default:0"`
	ProgressTotal     int64   `gorm:"not null;default:0"`
	ImportedCount     int64   `gorm:"not null;default:0"`
	UpdatedCount      int64   `gorm:"not null;default:0"`
	SkippedCount      int64   `gorm:"not null;default:0"`
	FailedCount       int64   `gorm:"not null;default:0"`
	Attempts          int     `gorm:"not null;default:0"`
	MaxAttempts       int     `gorm:"not null;default:5"`
	ErrorMessage      *string `gorm:"type:text"`
	HeartbeatAt       *time.Time
	LeaseExpiresAt    *time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (ImportJob) TableName() string {
	return "import_jobs"
}
