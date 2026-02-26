package user_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/mohammadpnp/user-import/internal/application/user"
	domain "github.com/mohammadpnp/user-import/internal/domain/user"
)

type fakeUserQueryRepo struct {
	user      *domain.User
	returnErr error
}

func (f *fakeUserQueryRepo) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	return f.user, nil
}

func TestGetUserByIDSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeUserQueryRepo{user: &domain.User{
		ID:          "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e",
		Name:        "Alice",
		Email:       "alice@example.com",
		PhoneNumber: "1234567890",
		Addresses: []domain.Address{{
			Street:  "1 Main",
			City:    "Austin",
			State:   "TX",
			ZipCode: "78701",
			Country: "USA",
		}},
	}}

	uc := app.NewGetUserByID(repo)

	out, err := uc.Execute(context.Background(), app.GetUserByIDInput{ID: "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID != "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e" {
		t.Fatalf("unexpected id: %s", out.ID)
	}
	if len(out.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(out.Addresses))
	}
}

func TestGetUserByIDInvalidID(t *testing.T) {
	t.Parallel()

	uc := app.NewGetUserByID(&fakeUserQueryRepo{})

	_, err := uc.Execute(context.Background(), app.GetUserByIDInput{ID: "not-a-uuid"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, app.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	t.Parallel()

	uc := app.NewGetUserByID(&fakeUserQueryRepo{returnErr: domain.ErrUserNotFound})

	_, err := uc.Execute(context.Background(), app.GetUserByIDInput{ID: "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, app.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByIDRepositoryError(t *testing.T) {
	t.Parallel()

	uc := app.NewGetUserByID(&fakeUserQueryRepo{returnErr: errors.New("db down")})

	_, err := uc.Execute(context.Background(), app.GetUserByIDInput{ID: "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, app.ErrGetUserByID) {
		t.Fatalf("expected ErrGetUserByID, got %v", err)
	}
}
