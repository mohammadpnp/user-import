package user

import (
	"context"
	"time"
)

type ImportJobRepository interface {
	Enqueue(ctx context.Context, sourcePath string) (string, error)
	ClaimNext(ctx context.Context, leaseDuration time.Duration) (*ImportJob, error)
	Heartbeat(ctx context.Context, jobID string, leaseDuration time.Duration) error
	UpdateProgress(ctx context.Context, jobID string, progress ImportProgress) error
	Complete(ctx context.Context, jobID string, summary ImportSummary) error
	Requeue(ctx context.Context, jobID string, reason string) error
	Fail(ctx context.Context, jobID string, reason string) error
}

type UserBulkImporter interface {
	ImportChunk(ctx context.Context, jobID string, users []User) (ImportChunkResult, error)
}

type UserQueryRepository interface {
	GetByID(ctx context.Context, userID string) (*User, error)
}
