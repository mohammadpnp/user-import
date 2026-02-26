package user_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/mohammadpnp/user-import/internal/application/user"
)

type fakeImportJobRepository struct {
	jobID     string
	called    bool
	gotPath   string
	returnErr error
}

func (f *fakeImportJobRepository) Enqueue(ctx context.Context, sourcePath string) (string, error) {
	f.called = true
	f.gotPath = sourcePath
	if f.returnErr != nil {
		return "", f.returnErr
	}
	return f.jobID, nil
}

func TestStartImportUsersFromJSONSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeImportJobRepository{jobID: "job-1"}
	uc := app.NewStartImportUsersFromJSON(repo)

	out, err := uc.Execute(context.Background(), app.StartImportUsersFromJSONInput{
		SourcePath: "users_data.json",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if repo.gotPath != "users_data.json" {
		t.Fatalf("unexpected source path: %s", repo.gotPath)
	}
	if out.JobID != "job-1" {
		t.Fatalf("unexpected job id: %s", out.JobID)
	}
	if out.Status != "queued" {
		t.Fatalf("unexpected status: %s", out.Status)
	}
}

func TestStartImportUsersFromJSONInvalidPath(t *testing.T) {
	t.Parallel()

	uc := app.NewStartImportUsersFromJSON(&fakeImportJobRepository{})

	_, err := uc.Execute(context.Background(), app.StartImportUsersFromJSONInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, app.ErrInvalidImportSource) {
		t.Fatalf("expected ErrInvalidImportSource, got %v", err)
	}
}

func TestStartImportUsersFromJSONRepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db down")
	uc := app.NewStartImportUsersFromJSON(&fakeImportJobRepository{returnErr: repoErr})

	_, err := uc.Execute(context.Background(), app.StartImportUsersFromJSONInput{SourcePath: "users_data.json"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, app.ErrEnqueueImportJob) {
		t.Fatalf("expected ErrEnqueueImportJob, got %v", err)
	}
}
