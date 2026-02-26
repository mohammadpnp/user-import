package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mohammadpnp/user-import/internal/infrastructure/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestImportJobRepositoryEnqueueIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	createSQL := `
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE TABLE IF NOT EXISTS import_jobs (
      id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
      source_path TEXT NOT NULL,
      status TEXT NOT NULL,
      progress_processed BIGINT NOT NULL DEFAULT 0,
      progress_total BIGINT NOT NULL DEFAULT 0,
      imported_count BIGINT NOT NULL DEFAULT 0,
      updated_count BIGINT NOT NULL DEFAULT 0,
      skipped_count BIGINT NOT NULL DEFAULT 0,
      failed_count BIGINT NOT NULL DEFAULT 0,
      attempts INT NOT NULL DEFAULT 0,
      max_attempts INT NOT NULL DEFAULT 5,
      error_message TEXT,
      heartbeat_at TIMESTAMPTZ,
      lease_expires_at TIMESTAMPTZ,
      started_at TIMESTAMPTZ,
      finished_at TIMESTAMPTZ,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      CHECK (status IN ('queued','running','succeeded','failed'))
    );
    `
	if err := db.Exec(createSQL).Error; err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	repo := repository.NewImportJobRepository(db)

	jobID, err := repo.Enqueue(context.Background(), "users_data.json")
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if strings.TrimSpace(jobID) == "" {
		t.Fatal("expected non-empty job id")
	}
}
