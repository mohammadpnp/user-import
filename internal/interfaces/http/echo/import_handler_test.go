package echo_test

import (
	"bytes"
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

type fakeImportUseCase struct {
	output app.StartImportUsersFromJSONOutput
	err    error
}

func (f *fakeImportUseCase) Execute(ctx context.Context, in app.StartImportUsersFromJSONInput) (app.StartImportUsersFromJSONOutput, error) {
	if f.err != nil {
		return app.StartImportUsersFromJSONOutput{}, f.err
	}
	return f.output, nil
}

func TestImportHandlerSuccess(t *testing.T) {
	t.Parallel()

	e := echo.New()
	handler := httpecho.NewImportHandler(&fakeImportUseCase{output: app.StartImportUsersFromJSONOutput{
		JobID:  "job-1",
		Status: "queued",
	}})
	httpecho.RegisterRoutes(e, handler)

	body := []byte(`{"source_path":"users_data.json"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/users", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unexpected json: %v", err)
	}

	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected data payload: %#v", got["data"])
	}
	if data["job_id"] != "job-1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImportHandlerBadJSON(t *testing.T) {
	t.Parallel()

	e := echo.New()
	handler := httpecho.NewImportHandler(&fakeImportUseCase{})
	httpecho.RegisterRoutes(e, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/users", bytes.NewReader([]byte(`{"source_path":`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestImportHandlerInvalidSource(t *testing.T) {
	t.Parallel()

	e := echo.New()
	handler := httpecho.NewImportHandler(&fakeImportUseCase{err: app.ErrInvalidImportSource})
	httpecho.RegisterRoutes(e, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/users", bytes.NewReader([]byte(`{"source_path":""}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestImportHandlerInternalError(t *testing.T) {
	t.Parallel()

	e := echo.New()
	handler := httpecho.NewImportHandler(&fakeImportUseCase{err: errors.New("boom")})
	httpecho.RegisterRoutes(e, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/users", bytes.NewReader([]byte(`{"source_path":"users_data.json"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
