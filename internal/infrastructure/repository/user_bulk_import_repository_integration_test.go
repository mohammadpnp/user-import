package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/mohammadpnp/user-import/internal/domain/user"
	"github.com/mohammadpnp/user-import/internal/infrastructure/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUserBulkImportRepositoryImportChunkIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	schemaSQL := `
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE TABLE IF NOT EXISTS users (
      id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
      name VARCHAR(255) NOT NULL,
      email VARCHAR(320) NOT NULL UNIQUE,
      phone_number VARCHAR(32) NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS addresses (
      id BIGSERIAL PRIMARY KEY,
      user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      street VARCHAR(255) NOT NULL,
      city VARCHAR(120) NOT NULL,
      state VARCHAR(120) NOT NULL,
      zip_code VARCHAR(20) NOT NULL,
      country VARCHAR(120) NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE UNLOGGED TABLE IF NOT EXISTS stg_users (
      job_id UUID NOT NULL,
      row_index BIGINT NOT NULL,
      external_id TEXT,
      name TEXT NOT NULL,
      email TEXT NOT NULL,
      phone_number TEXT NOT NULL
    );
    CREATE UNLOGGED TABLE IF NOT EXISTS stg_addresses (
      job_id UUID NOT NULL,
      row_index BIGINT NOT NULL,
      user_external_id TEXT,
      user_email TEXT NOT NULL,
      street TEXT NOT NULL,
      city TEXT NOT NULL,
      state TEXT NOT NULL,
      zip_code TEXT NOT NULL,
      country TEXT NOT NULL
    );
    `
	if err := gdb.Exec(schemaSQL).Error; err != nil {
		t.Fatalf("failed schema setup: %v", err)
	}
	cleanupSQL := `
    DELETE FROM addresses;
    DELETE FROM users;
    DELETE FROM stg_addresses;
    DELETE FROM stg_users;
    `
	if err := gdb.Exec(cleanupSQL).Error; err != nil {
		t.Fatalf("failed cleanup: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to create pgx pool: %v", err)
	}
	defer pool.Close()

	repo := repository.NewUserBulkImportRepository(pool)

	users := []domain.User{{
		ID:          "f7bc5d17-e7b2-49a1-9fd2-061b58f44f85",
		Name:        "Alice",
		Email:       "alice@example.com",
		PhoneNumber: "1111111111",
		Addresses: []domain.Address{{
			Street:  "1 Main",
			City:    "Austin",
			State:   "TX",
			ZipCode: "78701",
			Country: "USA",
		}},
	}}

	result, err := repo.ImportChunk(context.Background(), "4955eb4d-c7f2-42f6-80ca-33838ce37c31", users)
	if err != nil {
		t.Fatalf("import chunk failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("expected imported=1, got %d", result.ImportedCount)
	}

	users[0].PhoneNumber = "2222222222"
	users[0].Addresses = []domain.Address{{
		Street:  "2 Main",
		City:    "Austin",
		State:   "TX",
		ZipCode: "78702",
		Country: "USA",
	}}
	result, err = repo.ImportChunk(context.Background(), "26a700f4-6765-4dce-b1a7-3a18f2fd4f56", users)
	if err != nil {
		t.Fatalf("import chunk update failed: %v", err)
	}
	if result.UpdatedCount != 1 {
		t.Fatalf("expected updated=1, got %d", result.UpdatedCount)
	}

	var addressCount int64
	if err := gdb.Raw("SELECT COUNT(*) FROM addresses WHERE user_id = ?", users[0].ID).Scan(&addressCount).Error; err != nil {
		t.Fatalf("count addresses failed: %v", err)
	}
	if addressCount != 1 {
		t.Fatalf("expected 1 address after replacement, got %d", addressCount)
	}
}
