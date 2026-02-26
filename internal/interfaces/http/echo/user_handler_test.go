package echo_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	app "github.com/mohammadpnp/user-import/internal/application/user"
	httpecho "github.com/mohammadpnp/user-import/internal/interfaces/http/echo"
)

type fakeGetUserUseCase struct {
	out app.GetUserByIDOutput
	err error
}

func (f *fakeGetUserUseCase) Execute(ctx context.Context, in app.GetUserByIDInput) (app.GetUserByIDOutput, error) {
	if f.err != nil {
		return app.GetUserByIDOutput{}, f.err
	}
	return f.out, nil
}

func TestGetUserByIDHandlerSuccess(t *testing.T) {
	t.Parallel()

	e := echo.New()
	userHandler := httpecho.NewUserHandler(&fakeGetUserUseCase{out: app.GetUserByIDOutput{
		ID:          "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e",
		Name:        "Alice",
		Email:       "alice@example.com",
		PhoneNumber: "1234567890",
		Addresses: []app.GetUserAddressOutput{{
			Street:  "1 Main",
			City:    "Austin",
			State:   "TX",
			ZipCode: "78701",
			Country: "USA",
		}},
	}})
	httpecho.RegisterRoutes(e, nil, userHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unexpected json: %v", err)
	}

	data := got["data"].(map[string]any)
	if data["id"] != "a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e" {
		t.Fatalf("unexpected id: %#v", data["id"])
	}
}

func TestGetUserByIDHandlerInvalidID(t *testing.T) {
	t.Parallel()

	e := echo.New()
	userHandler := httpecho.NewUserHandler(&fakeGetUserUseCase{err: app.ErrInvalidUserID})
	httpecho.RegisterRoutes(e, nil, userHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/not-uuid", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetUserByIDHandlerNotFound(t *testing.T) {
	t.Parallel()

	e := echo.New()
	userHandler := httpecho.NewUserHandler(&fakeGetUserUseCase{err: app.ErrUserNotFound})
	httpecho.RegisterRoutes(e, nil, userHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetUserByIDHandlerInternalError(t *testing.T) {
	t.Parallel()

	e := echo.New()
	userHandler := httpecho.NewUserHandler(&fakeGetUserUseCase{err: errors.New("boom")})
	httpecho.RegisterRoutes(e, nil, userHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/a3f91a91-7fdd-43bf-bfd2-00bc02f6c53e", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
