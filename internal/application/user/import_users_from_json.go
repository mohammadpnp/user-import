package user

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
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

type startImportUsersFromJSON struct {
	importJobRepo domain.ImportJobRepository
}

func NewStartImportUsersFromJSON(importJobRepo domain.ImportJobRepository) StartImportUsersFromJSON {
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
