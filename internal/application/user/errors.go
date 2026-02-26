package user

import "errors"

var (
	ErrInvalidImportSource = errors.New("invalid import source")
	ErrEnqueueImportJob    = errors.New("failed to enqueue import job")
)
