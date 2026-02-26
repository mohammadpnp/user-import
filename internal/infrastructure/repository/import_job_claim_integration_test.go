package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
	"github.com/mohammadpnp/user-import/internal/infrastructure/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestImportJobRepositoryClaimAndLifecycleIntegration(t *testing.T) {
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
	if err := db.Exec("DELETE FROM import_jobs").Error; err != nil {
		t.Fatalf("failed to cleanup import_jobs: %v", err)
	}

	repo := repository.NewImportJobRepository(db)

	jobID, err := repo.Enqueue(context.Background(), "users_data.json")
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	claimed, err := repo.ClaimNext(context.Background(), 30*time.Second)
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected claimed job")
	}
	if claimed.ID != jobID {
		t.Fatalf("unexpected job id: %s", claimed.ID)
	}

	if err := repo.Heartbeat(context.Background(), claimed.ID, 30*time.Second); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	progress := domain.ImportProgress{
		ProcessedCount: 10,
		ImportedCount:  8,
		UpdatedCount:   1,
		SkippedCount:   1,
		FailedCount:    0,
	}
	if err := repo.UpdateProgress(context.Background(), claimed.ID, progress); err != nil {
		t.Fatalf("update progress failed: %v", err)
	}

	summary := domain.ImportSummary{
		ProcessedCount: 10,
		ImportedCount:  8,
		UpdatedCount:   1,
		SkippedCount:   1,
		FailedCount:    0,
	}
	if err := repo.Complete(context.Background(), claimed.ID, summary); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
}
