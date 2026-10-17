// Package domain holds the business entities and rules. It imports nothing
// from Fiber or PostgreSQL, so every rule here can be tested on its own.
package domain

import "errors"

// Errors the rest of the program can check with errors.Is.
var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// ValidationError means the client sent bad input. The message is safe to
// show to the user.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

// ConflictError means the data clashes with something that already exists,
// for example an email that is already registered.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string { return e.Message }
