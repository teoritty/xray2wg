package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("not_found")
	ErrConflict        = errors.New("conflict")
	ErrNodeInUse       = errors.New("manual node is in use by a tunnel; change the tunnel outbound first")
	ErrValidation      = errors.New("validation")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrRateLimited     = errors.New("rate_limited")
	ErrBootstrapDone   = errors.New("bootstrap_done")
	ErrInvalidPassword = errors.New("invalid_password")
)

// ErrorCode identifies a stable API error code.
type ErrorCode string

const (
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	CodeForbidden     ErrorCode = "FORBIDDEN"
	CodeValidation    ErrorCode = "VALIDATION"
	CodeConflict      ErrorCode = "CONFLICT"
	CodeNodeInUse     ErrorCode = "NODE_IN_USE"
	CodeRateLimited   ErrorCode = "RATE_LIMIT"
	CodeBootstrapDone ErrorCode = "BOOTSTRAP_DONE"
	CodeInternal      ErrorCode = "INTERNAL"
)

// Error is a domain-level error with a client-safe message and optional wrapped cause.
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause for errors.Is / errors.As chains.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewNotFoundError builds a NOT_FOUND error with a client-visible message.
func NewNotFoundError(msg string) *Error {
	return &Error{Code: CodeNotFound, Message: msg}
}

// NewUnauthorizedError builds an UNAUTHORIZED error with a client-visible message.
func NewUnauthorizedError(msg string) *Error {
	return &Error{Code: CodeUnauthorized, Message: msg}
}

// NewValidationError builds a VALIDATION error with a client-visible message.
func NewValidationError(msg string) *Error {
	return &Error{Code: CodeValidation, Message: msg}
}

// NewConflictError builds a CONFLICT error with a client-visible message.
func NewConflictError(msg string) *Error {
	return &Error{Code: CodeConflict, Message: msg}
}

// NewForbiddenError builds a FORBIDDEN error with a client-visible message.
func NewForbiddenError(msg string) *Error {
	return &Error{Code: CodeForbidden, Message: msg}
}
