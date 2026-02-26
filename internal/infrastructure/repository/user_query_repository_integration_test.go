package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
	"github.com/mohammadpnp/user-import/internal/infrastructure/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUserQueryRepositoryGetByIDIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	schemaSQL := `
    CREATE TABLE IF NOT EXISTS users (
      id UUID PRIMARY KEY,
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
    `
	if err := db.Exec(schemaSQL).Error; err != nil {
		t.Fatalf("failed schema setup: %v", err)
	}

	userID := "d5987b5f-506d-4d84-934f-d5b5535a64e8"
	if err := db.Exec("DELETE FROM addresses WHERE user_id = ?", userID).Error; err != nil {
		t.Fatalf("cleanup addresses failed: %v", err)
	}
	if err := db.Exec("DELETE FROM users WHERE id = ?", userID).Error; err != nil {
		t.Fatalf("cleanup users failed: %v", err)
	}

	insertUserSQL := `
    INSERT INTO users (id, name, email, phone_number)
    VALUES (?, ?, ?, ?)
    `
	if err := db.Exec(insertUserSQL, userID, "Alice", "alice-query@example.com", "1234567890").Error; err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	insertAddressSQL := `
    INSERT INTO addresses (user_id, street, city, state, zip_code, country)
    VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)
    `
	if err := db.Exec(insertAddressSQL,
		userID, "1 Main", "Austin", "TX", "78701", "USA",
		userID, "2 Main", "Austin", "TX", "78702", "USA",
	).Error; err != nil {
		t.Fatalf("insert addresses failed: %v", err)
	}

	repo := repository.NewUserQueryRepository(db)

	got, err := repo.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != userID {
		t.Fatalf("unexpected id: %s", got.ID)
	}
	if len(got.Addresses) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(got.Addresses))
	}

	_, err = repo.GetByID(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
