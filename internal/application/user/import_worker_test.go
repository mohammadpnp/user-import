package user_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	app "github.com/mohammadpnp/user-import/internal/application/user"
	domain "github.com/mohammadpnp/user-import/internal/domain/user"
)

type fakeWorkerRepo struct {
	claimedJob      *domain.ImportJob
	claimErr        error
	progressCalls   []domain.ImportProgress
	completeSummary *domain.ImportSummary
	requeueCalled   bool
	failCalled      bool
	failMessage     string
}

func (f *fakeWorkerRepo) Enqueue(ctx context.Context, sourcePath string) (string, error) {
	return "", nil
}

func (f *fakeWorkerRepo) ClaimNext(ctx context.Context, leaseDuration time.Duration) (*domain.ImportJob, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	job := f.claimedJob
	f.claimedJob = nil
	return job, nil
}

func (f *fakeWorkerRepo) Heartbeat(ctx context.Context, jobID string, leaseDuration time.Duration) error {
	return nil
}

func (f *fakeWorkerRepo) UpdateProgress(ctx context.Context, jobID string, progress domain.ImportProgress) error {
	f.progressCalls = append(f.progressCalls, progress)
	return nil
}

func (f *fakeWorkerRepo) Complete(ctx context.Context, jobID string, summary domain.ImportSummary) error {
	f.completeSummary = &summary
	return nil
}

func (f *fakeWorkerRepo) Requeue(ctx context.Context, jobID string, reason string) error {
	f.requeueCalled = true
	f.failMessage = reason
	return nil
}

func (f *fakeWorkerRepo) Fail(ctx context.Context, jobID string, reason string) error {
	f.failCalled = true
	f.failMessage = reason
	return nil
}

type fakeSource struct {
	data string
	err  error
}

func (f *fakeSource) Open(ctx context.Context, sourcePath string) (io.ReadCloser, error) {
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(strings.NewReader(f.data)), nil
}

type fakeBulkImporter struct {
	result app.ImportChunkResult
	err    error
	calls  int
	rows   int
}

func (f *fakeBulkImporter) ImportChunk(ctx context.Context, jobID string, users []domain.User) (app.ImportChunkResult, error) {
	f.calls++
	f.rows += len(users)
	if f.err != nil {
		return app.ImportChunkResult{}, f.err
	}
	return f.result, nil
}

func TestImportWorkerProcessJobSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	source := &fakeSource{data: `[
      {
        "id":"ab5e6ab5-ae1a-4a52-94f3-9c266d266c79",
        "name":"Alice",
        "email":"alice@example.com",
        "phone_number":"1111111111",
        "addresses":[{"street":"1 Main","city":"Austin","state":"TX","zip_code":"78701","country":"USA"}]
      },
      {
        "id":"",
        "name":"Broken",
        "email":"bad-email",
        "phone_number":"2222222222",
        "addresses":[{"street":"2 Main","city":"Austin","state":"TX","zip_code":"78702","country":"USA"}]
      }
    ]`}
	importer := &fakeBulkImporter{result: app.ImportChunkResult{ImportedCount: 1, UpdatedCount: 0, SkippedCount: 0}}

	worker := app.NewImportWorker(repo, source, importer, app.ImportWorkerConfig{ChunkSize: 1, LeaseDuration: 30 * time.Second})

	err := worker.ProcessJob(context.Background(), domain.ImportJob{
		ID:          "job-1",
		SourcePath:  "users_data.json",
		Attempts:    1,
		MaxAttempts: 5,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if importer.calls != 1 {
		t.Fatalf("expected 1 importer call, got %d", importer.calls)
	}
	if importer.rows != 1 {
		t.Fatalf("expected 1 imported row, got %d", importer.rows)
	}

	if repo.completeSummary == nil {
		t.Fatal("expected complete summary")
	}
	if repo.completeSummary.ImportedCount != 1 {
		t.Fatalf("expected imported=1, got %d", repo.completeSummary.ImportedCount)
	}
	if repo.completeSummary.FailedCount != 1 {
		t.Fatalf("expected failed=1, got %d", repo.completeSummary.FailedCount)
	}
	if repo.completeSummary.SkippedCount != 1 {
		t.Fatalf("expected skipped=1, got %d", repo.completeSummary.SkippedCount)
	}
	if repo.completeSummary.ProcessedCount != 2 {
		t.Fatalf("expected processed=2, got %d", repo.completeSummary.ProcessedCount)
	}
	if len(repo.progressCalls) == 0 {
		t.Fatal("expected progress updates")
	}
}

func TestImportWorkerProcessJobRetryableFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	source := &fakeSource{data: `[{"id":"ab5e6ab5-ae1a-4a52-94f3-9c266d266c79","name":"Alice","email":"alice@example.com","phone_number":"1111111111","addresses":[]}]`}
	importer := &fakeBulkImporter{err: errors.New("copy failed")}

	worker := app.NewImportWorker(repo, source, importer, app.ImportWorkerConfig{ChunkSize: 10, LeaseDuration: 30 * time.Second})

	err := worker.ProcessJob(context.Background(), domain.ImportJob{ID: "job-1", SourcePath: "users_data.json", Attempts: 1, MaxAttempts: 3})
	if err == nil {
		t.Fatal("expected error")
	}
	if !repo.requeueCalled {
		t.Fatal("expected requeue to be called")
	}
	if repo.failCalled {
		t.Fatal("did not expect fail to be called")
	}
}

func TestImportWorkerProcessJobTerminalFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	source := &fakeSource{data: `[{"id":"ab5e6ab5-ae1a-4a52-94f3-9c266d266c79","name":"Alice","email":"alice@example.com","phone_number":"1111111111","addresses":[]}]`}
	importer := &fakeBulkImporter{err: errors.New("copy failed")}

	worker := app.NewImportWorker(repo, source, importer, app.ImportWorkerConfig{ChunkSize: 10, LeaseDuration: 30 * time.Second})

	err := worker.ProcessJob(context.Background(), domain.ImportJob{ID: "job-1", SourcePath: "users_data.json", Attempts: 3, MaxAttempts: 3})
	if err == nil {
		t.Fatal("expected error")
	}
	if !repo.failCalled {
		t.Fatal("expected fail to be called")
	}
	if repo.requeueCalled {
		t.Fatal("did not expect requeue to be called")
	}
}
