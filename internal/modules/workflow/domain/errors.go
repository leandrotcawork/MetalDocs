package domain

import "errors"

var (
	ErrInvalidCommand    = errors.New("invalid workflow command")
	ErrInvalidTransition = errors.New("invalid workflow transition")
)
