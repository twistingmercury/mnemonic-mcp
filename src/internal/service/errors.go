package service

import "errors"

var (
	// ErrNotFound indicates the requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrServiceUnavailable indicates a backend dependency is unreachable.
	ErrServiceUnavailable = errors.New("service unavailable")
)
