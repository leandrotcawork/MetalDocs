package domain

import "errors"

var (
	ErrUserNotFound = errors.New("iam user not found")
	ErrUserInactive = errors.New("iam user inactive")
)
