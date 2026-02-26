package user

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type StartImportUsersFromJSONInput struct {
	SourcePath string
}

type StartImportUsersFromJSONOutput struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type StartImportUsersFromJSON interface {
	Execute(ctx context.Context, in StartImportUsersFromJSONInput) (StartImportUsersFromJSONOutput, error)
}

type importJobEnqueuer interface {
	Enqueue(ctx context.Context, sourcePath string) (string, error)
}

type startImportUsersFromJSON struct {
	importJobRepo importJobEnqueuer
}

func NewStartImportUsersFromJSON(importJobRepo importJobEnqueuer) StartImportUsersFromJSON {
	return &startImportUsersFromJSON{importJobRepo: importJobRepo}
}

func (uc *startImportUsersFromJSON) Execute(ctx context.Context, in StartImportUsersFromJSONInput) (StartImportUsersFromJSONOutput, error) {
	sourcePath := strings.TrimSpace(in.SourcePath)
	if sourcePath == "" || strings.ToLower(filepath.Ext(sourcePath)) != ".json" {
		return StartImportUsersFromJSONOutput{}, ErrInvalidImportSource
	}

	jobID, err := uc.importJobRepo.Enqueue(ctx, sourcePath)
	if err != nil {
		return StartImportUsersFromJSONOutput{}, fmt.Errorf("%w: %v", ErrEnqueueImportJob, err)
	}

	return StartImportUsersFromJSONOutput{
		JobID:  jobID,
		Status: "queued",
	}, nil
}
