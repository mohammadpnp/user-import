package user_test

import (
	"testing"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
)

func TestNewUserValid(t *testing.T) {
	t.Parallel()

	u, err := domain.NewUser(
		"a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e",
		"Alice",
		"alice@example.com",
		"1234567890",
		[]domain.Address{{
			Street:  "123 Main St",
			City:    "Austin",
			State:   "Texas",
			ZipCode: "78701",
			Country: "USA",
		}},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if u.Email != "alice@example.com" {
		t.Fatalf("unexpected email: %s", u.Email)
	}
}

func TestNewUserInvalidEmail(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUser(
		"a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e",
		"Alice",
		"alice-at-example.com",
		"1234567890",
		nil,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != domain.ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestNewUserInvalidAddress(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUser(
		"a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e",
		"Alice",
		"alice@example.com",
		"1234567890",
		[]domain.Address{{
			Street:  "",
			City:    "Austin",
			State:   "Texas",
			ZipCode: "78701",
			Country: "USA",
		}},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != domain.ErrInvalidAddress {
		t.Fatalf("expected ErrInvalidAddress, got %v", err)
	}
}
