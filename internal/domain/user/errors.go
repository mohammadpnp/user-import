package user

import "errors"

var (
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidAddress = errors.New("invalid address")
)
