package user

import "errors"

var (
	ErrInvalidImportSource = errors.New("invalid import source")
	ErrEnqueueImportJob    = errors.New("failed to enqueue import job")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrUserNotFound        = errors.New("user not found")
	ErrGetUserByID         = errors.New("failed to get user by id")
)
